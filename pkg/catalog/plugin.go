package catalog

import (
	pluginv1alpha1 "github.com/openservicemesh/osm/pkg/apis/plugin/v1alpha1"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetPluginPolicies returns the plugin policies for the given mesh service
func (mc *MeshCatalog) GetPluginPolicies(svc service.MeshService) ([]*trafficpolicy.PluginPolicy, error) {
	plugins := mc.pluginController.GetPlugins()
	if plugins == nil {
		log.Trace().Msgf("Did not find plugin policy for service %s", svc)
		return nil, nil
	}

	var pluginPolicies []*trafficpolicy.PluginPolicy

	for _, plugin := range plugins {
		policy := new(trafficpolicy.PluginPolicy)
		policy.Name = plugin.Name
		policy.Namespace = plugin.Namespace
		policy.Script = plugin.Spec.PipyScript
		pluginPolicies = append(pluginPolicies, policy)
	}

	return pluginPolicies, nil
}

// GetPluginChains returns the plugin chains for the given mesh service
func (mc *MeshCatalog) GetPluginChains(svc service.MeshService) ([]*pluginv1alpha1.PluginChain, error) {
	pluginChains := mc.pluginController.GetPluginChains()
	if pluginChains == nil {
		log.Trace().Msgf("Did not find plugin chain for service %s", svc)
		return nil, nil
	}
	return pluginChains, nil
}

// GetPluginConfigs returns the plugin configs for the given mesh service
func (mc *MeshCatalog) GetPluginConfigs(svc service.MeshService) ([]*pluginv1alpha1.PluginConfig, error) {
	pluginConfigs := mc.pluginController.GetPluginConfigs()
	if pluginConfigs == nil {
		log.Trace().Msgf("Did not find plugin config for service %s", svc)
		return nil, nil
	}
	return pluginConfigs, nil
}
