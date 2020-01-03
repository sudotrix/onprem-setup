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
package deployment

import (
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	k8s_labels "k8s.io/apimachinery/pkg/labels"

	"istio.io/api/annotation"

	"istio.io/istio/galley/pkg/config/analysis"
	"istio.io/istio/galley/pkg/config/analysis/analyzers/injection"
	"istio.io/istio/galley/pkg/config/analysis/msg"
	"istio.io/istio/galley/pkg/config/meta/metadata"
	"istio.io/istio/galley/pkg/config/meta/schema/collection"
	"istio.io/istio/galley/pkg/config/resource"
)

type ServiceAssociationAnalyzer struct{}

var _ analysis.Analyzer = &ServiceAssociationAnalyzer{}

type PortMap map[int32]ProtocolMap
type ProtocolMap map[core_v1.Protocol]ServiceNames
type ServiceNames []string
type ServiceSpecWithName struct {
	Name string
	Spec *core_v1.ServiceSpec
}

func (s *ServiceAssociationAnalyzer) Metadata() analysis.Metadata {
	return analysis.Metadata{
		Name:        "deployment.MultiServiceAnalyzer",
		Description: "Checks association between services and pods",
		Inputs: collection.Names{
			metadata.K8SCoreV1Services,
			metadata.K8SAppsV1Deployments,
			metadata.K8SCoreV1Namespaces,
		},
	}
}
func (s *ServiceAssociationAnalyzer) Analyze(c analysis.Context) {
	c.ForEach(metadata.K8SAppsV1Deployments, func(r *resource.Instance) bool {
		if inMesh(r, c) {
			s.analyzeDeployment(r, c)
		}
		return true
	})
}

// analyzeDeployment analyzes the specific service mesh deployment
func (s *ServiceAssociationAnalyzer) analyzeDeployment(r *resource.Instance, c analysis.Context) {
	d := r.Message.(*apps_v1.Deployment)

	// Find matching services with resulting pod from deployment
	matchingSvcs := s.findMatchingServices(d, c)

	// If there isn't any matching service, generate message: At least one service is needed.
	if len(matchingSvcs) == 0 {
		c.Report(metadata.K8SAppsV1Deployments, msg.NewDeploymentRequiresServiceAssociated(r, d.Name))
		return
	}

	// Generate a port map from the matching services.
	// It creates a structure that will allow us to detect
	// if there are different protocols for the same port.
	portMap := servicePortMap(matchingSvcs)

	// Determining which ports use more than one protocol.
	for port := range portMap {
		// In case there are two protocols using same port number, generate a message
		protMap := portMap[port]
		if len(protMap) > 1 {
			// Collect names from both protocols
			svcNames := make(ServiceNames, 0)
			for protocol := range protMap {
				svcNames = append(svcNames, protMap[protocol]...)
			}

			// Reporting the message for the deployment, port and conflicting services.
			c.Report(metadata.K8SAppsV1Deployments, msg.NewDeploymentAssociatedToMultipleServices(r, d.Name, port, svcNames))
		}
	}
}

// findMatchingServices returns an slice of Services that matches with deployment's pods.
func (s *ServiceAssociationAnalyzer) findMatchingServices(d *apps_v1.Deployment, c analysis.Context) []ServiceSpecWithName {
	matchingSvcs := make([]ServiceSpecWithName, 0)

	c.ForEach(metadata.K8SCoreV1Services, func(r *resource.Instance) bool {
		s := r.Message.(*core_v1.ServiceSpec)

		sSelector := k8s_labels.SelectorFromSet(s.Selector)
		pLabels := k8s_labels.Set(d.Spec.Template.Labels)
		if sSelector.Matches(pLabels) {
			matchingSvcs = append(matchingSvcs, ServiceSpecWithName{r.Metadata.FullName.String(), s})
		}

		return true
	})

	return matchingSvcs
}

// servicePortMap build a map of ports and protocols for each Service. e.g. m[80]["TCP"] -> svcA, svcB, svcC
func servicePortMap(svcs []ServiceSpecWithName) PortMap {
	portMap := PortMap{}

	for _, swn := range svcs {
		svc := swn.Spec
		for _, sPort := range svc.Ports {
			// If it is the first occurrence of this port, create a ProtocolMap
			if _, ok := portMap[sPort.Port]; !ok {
				portMap[sPort.Port] = ProtocolMap{}
			}

			// Default protocol is TCP
			protocol := sPort.Protocol
			if protocol == "" {
				protocol = core_v1.ProtocolTCP
			}

			// Appending the service information for the Port/Protocol combination
			portMap[sPort.Port][protocol] = append(portMap[sPort.Port][protocol], swn.Name)
		}
	}

	return portMap
}

// inMesh returns true if deployment is in the service mesh (has sidecar)
func inMesh(r *resource.Instance, c analysis.Context) bool {
	d := r.Message.(*apps_v1.Deployment)

	// If Pod has annotation, return the injection annotation value
	if piv, pivok := getPodSidecarInjectionStatus(d); pivok {
		return piv
	}

	// In case the annotation is not present but there is a auto-injection label on the namespace,
	// return the auto-injection label status
	if niv, nivok := getNamesSidecarInjectionStatus(resource.Namespace(d.Namespace), c); nivok {
		return niv
	}

	return false
}

// getPodSidecarInjectionStatus returns two booleans: enabled and ok.
// enabled is true when deployment d PodSpec has either the annotation 'sidecar.istio.io/inject: "true"'
// ok is true when the PodSpec doesn't have the 'sidecar.istio.io/inject' annotation present.
func getPodSidecarInjectionStatus(d *apps_v1.Deployment) (enabled bool, ok bool) {
	v, ok := d.Spec.Template.Annotations[annotation.SidecarInject.Name]
	return v == "true", ok
}

// autoInjectionEnabled returns two booleans: enabled and ok.
// enabled is true when namespace ns has 'istio-injection' label set to 'enabled'
// ok is true when the namespace doesn't have the label 'istio-injection'
func getNamesSidecarInjectionStatus(ns resource.Namespace, c analysis.Context) (enabled bool, ok bool) {
	enabled, ok = false, false

	namespace := c.Find(metadata.K8SCoreV1Namespaces, resource.NewFullName("", resource.LocalName(ns)))
	if namespace != nil {
		enabled, ok = namespace.Metadata.Labels[injection.InjectionLabelName] == injection.InjectionLabelEnableValue, true
	}

	return enabled, ok
}
