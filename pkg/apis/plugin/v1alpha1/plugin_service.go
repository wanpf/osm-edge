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
	// Onload is the type used to represent the plugin service onload chain.
	// +optional
	Onload *OnloadChainSpec `json:"onload,omitempty"`

	// Inbound is the type used to represent the plugin service inbound chain.
	// +optional
	Inbound *InboundChainSpec `json:"inbound,omitempty"`

	// Outbound is the type used to represent the plugin service outbound chain.
	// +optional
	Outbound *InboundChainSpec `json:"outbound,omitempty"`

	// Unload is the type used to represent the plugin service unload chain.
	// +optional
	Unload *UnloadChainSpec `json:"unload,omitempty"`
}

// OnloadChainSpec is the type used to represent the plugin service onload chain.
type OnloadChainSpec struct {
	// Plugins is a list of mounted plugins applied
	Plugins []MountedPlugin `json:"plugins"`
}

// UnloadChainSpec is the type used to represent the plugin service unload chain.
type UnloadChainSpec struct {
	// Plugins is a list of mounted plugins applied
	Plugins []MountedPlugin `json:"plugins"`
}

// InboundChainSpec is the type used to represent the plugin service inbound chain.
type InboundChainSpec struct {
	// Sources are the pod or group of pods to allow plugin traffic
	Sources []IdentityBindingSubject `json:"sources,omitempty"`
}

// OutboundChainSpec is the type used to represent the plugin service outbound chain.
type OutboundChainSpec struct {
	// Destinations are the pod or group of pods to allow plugin traffic
	Destinations []IdentityBindingSubject `json:"destinations,omitempty"`
}

// IdentityBindingSubject is a Kubernetes objects which should be allowed access to the plugin traffic target
type IdentityBindingSubject struct {
	// Kind is the type of Subject to allow ingress (ServiceAccount | Group)
	Kind string `json:"kind"`

	// Name of the Subject, i.e. ServiceAccountName
	Name string `json:"name"`

	// Namespace where the Subject is deployed
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Rules are the traffic rules to allow
	// +optional
	Rules []TrafficTargetRule `json:"rules,omitempty"`

	// Plugins is a list of mounted plugins applied
	Plugins []MountedPlugin `json:"plugins"`
}

// TrafficTargetRule is the TrafficSpec to allow for a TrafficTarget
type TrafficTargetRule struct {
	// Kind is the kind of TrafficSpec to allow
	Kind string `json:"kind"`

	// Name of the TrafficSpec to use
	Name string `json:"name"`

	// Matches is a list of TrafficSpec routes to allow traffic for
	// +optional
	Matches []string `json:"matches,omitempty"`

	// Plugins is a list of mounted plugins applied
	Plugins []MountedPlugin `json:"plugins"`
}

// MountedPlugin is the type used to represent the mounted plugin.
type MountedPlugin struct {
	// Namespace defines the namespace of the plugin.
	Namespace string `json:"namespace"`

	// Name defines the Name of the plugin.
	Name string `json:"name"`

	// MountPoint defines the mount point of the plugin.
	MountPoint string `json:"mountpoint"`
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
