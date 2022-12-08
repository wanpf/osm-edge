package trafficpolicy

// PluginPolicy defines plugins for a given backend
type PluginPolicy struct {
	// Namespace defines namespace of the plugin
	Namespace string

	// Name defines Name of the plugin
	Name string

	// Script defines pipy script used by the PlugIn.
	Script string
}
