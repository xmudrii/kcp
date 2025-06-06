/*
Copyright The KCP Authors.

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

// Code generated by cluster-client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"

	kcpclient "github.com/kcp-dev/apimachinery/v2/pkg/client"
	"github.com/kcp-dev/logicalcluster/v3"

	kcpapisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/typed/apis/v1alpha1"
)

// APIBindingsClusterGetter has a method to return a APIBindingClusterInterface.
// A group's cluster client should implement this interface.
type APIBindingsClusterGetter interface {
	APIBindings() APIBindingClusterInterface
}

// APIBindingClusterInterface can operate on APIBindings across all clusters,
// or scope down to one cluster and return a kcpv1alpha1.APIBindingInterface.
type APIBindingClusterInterface interface {
	Cluster(logicalcluster.Path) kcpv1alpha1.APIBindingInterface
	List(ctx context.Context, opts v1.ListOptions) (*kcpapisv1alpha1.APIBindingList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	APIBindingClusterExpansion
}

type aPIBindingsClusterInterface struct {
	clientCache kcpclient.Cache[*kcpv1alpha1.ApisV1alpha1Client]
}

// Cluster scopes the client down to a particular cluster.
func (c *aPIBindingsClusterInterface) Cluster(clusterPath logicalcluster.Path) kcpv1alpha1.APIBindingInterface {
	if clusterPath == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return c.clientCache.ClusterOrDie(clusterPath).APIBindings()
}

// List returns the entire collection of all APIBindings across all clusters.
func (c *aPIBindingsClusterInterface) List(ctx context.Context, opts v1.ListOptions) (*kcpapisv1alpha1.APIBindingList, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).APIBindings().List(ctx, opts)
}

// Watch begins to watch all APIBindings across all clusters.
func (c *aPIBindingsClusterInterface) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).APIBindings().Watch(ctx, opts)
}
