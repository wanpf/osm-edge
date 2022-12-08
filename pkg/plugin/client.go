package plugin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	pluginv1alpha1 "github.com/openservicemesh/osm/pkg/apis/plugin/v1alpha1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewPluginController returns a plugin.Controller interface related to functionality provided by the resources in the plugin.flomesh.io API group
func NewPluginController(informerCollection *informers.InformerCollection, kubeClient kubernetes.Interface, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	pluginEventTypes := k8s.EventTypes{
		Add:    announcements.PluginAdded,
		Update: announcements.PluginUpdated,
		Delete: announcements.PluginDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPlugin, k8s.GetEventHandlerFuncs(shouldObserve, pluginEventTypes, msgBroker))

	pluginChainEventTypes := k8s.EventTypes{
		Add:    announcements.PluginChainAdded,
		Update: announcements.PluginChainUpdated,
		Delete: announcements.PluginChainDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPluginChain, k8s.GetEventHandlerFuncs(shouldObserve, pluginChainEventTypes, msgBroker))

	pluginServiceEventTypes := k8s.EventTypes{
		Add:    announcements.PluginServiceAdded,
		Update: announcements.PluginServiceUpdated,
		Delete: announcements.PluginServiceDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyPluginService, k8s.GetEventHandlerFuncs(shouldObserve, pluginServiceEventTypes, msgBroker))

	return client
}

// GetPlugins lists plugins
func (c *Client) GetPlugins() []*pluginv1alpha1.Plugin {
	var plugins []*pluginv1alpha1.Plugin
	for _, pluginIface := range c.informers.List(informers.InformerKeyPlugin) {
		plugin := pluginIface.(*pluginv1alpha1.Plugin)
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetPluginChains lists plugin chains
func (c *Client) GetPluginChains() []*pluginv1alpha1.PluginChain {
	var pluginChains []*pluginv1alpha1.PluginChain
	for _, pluginChainIface := range c.informers.List(informers.InformerKeyPluginChain) {
		pluginChain := pluginChainIface.(*pluginv1alpha1.PluginChain)
		pluginChains = append(pluginChains, pluginChain)
	}
	return pluginChains
}

// GetPluginServices lists plugin services
func (c *Client) GetPluginServices() []*pluginv1alpha1.PluginService {
	var pluginServices []*pluginv1alpha1.PluginService
	for _, pluginServiceIface := range c.informers.List(informers.InformerKeyPluginService) {
		pluginService := pluginServiceIface.(*pluginv1alpha1.PluginService)
		pluginServices = append(pluginServices, pluginService)
	}
	return pluginServices
}
