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
	Status PluginConfigStatus `json:"status,omitempty"`
}

// PluginChainSpec is the type used to represent the PluginChain specification.
type PluginChainSpec struct {

	// Entry defines the entry of the plugin.
	Entry string `json:"entry"`
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
