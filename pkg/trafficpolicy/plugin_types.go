package trafficpolicy

import pluginv1alpha1 "github.com/openservicemesh/osm/pkg/apis/plugin/v1alpha1"

// PluginPolicy defines plugins for a given backend
type PluginPolicy struct {
	pluginv1alpha1.PluginIdentity

	// Script defines pipy script used by the PlugIn.
	Script string
}
