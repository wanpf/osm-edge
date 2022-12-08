package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginConfig is the type used to represent a PluginConfig policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginConfig struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PlugIn specification
	// +optional
	Spec PluginConfigSpec `json:"spec,omitempty"`

	// Status is the status of the PlugIn configuration.
	// +optional
	Status PluginConfigStatus `json:"status,omitempty"`
}

// PluginConfigSpec is the type used to represent the PluginConfig policy specification.
type PluginConfigSpec struct {
	// JSON defines the json config of the plugin.
	JSON string `json:"json"`
}

// PluginConfigList defines the list of PluginConfig objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PluginConfig `json:"items"`
}

// PluginConfigStatus is the type used to represent the status of a PluginConfig resource.
type PluginConfigStatus struct {
	// CurrentStatus defines the current status of an AccessCert resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an AccessCert resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
