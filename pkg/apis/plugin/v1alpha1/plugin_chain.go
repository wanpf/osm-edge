package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginChain is the type used to represent a PluginChain.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginChain struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PlugIn specification
	// +optional
	Spec PluginChainSpec `json:"spec,omitempty"`

	// Status is the status of the PlugIn configuration.
	// +optional
	Status PluginChainStatus `json:"status,omitempty"`
}

// PluginChainSpec is the type used to represent the PluginChain specification.
type PluginChainSpec struct {
	// InboundChains defines the chains of inbound.
	InboundChains InboundChains `json:"InboundChains"`

	// OutboundChains defines the chains of outbound.
	OutboundChains OutboundChains `json:"OutboundChains"`
}

// InboundChains is the type used to represent inbound chains.
type InboundChains struct {
	// InboundL4Chains defines the l4 chains of outbound.
	InboundL4Chains InboundL4Chains `json:"L4"`

	// InboundL7Chains defines the l7 chains of outbound.
	InboundL7Chains InboundL7Chains `json:"L7"`
}

// OutboundChains is the type used to represent outbound chains.
type OutboundChains struct {
	// OutboundL4Chains defines the l4 chains of outbound.
	OutboundL4Chains OutboundL4Chains `json:"L4"`

	// OutboundL7Chains defines the l7 chains of outbound.
	OutboundL7Chains OutboundL7Chains `json:"L7"`
}

// InboundL4Chains is the type used to represent inbound L4 chains.
type InboundL4Chains struct {
	// TCPFirst defines TCPFirst sub-chains.
	TCPFirst []string `json:"TCPFirst"`

	// TCPAfterTLS defines TCPAfterTLS sub-chains.
	TCPAfterTLS []string `json:"TCPAfterTLS"`

	// TCPAfterRouting defines TCPAfterRouting sub-chains.
	TCPAfterRouting []string `json:"TCPAfterRouting"`

	// TCPLast defines TCPLast sub-chains.
	TCPLast []string `json:"TCPLast"`
}

// InboundL7Chains is the type used to represent inbound L7 chains.
type InboundL7Chains struct {
	// HTTPFirst defines HTTPFirst sub-chains.
	HTTPFirst []string `json:"HTTPFirst"`

	// HTTPAfterTLS defines HTTPAfterTLS sub-chains.
	HTTPAfterTLS []string `json:"HTTPAfterTLS"`

	// HTTPAfterDemux defines HTTPAfterDemux sub-chains.
	HTTPAfterDemux []string `json:"HTTPAfterDemux"`

	// HTTPAfterRouting defines HTTPAfterRouting sub-chains.
	HTTPAfterRouting []string `json:"HTTPAfterRouting"`

	// HTTPAfterMux defines HTTPAfterMux sub-chains.
	HTTPAfterMux []string `json:"HTTPAfterMux"`

	// HTTPLast defines HTTPLast sub-chains.
	HTTPLast []string `json:"HTTPLast"`
}

// OutboundL4Chains is the type used to represent outbound L4 chains.
type OutboundL4Chains struct {
	// TCPFirst defines TCPFirst sub-chains.
	TCPFirst []string `json:"TCPFirst"`

	// TCPAfterRouting defines TCPAfterRouting sub-chains.
	TCPAfterRouting []string `json:"TCPAfterRouting"`

	// TCPLast defines TCPLast sub-chains.
	TCPLast []string `json:"TCPLast"`
}

// OutboundL7Chains is the type used to represent outbound L7 chains.
type OutboundL7Chains struct {
	// HTTPFirst defines HTTPFirst sub-chains.
	HTTPFirst []string `json:"HTTPFirst"`

	// HTTPAfterDemux defines HTTPAfterDemux sub-chains.
	HTTPAfterDemux []string `json:"HTTPAfterDemux"`

	// HTTPAfterRouting defines HTTPAfterRouting sub-chains.
	HTTPAfterRouting []string `json:"HTTPAfterRouting"`

	// HTTPAfterMux defines HTTPAfterMux sub-chains.
	HTTPAfterMux []string `json:"HTTPAfterMux"`

	// HTTPLast defines tcp last sub-chains.
	HTTPLast []string `json:"HTTPLast"`
}

// PluginChainList defines the list of PluginChain objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PluginChain `json:"items"`
}

// PluginChainStatus is the type used to represent the status of a PluginChain resource.
type PluginChainStatus struct {
	// CurrentStatus defines the current status of an AccessCert resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an AccessCert resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
