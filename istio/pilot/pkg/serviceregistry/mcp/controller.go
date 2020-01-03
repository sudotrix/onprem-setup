// Copyright 2018 Istio Authors
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

package mcp

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"

	"istio.io/pkg/ledger"
	"istio.io/pkg/log"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/serviceregistry/kube"
	"istio.io/istio/pkg/config/schema"
	"istio.io/istio/pkg/config/schemas"
	"istio.io/istio/pkg/mcp/sink"
)

var (
	errUnsupported = errors.New("this operation is not supported by mcp controller")
)

const ledgerLogf = "error tracking pilot config versions for mcp distribution: %v"

// Controller is a combined interface for ConfigStoreCache
// and MCP Updater
type Controller interface {
	model.ConfigStoreCache
	sink.Updater
}

// Options stores the configurable attributes of a Control
type Options struct {
	DomainSuffix string
	XDSUpdater   model.XDSUpdater
	ConfigLedger ledger.Ledger
}

// controller is a temporary storage for the changes received
// via MCP server
type controller struct {
	configStoreMu sync.RWMutex
	// keys [type][namespace][name]
	configStore             map[string]map[string]map[string]*model.Config
	descriptorsByCollection map[string]schema.Instance
	options                 *Options
	eventHandlers           map[string][]func(model.Config, model.Config, model.Event)
	ledger                  ledger.Ledger
	supportedConfig         schema.Set

	syncedMu sync.Mutex
	synced   map[string]bool
}

// NewController provides a new Controller controller
func NewController(options *Options) Controller {
	var supportedCfg schema.Set
	descriptorsByMessageName := make(map[string]schema.Instance, len(schemas.Istio))
	synced := make(map[string]bool)
	for _, descriptor := range schemas.Istio {
		// don't register duplicate descriptors for the same collection
		if _, ok := descriptorsByMessageName[descriptor.Collection]; !ok {
			descriptorsByMessageName[descriptor.Collection] = descriptor
			synced[descriptor.Collection] = false
		}
		supportedCfg = append(supportedCfg, descriptor)
	}

	return &controller{
		configStore:             make(map[string]map[string]map[string]*model.Config),
		options:                 options,
		descriptorsByCollection: descriptorsByMessageName,
		eventHandlers:           make(map[string][]func(model.Config, model.Config, model.Event)),
		synced:                  synced,
		ledger:                  options.ConfigLedger,
		supportedConfig:         supportedCfg,
	}
}

// ConfigDescriptor returns all the ConfigDescriptors that this
// controller is responsible for
func (c *controller) ConfigDescriptor() schema.Set {
	return c.supportedConfig
}

// List returns all the config that is stored by type and namespace
// if namespace is empty string it returns config for all the namespaces
func (c *controller) List(typ, namespace string) (out []model.Config, err error) {
	_, ok := c.ConfigDescriptor().GetByType(typ)
	if !ok {
		return nil, fmt.Errorf("list unknown type %s", typ)
	}
	c.configStoreMu.Lock()
	byType, ok := c.configStore[typ]
	c.configStoreMu.Unlock()
	if !ok {
		return nil, nil
	}

	if namespace == "" {
		// ByType does not need locking since
		// we replace the entire sub-map
		for _, byNamespace := range byType {
			for _, config := range byNamespace {
				out = append(out, *config)
			}
		}
		return out, nil
	}

	for _, config := range byType[namespace] {
		out = append(out, *config)
	}
	return out, nil
}

// Apply receives changes from MCP server and creates the
// corresponding config
func (c *controller) Apply(change *sink.Change) error {
	descriptor, ok := c.descriptorsByCollection[change.Collection]
	if !ok || change.Collection == schemas.SyntheticServiceEntry.Collection {
		return fmt.Errorf("apply type not supported %s", change.Collection)
	}

	s, valid := c.ConfigDescriptor().GetByType(descriptor.Type)
	if !valid {
		return fmt.Errorf("descriptor type not supported %s", descriptor.Type)
	}

	// innerStore is [namespace][name]
	innerStore := make(map[string]map[string]*model.Config)
	for _, obj := range change.Objects {
		namespace, name := extractNameNamespace(obj.Metadata.Name)

		createTime := time.Now()
		if obj.Metadata.CreateTime != nil {
			var err error
			if createTime, err = types.TimestampFromProto(obj.Metadata.CreateTime); err != nil {
				// Do not return an error, instead discard the resources so that Pilot can process the rest.
				log.Warnf("Discarding incoming MCP resource: invalid resource timestamp (%s/%s): %v", namespace, name, err)
				continue
			}
		}

		conf := &model.Config{
			ConfigMeta: model.ConfigMeta{
				Type:              descriptor.Type,
				Group:             descriptor.Group,
				Version:           descriptor.Version,
				Name:              name,
				Namespace:         namespace,
				ResourceVersion:   obj.Metadata.Version,
				CreationTimestamp: createTime,
				Labels:            obj.Metadata.Labels,
				Annotations:       obj.Metadata.Annotations,
				Domain:            c.options.DomainSuffix,
			},
			Spec: obj.Body,
		}

		if err := s.Validate(conf.Name, conf.Namespace, conf.Spec); err != nil {
			// Do not return an error, instead discard the resources so that Pilot can process the rest.
			log.Warnf("Discarding incoming MCP resource: validation failed (%s/%s): %v", conf.Namespace, conf.Name, err)
			continue
		}

		namedConfig, ok := innerStore[conf.Namespace]
		if ok {
			namedConfig[conf.Name] = conf
		} else {
			innerStore[conf.Namespace] = map[string]*model.Config{
				conf.Name: conf,
			}
		}

		_, err := c.ledger.Put(conf.Key(), obj.Metadata.Version)
		if err != nil {
			log.Warnf(ledgerLogf, err)
		}
	}
	for _, removed := range change.Removed {
		err := c.ledger.Delete(kube.KeyFunc(change.Collection, removed))
		if err != nil {
			log.Warnf(ledgerLogf, err)
		}
	}

	var prevStore map[string]map[string]*model.Config

	c.configStoreMu.Lock()
	prevStore = c.configStore[descriptor.Type]
	c.configStore[descriptor.Type] = innerStore
	c.configStoreMu.Unlock()
	c.sync(change.Collection)

	if descriptor.Type == schemas.ServiceEntry.Type {
		c.serviceEntryEvents(innerStore, prevStore)
	} else if c.options.XDSUpdater != nil {
		c.options.XDSUpdater.ConfigUpdate(&model.PushRequest{
			Full:               true,
			ConfigTypesUpdated: map[string]struct{}{descriptor.Type: {}},
		})
	}
	return nil
}

// HasSynced returns true if the first batch of items has been popped
func (c *controller) HasSynced() bool {
	var notReady []string

	c.syncedMu.Lock()
	for messageName, synced := range c.synced {
		if !synced {
			notReady = append(notReady, messageName)
		}
	}
	c.syncedMu.Unlock()

	if len(notReady) > 0 {
		return false
	}

	log.Infof("Configuration synced")
	return true
}

// RegisterEventHandler registers a handler using the type as a key
func (c *controller) RegisterEventHandler(typ string, handler func(model.Config, model.Config, model.Event)) {
	c.eventHandlers[typ] = append(c.eventHandlers[typ], handler)
}

func (c *controller) Version() string {
	return c.ledger.RootHash()
}

func (c *controller) GetResourceAtVersion(version string, key string) (resourceVersion string, err error) {
	return c.ledger.GetPreviousValue(version, key)
}

// Run is not implemented
func (c *controller) Run(<-chan struct{}) {
	log.Warnf("Run: %s", errUnsupported)
}

// Get is not implemented
func (c *controller) Get(_, _, _ string) *model.Config {
	log.Warnf("get %s", errUnsupported)
	return nil
}

// Update is not implemented
func (c *controller) Update(model.Config) (newRevision string, err error) {
	log.Warnf("update %s", errUnsupported)
	return "", errUnsupported
}

// Create is not implemented
func (c *controller) Create(model.Config) (revision string, err error) {
	log.Warnf("create %s", errUnsupported)
	return "", errUnsupported
}

// Delete is not implemented
func (c *controller) Delete(_, _, _ string) error {
	return errUnsupported
}

func (c *controller) sync(collection string) {
	c.syncedMu.Lock()
	c.synced[collection] = true
	c.syncedMu.Unlock()
}

func (c *controller) serviceEntryEvents(currentStore, prevStore map[string]map[string]*model.Config) {
	dispatch := func(model model.Config, event model.Event) {}
	if handlers, ok := c.eventHandlers[schemas.ServiceEntry.Type]; ok {
		dispatch = func(config model.Config, event model.Event) {
			log.Debugf("MCP event dispatch: key=%v event=%v", config.Key(), event.String())
			for _, handler := range handlers {
				handler(model.Config{}, config, event)
			}
		}
	}

	// add/update
	for namespace, byName := range currentStore {
		for name, config := range byName {
			if prevByNamespace, ok := prevStore[namespace]; ok {
				if prevConfig, ok := prevByNamespace[name]; ok {
					if config.ResourceVersion != prevConfig.ResourceVersion {
						dispatch(*config, model.EventUpdate)
					}
				} else {
					dispatch(*config, model.EventAdd)
				}
			} else {
				dispatch(*config, model.EventAdd)
			}
		}
	}

	// delete
	for namespace, prevByName := range prevStore {
		for name, prevConfig := range prevByName {
			if _, ok := currentStore[namespace][name]; !ok {
				dispatch(*prevConfig, model.EventDelete)
			}
		}
	}
}

func extractNameNamespace(metadataName string) (string, string) {
	segments := strings.Split(metadataName, "/")
	if len(segments) == 2 {
		return segments[0], segments[1]
	}
	return "", segments[0]
}
