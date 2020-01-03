// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serviceentry

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/types"

	"istio.io/api/annotation"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/pkg/ledger"
	"istio.io/pkg/log"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/schema"
	"istio.io/istio/pkg/config/schemas"
	"istio.io/istio/pkg/mcp/sink"
)

var (
	errUnsupported      = errors.New("this operation is not supported by mcp controller")
	endpointKey         = annotation.AlphaNetworkingEndpointsVersion.Name
	serviceKey          = annotation.AlphaNetworkingServiceVersion.Name
	notReadyEndpointkey = annotation.AlphaNetworkingNotReadyEndpoints.Name
)

// Controller is a combined interface for ConfigStoreCache
// and MCP Updater
type Controller interface {
	model.ConfigStoreCache
	sink.Updater
}

// Options stores the configurable attributes of a Control
type Options struct {
	ClusterID    string
	DomainSuffix string
	XDSUpdater   model.XDSUpdater
	ConfigLedger ledger.Ledger
}

// SyntheticServiceEntryController is a temporary storage for the changes received
// via MCP server
type SyntheticServiceEntryController struct {
	configStoreMu sync.RWMutex
	// keys [namespace][name]
	configStore  map[string]map[string]*model.Config
	eventHandler func(model.Config, model.Config, model.Event)
	synced       uint32
	*Options
}

// NewSyntheticServiceEntryController provides a new incremental Controller controller
func NewSyntheticServiceEntryController(options *Options) Controller {
	return &SyntheticServiceEntryController{
		configStore: make(map[string]map[string]*model.Config),
		Options:     options,
	}
}

// ConfigDescriptor returns all the ConfigDescriptors that this
// controller is responsible for
func (c *SyntheticServiceEntryController) ConfigDescriptor() schema.Set {
	return schema.Set{schemas.SyntheticServiceEntry}
}

// List returns all the SyntheticServiceEntries that is stored by type and namespace
// if namespace is empty string it returns config for all the namespaces
func (c *SyntheticServiceEntryController) List(typ, namespace string) (out []model.Config, err error) {
	if typ != schemas.SyntheticServiceEntry.Type {
		return nil, fmt.Errorf("list unknown type %s", typ)
	}

	c.configStoreMu.Lock()
	defer c.configStoreMu.Unlock()

	// retrieve all configs
	if namespace == "" {
		for _, allNamespaces := range c.configStore {
			for _, config := range allNamespaces {
				out = append(out, *config)
			}
		}
		return out, nil
	}

	byNamespace, ok := c.configStore[namespace]
	if !ok {
		return nil, nil
	}

	for _, config := range byNamespace {
		out = append(out, *config)
	}
	return out, nil
}

// Apply receives changes from MCP server and creates the
// corresponding config
func (c *SyntheticServiceEntryController) Apply(change *sink.Change) error {
	if change.Collection != schemas.SyntheticServiceEntry.Collection {
		return fmt.Errorf("apply: type not supported %s", change.Collection)
	}

	defer atomic.AddUint32(&c.synced, 1)

	if len(change.Objects) == 0 {
		return nil
	}

	if change.Incremental {
		// removed first
		c.removeConfig(change.Removed)
		c.incrementalUpdate(change.Objects)
	} else {
		c.configStoreUpdate(change.Objects)
	}

	return nil
}

// HasSynced returns true if the first batch of items has been popped
func (c *SyntheticServiceEntryController) HasSynced() bool {
	return atomic.LoadUint32(&c.synced) != 0
}

func (c *SyntheticServiceEntryController) dispatch(config model.Config, event model.Event) {
	if c.eventHandler != nil {
		c.eventHandler(model.Config{}, config, event)
	}
}

// RegisterEventHandler registers a handler using the type as a key
func (c *SyntheticServiceEntryController) RegisterEventHandler(_ string, handler func(model.Config, model.Config, model.Event)) {
	// TODO: investigate why it is called more than one
	if c.eventHandler == nil {
		c.eventHandler = handler
	}
}

// Version is not implemented
func (c *SyntheticServiceEntryController) Version() string {
	log.Warnf("version: %s", errUnsupported)
	return ""
}

// GetResourceAtVersion is not implemented
func (c *SyntheticServiceEntryController) GetResourceAtVersion(string, string) (resourceVersion string, err error) {
	log.Warnf("getResourceAtVersion: %s", errUnsupported)
	return "", nil
}

// Run is not implemented
func (c *SyntheticServiceEntryController) Run(<-chan struct{}) {
	log.Warnf("run: %s", errUnsupported)
}

// Get is not implemented
func (c *SyntheticServiceEntryController) Get(string, string, string) *model.Config {
	log.Warnf("get %s", errUnsupported)
	return nil
}

// Update is not implemented
func (c *SyntheticServiceEntryController) Update(model.Config) (newRevision string, err error) {
	log.Warnf("update %s", errUnsupported)
	return "", errUnsupported
}

// Create is not implemented
func (c *SyntheticServiceEntryController) Create(model.Config) (revision string, err error) {
	log.Warnf("create %s", errUnsupported)
	return "", errUnsupported
}

// Delete is not implemented
func (c *SyntheticServiceEntryController) Delete(string, string, string) error {
	log.Warnf("delete %s", errUnsupported)
	return errUnsupported
}

func (c *SyntheticServiceEntryController) removeConfig(configName []string) {
	if len(configName) == 0 {
		return
	}
	c.configStoreMu.Lock()
	defer c.configStoreMu.Unlock()

	for _, fullName := range configName {
		namespace, name := extractNameNamespace(fullName)
		if byNamespace, ok := c.configStore[namespace]; ok {
			if conf, ok := byNamespace[name]; ok {
				delete(byNamespace, conf.Name)
				c.dispatch(*conf, model.EventDelete)
			}
			// clear parent map also
			if len(byNamespace) == 0 {
				delete(byNamespace, namespace)
			}
		}
	}
}

func (c *SyntheticServiceEntryController) convertToConfig(obj *sink.Object) (conf *model.Config, err error) {
	namespace, name := extractNameNamespace(obj.Metadata.Name)

	createTime := time.Now()
	if obj.Metadata.CreateTime != nil {
		if createTime, err = types.TimestampFromProto(obj.Metadata.CreateTime); err != nil {
			log.Warnf("Discarding incoming MCP resource: invalid resource timestamp (%s/%s): %v", namespace, name, err)
			return nil, err
		}
	}

	conf = &model.Config{
		ConfigMeta: model.ConfigMeta{
			Type:              schemas.SyntheticServiceEntry.Type,
			Group:             schemas.SyntheticServiceEntry.Group,
			Version:           schemas.SyntheticServiceEntry.Version,
			Name:              name,
			Namespace:         namespace,
			ResourceVersion:   obj.Metadata.Version,
			CreationTimestamp: createTime,
			Labels:            obj.Metadata.Labels,
			Annotations:       obj.Metadata.Annotations,
			Domain:            c.DomainSuffix,
		},
		Spec: obj.Body,
	}

	s, _ := c.ConfigDescriptor().GetByType(schemas.SyntheticServiceEntry.Type)
	if err = s.Validate(conf.Name, conf.Namespace, conf.Spec); err != nil {
		log.Warnf("Discarding incoming MCP resource: validation failed (%s/%s): %v", conf.Namespace, conf.Name, err)
		return nil, err
	}
	return conf, nil

}

func (c *SyntheticServiceEntryController) configStoreUpdate(resources []*sink.Object) {
	svcChangeByNamespace := make(map[string]struct{})
	configs := make(map[string]map[string]*model.Config)
	for _, obj := range resources {
		conf, err := c.convertToConfig(obj)
		if err != nil {
			continue
		}

		namedConf, ok := configs[conf.Namespace]
		if ok {
			namedConf[conf.Name] = conf
			c.dispatch(*conf, model.EventUpdate)
		} else {
			configs[conf.Namespace] = map[string]*model.Config{
				conf.Name: conf,
			}
			c.dispatch(*conf, model.EventAdd)
		}

		svcChanged := c.isFullUpdateRequired(conf)
		if svcChanged {
			svcChangeByNamespace[conf.Namespace] = struct{}{}
			continue
		}
	}

	c.configStoreMu.Lock()
	c.configStore = configs
	c.configStoreMu.Unlock()

	// TODO: Service change is not triggering full update in the e-e-pilot test. Even endpoint change is not
	// functioning correctly. Currently it is working because on edsUpdate if we set endpoints to 0, we remove
	// the service from EndpointShardsByService and subsequent eds updates trigger a full push. That is being
	// fixed in https://github.com/istio/istio/pull/18574. Need to fix this issue and re-enable conditional
	// full push. For now, any configupdate triggers a full push much like service entries.
	if c.XDSUpdater != nil {
		c.XDSUpdater.ConfigUpdate(&model.PushRequest{
			Full:               true,
			ConfigTypesUpdated: map[string]struct{}{schemas.SyntheticServiceEntry.Type: {}},
			NamespacesUpdated:  svcChangeByNamespace,
		})
	}

}

func (c *SyntheticServiceEntryController) incrementalUpdate(resources []*sink.Object) {
	svcChangeByNamespace := make(map[string]struct{})
	for _, obj := range resources {
		conf, err := c.convertToConfig(obj)
		if err != nil {
			continue
		}

		// should we check resource version??
		svcChanged := c.isFullUpdateRequired(conf)
		var oldEpVersion string
		c.configStoreMu.Lock()
		namedConf, ok := c.configStore[conf.Namespace]
		if ok {
			if namedConf[conf.Name] != nil {
				oldEpVersion = version(namedConf[conf.Name].Annotations, endpointKey)
			}
			namedConf[conf.Name] = conf
			c.dispatch(*conf, model.EventUpdate)
		} else {
			c.configStore[conf.Namespace] = map[string]*model.Config{
				conf.Name: conf,
			}
			c.dispatch(*conf, model.EventAdd)
		}
		c.configStoreMu.Unlock()

		if svcChanged {
			svcChangeByNamespace[conf.Namespace] = struct{}{}
			continue
		}

		newEpVersion := version(conf.Annotations, endpointKey)
		if oldEpVersion != newEpVersion {
			if err := c.edsUpdate(conf); err != nil {
				log.Warnf("edsUpdate: %v", err)
			}
		}
	}
	if len(svcChangeByNamespace) != 0 {
		if c.XDSUpdater != nil {
			c.XDSUpdater.ConfigUpdate(&model.PushRequest{
				Full:               true,
				ConfigTypesUpdated: map[string]struct{}{schemas.SyntheticServiceEntry.Type: {}},
				NamespacesUpdated:  svcChangeByNamespace,
			})
		}
	}
}

func (c *SyntheticServiceEntryController) edsUpdate(config *model.Config) error {
	se, ok := config.Spec.(*networking.ServiceEntry)
	if !ok {
		return errors.New("edsUpdate: wrong type")
	}
	istioEndpoints := convertEndpoints(se, config.Name, config.Namespace)
	hostname := hostName(config.Name, config.Namespace, c.DomainSuffix)
	return c.XDSUpdater.EDSUpdate(c.ClusterID, hostname, config.Namespace, istioEndpoints)
}

func (c *SyntheticServiceEntryController) configExist(namespace, name string) *model.Config {
	c.configStoreMu.Lock()
	defer c.configStoreMu.Unlock()
	configs, ok := c.configStore[namespace]
	if !ok {
		return nil
	}
	conf, ok := configs[name]
	if !ok {
		return nil
	}
	return conf
}

func (c *SyntheticServiceEntryController) isFullUpdateRequired(newConf *model.Config) bool {
	oldConf := c.configExist(newConf.Namespace, newConf.Name)
	if oldConf == nil {
		// if config does not exist send it down configUpdate path
		return true
	}
	oldVersion := version(oldConf.Annotations, serviceKey)
	newVersion := version(newConf.Annotations, serviceKey)
	return oldVersion != newVersion
}

func convertEndpoints(se *networking.ServiceEntry, cfgName, ns string) (endpoints []*model.IstioEndpoint) {
	for _, ep := range se.Endpoints {
		for portName, port := range ep.Ports {
			ep := &model.IstioEndpoint{
				Address:         ep.Address,
				EndpointPort:    port,
				ServicePortName: portName,
				Labels:          ep.Labels,
				UID:             cfgName + "." + ns,
				Network:         ep.Network,
				Locality:        ep.Locality,
				LbWeight:        ep.Weight,
				// ServiceAccount:??
			}
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

func version(anno map[string]string, key string) string {
	if version, ok := anno[key]; ok {
		return version
	}
	return ""
}

func hostName(name, namespace, domainSuffix string) string {
	return name + "." + namespace + ".svc." + domainSuffix
}

func extractNameNamespace(metadataName string) (string, string) {
	segments := strings.Split(metadataName, "/")
	if len(segments) == 2 {
		return segments[0], segments[1]
	}
	return "", segments[0]
}
