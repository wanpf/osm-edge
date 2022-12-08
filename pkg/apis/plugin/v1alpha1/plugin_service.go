package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PluginService is the type used to represent a plugin service policy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginService struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the PlugIn specification
	// +optional
	Spec PluginServiceSpec `json:"spec,omitempty"`

	// Status is the status of the plugin service configuration.
	// +optional
	Status PluginServiceStatus `json:"status,omitempty"`
}

// PluginServiceSpec is the type used to represent the plugin service specification.
type PluginServiceSpec struct {
	// JSON defines the json config of the plugin.
	JSON string `json:"json"`
}

// PluginServiceList defines the list of PluginService objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PluginServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PluginService `json:"items"`
}

// PluginServiceStatus is the type used to represent the status of a plugin service resource.
type PluginServiceStatus struct {
	// CurrentStatus defines the current status of an AccessCert resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an AccessCert resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}
