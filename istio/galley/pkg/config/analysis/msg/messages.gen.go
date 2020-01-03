// GENERATED FILE -- DO NOT EDIT
//

package msg

import (
	"istio.io/istio/galley/pkg/config/analysis/diag"
	"istio.io/istio/galley/pkg/config/resource"
)

var (
	// InternalError defines a diag.MessageType for message "InternalError".
	// Description: There was an internal error in the toolchain. This is almost always a bug in the implementation.
	InternalError = diag.NewMessageType(diag.Error, "IST0001", "Internal error: %v")

	// Deprecated defines a diag.MessageType for message "Deprecated".
	// Description: A feature that the configuration is depending on is now deprecated.
	Deprecated = diag.NewMessageType(diag.Warning, "IST0002", "Deprecated: %s")

	// ReferencedResourceNotFound defines a diag.MessageType for message "ReferencedResourceNotFound".
	// Description: A resource being referenced does not exist.
	ReferencedResourceNotFound = diag.NewMessageType(diag.Error, "IST0101", "Referenced %s not found: %q")

	// NamespaceNotInjected defines a diag.MessageType for message "NamespaceNotInjected".
	// Description: A namespace is not enabled for Istio injection.
	NamespaceNotInjected = diag.NewMessageType(diag.Warning, "IST0102", "The namespace is not enabled for Istio injection. Run 'kubectl label namespace %s istio-injection=enabled' to enable it, or 'kubectl label namespace %s istio-injection=disabled' to explicitly mark it as not needing injection")

	// PodMissingProxy defines a diag.MessageType for message "PodMissingProxy".
	// Description: A pod is missing the Istio proxy.
	PodMissingProxy = diag.NewMessageType(diag.Warning, "IST0103", "The pod is missing the Istio proxy. This can often be resolved by restarting or redeploying the workload.")

	// GatewayPortNotOnWorkload defines a diag.MessageType for message "GatewayPortNotOnWorkload".
	// Description: Unhandled gateway port
	GatewayPortNotOnWorkload = diag.NewMessageType(diag.Warning, "IST0104", "The gateway refers to a port that is not exposed on the workload (pod selector %s; port %d)")

	// IstioProxyVersionMismatch defines a diag.MessageType for message "IstioProxyVersionMismatch".
	// Description: The version of the Istio proxy running on the pod does not match the version used by the istio injector.
	IstioProxyVersionMismatch = diag.NewMessageType(diag.Warning, "IST0105", "The version of the Istio proxy running on the pod does not match the version used by the istio injector (pod version: %s; injector version: %s). This often happens after upgrading the Istio control-plane and can be fixed by redeploying the pod.")

	// SchemaValidationError defines a diag.MessageType for message "SchemaValidationError".
	// Description: The resource has a schema validation error.
	SchemaValidationError = diag.NewMessageType(diag.Error, "IST0106", "Schema validation error: %v")

	// MisplacedAnnotation defines a diag.MessageType for message "MisplacedAnnotation".
	// Description: An Istio annotation is applied to the wrong kind of resource.
	MisplacedAnnotation = diag.NewMessageType(diag.Warning, "IST0107", "Misplaced annotation: %s can only be applied to %s")

	// UnknownAnnotation defines a diag.MessageType for message "UnknownAnnotation".
	// Description: An Istio annotation is not recognized for any kind of resource
	UnknownAnnotation = diag.NewMessageType(diag.Warning, "IST0108", "Unknown annotation: %s")

	// ConflictingMeshGatewayVirtualServiceHosts defines a diag.MessageType for message "ConflictingMeshGatewayVirtualServiceHosts".
	// Description: Conflicting hosts on VirtualServices associated with mesh gateway
	ConflictingMeshGatewayVirtualServiceHosts = diag.NewMessageType(diag.Error, "IST0109", "The VirtualServices %s associated with mesh gateway define the same host %s which can lead to undefined behavior. This can be fixed by merging the conflicting VirtualServices into a single resource.")

	// ConflictingSidecarWorkloadSelectors defines a diag.MessageType for message "ConflictingSidecarWorkloadSelectors".
	// Description: A Sidecar resource selects the same workloads as another Sidecar resource
	ConflictingSidecarWorkloadSelectors = diag.NewMessageType(diag.Error, "IST0110", "The Sidecars %v in namespace %q select the same workload pod %q, which can lead to undefined behavior.")

	// MultipleSidecarsWithoutWorkloadSelectors defines a diag.MessageType for message "MultipleSidecarsWithoutWorkloadSelectors".
	// Description: More than one sidecar resource in a namespace has no workload selector
	MultipleSidecarsWithoutWorkloadSelectors = diag.NewMessageType(diag.Error, "IST0111", "The Sidecars %v in namespace %q have no workload selector, which can lead to undefined behavior.")

	// VirtualServiceDestinationPortSelectorRequired defines a diag.MessageType for message "VirtualServiceDestinationPortSelectorRequired".
	// Description: A VirtualService routes to a service with more than one port exposed, but does not specify which to use.
	VirtualServiceDestinationPortSelectorRequired = diag.NewMessageType(diag.Error, "IST0112", "This VirtualService routes to a service %q that exposes multiple ports %v. Specifying a port in the destination is required to disambiguate.")

	// MTLSPolicyConflict defines a diag.MessageType for message "MTLSPolicyConflict".
	// Description: A DestinationRule and Policy are in conflict with regards to mTLS.
	MTLSPolicyConflict = diag.NewMessageType(diag.Error, "IST0113", "A DestinationRule and Policy are in conflict with regards to mTLS for host %s. The DestinationRule %q specifies that mTLS must be %t but the Policy object %q specifies %s.")

	// PolicySpecifiesPortNameThatDoesntExist defines a diag.MessageType for message "PolicySpecifiesPortNameThatDoesntExist".
	// Description: A Policy targets a port name that cannot be found.
	PolicySpecifiesPortNameThatDoesntExist = diag.NewMessageType(diag.Warning, "IST0114", "Port name %s could not be found for host %s, which means the Policy won't be enforced.")

	// DestinationRuleUsesMTLSForWorkloadWithoutSidecar defines a diag.MessageType for message "DestinationRuleUsesMTLSForWorkloadWithoutSidecar".
	// Description: A DestinationRule uses mTLS for a workload that has no sidecar.
	DestinationRuleUsesMTLSForWorkloadWithoutSidecar = diag.NewMessageType(diag.Error, "IST0115", "DestinationRule %s uses mTLS for workload %s that has no sidecar. Traffic from workloads with sidecars will fail.")

	// DeploymentAssociatedToMultipleServices defines a diag.MessageType for message "DeploymentAssociatedToMultipleServices".
	// Description: The resulting pods of a service mesh deployment can't be associated with multiple services using the same port but different protocols.
	DeploymentAssociatedToMultipleServices = diag.NewMessageType(diag.Warning, "IST0116", "This deployment is associated with multiple services using port %d but different protocols: %v")

	// DeploymentRequiresServiceAssociated defines a diag.MessageType for message "DeploymentRequiresServiceAssociated".
	// Description: The resulting pods of a service mesh deployment must be associated with at least one service.
	DeploymentRequiresServiceAssociated = diag.NewMessageType(diag.Warning, "IST0117", "No service associated with this deployment. Service mesh deployments must be associated with a service.")

	// PortNameIsNotUnderNamingConvention defines a diag.MessageType for message "PortNameIsNotUnderNamingConvention".
	// Description: Port name is not under naming convention. Protocol detection is applied to the port.
	PortNameIsNotUnderNamingConvention = diag.NewMessageType(diag.Info, "IST0118", "Port name %s (port: %d, targetPort: %s) doesn't follow the naming convention of Istio port.")
)

// NewInternalError returns a new diag.Message based on InternalError.
func NewInternalError(r *resource.Instance, detail string) diag.Message {
	return diag.NewMessage(
		InternalError,
		originOrNil(r),
		detail,
	)
}

// NewDeprecated returns a new diag.Message based on Deprecated.
func NewDeprecated(r *resource.Instance, detail string) diag.Message {
	return diag.NewMessage(
		Deprecated,
		originOrNil(r),
		detail,
	)
}

// NewReferencedResourceNotFound returns a new diag.Message based on ReferencedResourceNotFound.
func NewReferencedResourceNotFound(r *resource.Instance, reftype string, refval string) diag.Message {
	return diag.NewMessage(
		ReferencedResourceNotFound,
		originOrNil(r),
		reftype,
		refval,
	)
}

// NewNamespaceNotInjected returns a new diag.Message based on NamespaceNotInjected.
func NewNamespaceNotInjected(r *resource.Instance, namespace string, namespace2 string) diag.Message {
	return diag.NewMessage(
		NamespaceNotInjected,
		originOrNil(r),
		namespace,
		namespace2,
	)
}

// NewPodMissingProxy returns a new diag.Message based on PodMissingProxy.
func NewPodMissingProxy(r *resource.Instance) diag.Message {
	return diag.NewMessage(
		PodMissingProxy,
		originOrNil(r),
	)
}

// NewGatewayPortNotOnWorkload returns a new diag.Message based on GatewayPortNotOnWorkload.
func NewGatewayPortNotOnWorkload(r *resource.Instance, selector string, port int) diag.Message {
	return diag.NewMessage(
		GatewayPortNotOnWorkload,
		originOrNil(r),
		selector,
		port,
	)
}

// NewIstioProxyVersionMismatch returns a new diag.Message based on IstioProxyVersionMismatch.
func NewIstioProxyVersionMismatch(r *resource.Instance, proxyVersion string, injectionVersion string) diag.Message {
	return diag.NewMessage(
		IstioProxyVersionMismatch,
		originOrNil(r),
		proxyVersion,
		injectionVersion,
	)
}

// NewSchemaValidationError returns a new diag.Message based on SchemaValidationError.
func NewSchemaValidationError(r *resource.Instance, err error) diag.Message {
	return diag.NewMessage(
		SchemaValidationError,
		originOrNil(r),
		err,
	)
}

// NewMisplacedAnnotation returns a new diag.Message based on MisplacedAnnotation.
func NewMisplacedAnnotation(r *resource.Instance, annotation string, kind string) diag.Message {
	return diag.NewMessage(
		MisplacedAnnotation,
		originOrNil(r),
		annotation,
		kind,
	)
}

// NewUnknownAnnotation returns a new diag.Message based on UnknownAnnotation.
func NewUnknownAnnotation(r *resource.Instance, annotation string) diag.Message {
	return diag.NewMessage(
		UnknownAnnotation,
		originOrNil(r),
		annotation,
	)
}

// NewConflictingMeshGatewayVirtualServiceHosts returns a new diag.Message based on ConflictingMeshGatewayVirtualServiceHosts.
func NewConflictingMeshGatewayVirtualServiceHosts(r *resource.Instance, virtualServices string, host string) diag.Message {
	return diag.NewMessage(
		ConflictingMeshGatewayVirtualServiceHosts,
		originOrNil(r),
		virtualServices,
		host,
	)
}

// NewConflictingSidecarWorkloadSelectors returns a new diag.Message based on ConflictingSidecarWorkloadSelectors.
func NewConflictingSidecarWorkloadSelectors(r *resource.Instance, conflictingSidecars []string, namespace string, workloadPod string) diag.Message {
	return diag.NewMessage(
		ConflictingSidecarWorkloadSelectors,
		originOrNil(r),
		conflictingSidecars,
		namespace,
		workloadPod,
	)
}

// NewMultipleSidecarsWithoutWorkloadSelectors returns a new diag.Message based on MultipleSidecarsWithoutWorkloadSelectors.
func NewMultipleSidecarsWithoutWorkloadSelectors(r *resource.Instance, conflictingSidecars []string, namespace string) diag.Message {
	return diag.NewMessage(
		MultipleSidecarsWithoutWorkloadSelectors,
		originOrNil(r),
		conflictingSidecars,
		namespace,
	)
}

// NewVirtualServiceDestinationPortSelectorRequired returns a new diag.Message based on VirtualServiceDestinationPortSelectorRequired.
func NewVirtualServiceDestinationPortSelectorRequired(r *resource.Instance, destHost string, destPorts []int) diag.Message {
	return diag.NewMessage(
		VirtualServiceDestinationPortSelectorRequired,
		originOrNil(r),
		destHost,
		destPorts,
	)
}

// NewMTLSPolicyConflict returns a new diag.Message based on MTLSPolicyConflict.
func NewMTLSPolicyConflict(r *resource.Instance, host string, destinationRuleName string, destinationRuleMTLSMode bool, policyName string, policyMTLSMode string) diag.Message {
	return diag.NewMessage(
		MTLSPolicyConflict,
		originOrNil(r),
		host,
		destinationRuleName,
		destinationRuleMTLSMode,
		policyName,
		policyMTLSMode,
	)
}

// NewPolicySpecifiesPortNameThatDoesntExist returns a new diag.Message based on PolicySpecifiesPortNameThatDoesntExist.
func NewPolicySpecifiesPortNameThatDoesntExist(r *resource.Instance, portName string, host string) diag.Message {
	return diag.NewMessage(
		PolicySpecifiesPortNameThatDoesntExist,
		originOrNil(r),
		portName,
		host,
	)
}

// NewDestinationRuleUsesMTLSForWorkloadWithoutSidecar returns a new diag.Message based on DestinationRuleUsesMTLSForWorkloadWithoutSidecar.
func NewDestinationRuleUsesMTLSForWorkloadWithoutSidecar(r *resource.Instance, destinationRuleName string, host string) diag.Message {
	return diag.NewMessage(
		DestinationRuleUsesMTLSForWorkloadWithoutSidecar,
		originOrNil(r),
		destinationRuleName,
		host,
	)
}

// NewDeploymentAssociatedToMultipleServices returns a new diag.Message based on DeploymentAssociatedToMultipleServices.
func NewDeploymentAssociatedToMultipleServices(r *resource.Instance, deployment string, port int32, services []string) diag.Message {
	return diag.NewMessage(
		DeploymentAssociatedToMultipleServices,
		originOrNil(r),
		deployment,
		port,
		services,
	)
}

// NewDeploymentRequiresServiceAssociated returns a new diag.Message based on DeploymentRequiresServiceAssociated.
func NewDeploymentRequiresServiceAssociated(r *resource.Instance, deployment string) diag.Message {
	return diag.NewMessage(
		DeploymentRequiresServiceAssociated,
		originOrNil(r),
		deployment,
	)
}

// NewPortNameIsNotUnderNamingConvention returns a new diag.Message based on PortNameIsNotUnderNamingConvention.
func NewPortNameIsNotUnderNamingConvention(r *resource.Instance, portName string, port int, targetPort string) diag.Message {
	return diag.NewMessage(
		PortNameIsNotUnderNamingConvention,
		originOrNil(r),
		portName,
		port,
		targetPort,
	)
}

func originOrNil(r *resource.Instance) resource.Origin {
	var o resource.Origin
	if r != nil {
		o = r.Origin
	}
	return o
}
