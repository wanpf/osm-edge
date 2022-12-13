package catalog

import (
	pluginv1alpha1 "github.com/openservicemesh/osm/pkg/apis/plugin/v1alpha1"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetPluginPolicies returns the plugin policies
func (mc *MeshCatalog) GetPluginPolicies() []*trafficpolicy.PluginPolicy {
	plugins := mc.pluginController.GetPlugins()
	if plugins == nil {
		log.Trace().Msg("Did not find any plugin policy")
		return nil
	}

	var pluginPolicies []*trafficpolicy.PluginPolicy
	for _, plugin := range plugins {
		policy := new(trafficpolicy.PluginPolicy)
		policy.Name = plugin.Name
		policy.Script = plugin.Spec.PipyScript
		pluginPolicies = append(pluginPolicies, policy)
	}
	return pluginPolicies
}

// GetPluginConfigs lists plugin configs
func (mc *MeshCatalog) GetPluginConfigs() []*pluginv1alpha1.PluginConfig {
	return mc.pluginController.GetPluginConfigs()
}
