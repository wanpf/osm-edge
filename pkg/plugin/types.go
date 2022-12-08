// Package plugin implements the Kubernetes client for the resources in the plugin.flomesh.io API group
package plugin

import (
	"k8s.io/client-go/kubernetes"

	pluginv1alpha1 "github.com/openservicemesh/osm/pkg/apis/plugin/v1alpha1"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("plugin-controller")
)

// Client is the type used to represent the Kubernetes Client for the plugin.flomesh.io API group
type Client struct {
	informers      *informers.InformerCollection
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the plugin.flomesh.io API group
type Controller interface {
	// GetPluginService get plugin service
	GetPluginService(svc service.MeshService) *pluginv1alpha1.PluginService

	// GetPlugins lists plugins
	GetPlugins() []*pluginv1alpha1.Plugin

	// GetPluginChains lists plugin chains
	GetPluginChains() []*pluginv1alpha1.PluginChain

	// GetPluginServices lists plugin services
	GetPluginServices() []*pluginv1alpha1.PluginService
}
