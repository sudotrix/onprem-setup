// Copyright Istio Authors
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

package mesh

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/davecgh/go-spew/spew"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/pkg/filewatcher"
	"istio.io/pkg/log"
)

// NetworksHolder is a holder of a mesh networks configuration.
type NetworksHolder interface {
	Networks() *meshconfig.MeshNetworks
}

// NetworkWatcher watches changes to the mesh networks config.
type NetworksWatcher interface {
	NetworksHolder

	AddNetworksHandler(func())
}

var _ NetworksWatcher = &networksWatcher{}

type networksWatcher struct {
	mutex    sync.Mutex
	handlers []func()
	networks *meshconfig.MeshNetworks
}

// NewFixedNetworksWatcher creates a new NetworksWatcher that always returns the given config.
// It will never fire any events, since the config never changes.
func NewFixedNetworksWatcher(networks *meshconfig.MeshNetworks) NetworksWatcher {
	return &networksWatcher{
		networks: networks,
	}
}

// NewNetworksWatcher creates a new watcher for changes to the given networks config file.
func NewNetworksWatcher(fileWatcher filewatcher.FileWatcher, filename string) (NetworksWatcher, error) {
	meshNetworks, err := ReadMeshNetworks(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read mesh networks configuration from %q: %v", filename, err)
	}

	log.Infof("mesh networks configuration %s", spew.Sdump(meshNetworks))
	ResolveHostsInNetworksConfig(meshNetworks)
	log.Infof("mesh networks configuration post-resolution %s", spew.Sdump(meshNetworks))

	w := &networksWatcher{
		networks: meshNetworks,
	}

	// Watch the networks config file for changes and reload if it got modified
	addFileWatcher(fileWatcher, filename, func() {
		// Reload the config file
		meshNetworks, err := ReadMeshNetworks(filename)
		if err != nil {
			log.Warnf("failed to read mesh networks configuration from %q", filename)
			return
		}

		var handlers []func()

		w.mutex.Lock()
		if !reflect.DeepEqual(meshNetworks, w.networks) {
			log.Infof("mesh networks configuration file updated to: %s", spew.Sdump(meshNetworks))
			ResolveHostsInNetworksConfig(meshNetworks)
			log.Infof("mesh networks configuration post-resolution %s", spew.Sdump(meshNetworks))

			// Store the new config.
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&w.networks)), unsafe.Pointer(meshNetworks))
			handlers = append([]func(){}, w.handlers...)
		}
		w.mutex.Unlock()

		// Notify the handlers of the change.
		for _, h := range handlers {
			h()
		}
	})
	return w, nil
}

// Config returns the latest network configuration for the mesh.
func (w *networksWatcher) Networks() *meshconfig.MeshNetworks {
	return (*meshconfig.MeshNetworks)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&w.networks))))
}

// AddMeshHandler registers a callback handler for changes to the mesh network config.
func (w *networksWatcher) AddNetworksHandler(h func()) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.handlers = append(w.handlers, h)
}
