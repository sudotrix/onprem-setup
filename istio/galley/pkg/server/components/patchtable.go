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

package components

import (
	"io/ioutil"
	"net"

	"istio.io/istio/galley/pkg/config/event"
	"istio.io/istio/galley/pkg/config/meshcfg"
	"istio.io/istio/galley/pkg/config/processor"
	"istio.io/istio/galley/pkg/config/source/kube"
	"istio.io/istio/galley/pkg/config/source/kube/fs"
	"istio.io/istio/pkg/mcp/monitoring"
	"istio.io/pkg/filewatcher"
)

// The patch table for external dependencies for code in components.
var (
	netListen         = net.Listen
	newInterfaces     = kube.NewInterfacesFromConfigFile
	mcpMetricReporter = func(prefix string) monitoring.Reporter { return monitoring.NewStatsContext(prefix) }
	newFileWatcher    = filewatcher.NewWatcher
	readFile          = ioutil.ReadFile

	meshcfgNewFS        = func(path string) (event.Source, error) { return meshcfg.NewFS(path) }
	processorInitialize = processor.Initialize
	fsNew               = fs.New
)

func resetPatchTable() {
	netListen = net.Listen
	newInterfaces = kube.NewInterfacesFromConfigFile
	mcpMetricReporter = func(prefix string) monitoring.Reporter { return monitoring.NewStatsContext(prefix) }
	newFileWatcher = filewatcher.NewWatcher
	readFile = ioutil.ReadFile

	meshcfgNewFS = func(path string) (event.Source, error) { return meshcfg.NewFS(path) }
	processorInitialize = processor.Initialize
	fsNew = fs.New
}
