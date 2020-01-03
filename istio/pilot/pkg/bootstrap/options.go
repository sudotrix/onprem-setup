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

package bootstrap

import (
	"time"

	"istio.io/pkg/env"

	"istio.io/istio/pkg/config/constants"

	meshconfig "istio.io/api/mesh/v1alpha1"
	kubecontroller "istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	istiokeepalive "istio.io/istio/pkg/keepalive"
	"istio.io/pkg/ctrlz"
)

// MeshArgs provide configuration options for the mesh. If ConfigFile is provided, an attempt will be made to
// load the mesh from the file. Otherwise, a default mesh will be used with optional overrides.
type MeshArgs struct {
	ConfigFile string
	// Used for test
	MixerAddress string
}

// ConfigArgs provide configuration options for the configuration controller. If FileDir is set, that directory will
// be monitored for CRD yaml files and will update the controller as those files change (This is used for testing
// purposes). Otherwise, a CRD client is created based on the configuration.
type ConfigArgs struct {
	ControllerOptions          kubecontroller.Options
	ClusterRegistriesNamespace string
	KubeConfig                 string
	FileDir                    string

	// DistributionTracking control
	DistributionCacheRetention time.Duration

	DisableInstallCRDs bool

	// DistributionTracking control
	DistributionTrackingEnabled bool
}

// ConsulArgs provides configuration for the Consul service registry.
type ConsulArgs struct {
	ServerURL string
}

// ServiceArgs provides the composite configuration for all service registries in the system.
type ServiceArgs struct {
	Registries []string
	Consul     ConsulArgs
}

// PilotArgs provides all of the configuration parameters for the Pilot discovery service.
type PilotArgs struct {
	DiscoveryOptions         DiscoveryServiceOptions
	InjectionOptions         InjectionOptions
	Namespace                string
	Mesh                     MeshArgs
	Config                   ConfigArgs
	Service                  ServiceArgs
	MeshConfig               *meshconfig.MeshConfig
	NetworksConfigFile       string
	CtrlZOptions             *ctrlz.Options
	Plugins                  []string
	MCPMaxMessageSize        int
	MCPInitialWindowSize     int
	MCPInitialConnWindowSize int
	KeepaliveOptions         *istiokeepalive.Options
	// ForceStop is set as true when used for testing to make the server stop quickly
	ForceStop bool
	BasePort  int
}

// DiscoveryServiceOptions contains options for create a new discovery
// service instance.
type DiscoveryServiceOptions struct {
	// The listening address for HTTP. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	HTTPAddr string

	// The listening address for GRPC. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	GrpcAddr string

	// The listening address for secure GRPC. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	// "" means disabling secure GRPC, used in test.
	SecureGrpcAddr string

	// The listening address for secure GRPC with DNS-based certificates. Default is :15012, if certificates are available.
	// Will not start otherwise.
	SecureGrpcDNSAddr string

	// The listening address for the monitoring port. If the port in the address is empty or "0" (as in "127.0.0.1:" or "[::1]:0")
	// a port number is automatically chosen.
	MonitoringAddr string

	EnableProfiling bool
}

type InjectionOptions struct {
	InjectionDirectory string
	Port               int
}

var podNamespaceVar = env.RegisterStringVar("POD_NAMESPACE", "", "")

// Apply default value to PilotArgs
func (p *PilotArgs) Default() {
	// If the namespace isn't set, try looking it up from the environment.
	if p.Namespace == "" {
		p.Namespace = podNamespaceVar.Get()
	}

	if p.KeepaliveOptions == nil {
		p.KeepaliveOptions = istiokeepalive.DefaultOption()
	}
	if p.Config.ClusterRegistriesNamespace == "" {
		if p.Namespace != "" {
			p.Config.ClusterRegistriesNamespace = p.Namespace
		} else {
			p.Config.ClusterRegistriesNamespace = constants.IstioSystemNamespace
		}
	}
	if p.BasePort == 0 {
		p.BasePort = 15000
	}
}
