/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/openservicemesh/osm/pkg/gen/client/plugin/clientset/versioned/typed/plugin/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakePluginV1alpha1 struct {
	*testing.Fake
}

func (c *FakePluginV1alpha1) Plugins(namespace string) v1alpha1.PluginInterface {
	return &FakePlugins{c, namespace}
}

func (c *FakePluginV1alpha1) PluginChains(namespace string) v1alpha1.PluginChainInterface {
	return &FakePluginChains{c, namespace}
}

func (c *FakePluginV1alpha1) PluginServices(namespace string) v1alpha1.PluginServiceInterface {
	return &FakePluginServices{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakePluginV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
