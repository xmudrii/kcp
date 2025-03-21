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


// Code generated by kcp code-generator. DO NOT EDIT.

package v1alpha1

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"	
	"github.com/kcp-dev/logicalcluster/v3"
	
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"

	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	)

// APIExportEndpointSliceClusterLister can list APIExportEndpointSlices across all workspaces, or scope down to a APIExportEndpointSliceLister for one workspace.
// All objects returned here must be treated as read-only.
type APIExportEndpointSliceClusterLister interface {
	// List lists all APIExportEndpointSlices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*apisv1alpha1.APIExportEndpointSlice, err error)
	// Cluster returns a lister that can list and get APIExportEndpointSlices in one workspace.
Cluster(clusterName logicalcluster.Name) APIExportEndpointSliceLister
APIExportEndpointSliceClusterListerExpansion
}

type aPIExportEndpointSliceClusterLister struct {
	indexer cache.Indexer
}

// NewAPIExportEndpointSliceClusterLister returns a new APIExportEndpointSliceClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
func NewAPIExportEndpointSliceClusterLister(indexer cache.Indexer) *aPIExportEndpointSliceClusterLister {
	return &aPIExportEndpointSliceClusterLister{indexer: indexer}
}

// List lists all APIExportEndpointSlices in the indexer across all workspaces.
func (s *aPIExportEndpointSliceClusterLister) List(selector labels.Selector) (ret []*apisv1alpha1.APIExportEndpointSlice, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*apisv1alpha1.APIExportEndpointSlice))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get APIExportEndpointSlices.
func (s *aPIExportEndpointSliceClusterLister) Cluster(clusterName logicalcluster.Name) APIExportEndpointSliceLister {
return &aPIExportEndpointSliceLister{indexer: s.indexer, clusterName: clusterName}
}

// APIExportEndpointSliceLister can list all APIExportEndpointSlices, or get one in particular.
// All objects returned here must be treated as read-only.
type APIExportEndpointSliceLister interface {
	// List lists all APIExportEndpointSlices in the workspace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*apisv1alpha1.APIExportEndpointSlice, err error)
// Get retrieves the APIExportEndpointSlice from the indexer for a given workspace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*apisv1alpha1.APIExportEndpointSlice, error)
APIExportEndpointSliceListerExpansion
}
// aPIExportEndpointSliceLister can list all APIExportEndpointSlices inside a workspace.
type aPIExportEndpointSliceLister struct {
	indexer cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all APIExportEndpointSlices in the indexer for a workspace.
func (s *aPIExportEndpointSliceLister) List(selector labels.Selector) (ret []*apisv1alpha1.APIExportEndpointSlice, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*apisv1alpha1.APIExportEndpointSlice))
	})
	return ret, err
}

// Get retrieves the APIExportEndpointSlice from the indexer for a given workspace and name.
func (s *aPIExportEndpointSliceLister) Get(name string) (*apisv1alpha1.APIExportEndpointSlice, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(apisv1alpha1.Resource("apiexportendpointslices"), name)
	}
	return obj.(*apisv1alpha1.APIExportEndpointSlice), nil
}
// NewAPIExportEndpointSliceLister returns a new APIExportEndpointSliceLister.
// We assume that the indexer:
// - is fed by a workspace-scoped LIST+WATCH
// - uses cache.MetaNamespaceKeyFunc as the key function
func NewAPIExportEndpointSliceLister(indexer cache.Indexer) *aPIExportEndpointSliceScopedLister {
	return &aPIExportEndpointSliceScopedLister{indexer: indexer}
}

// aPIExportEndpointSliceScopedLister can list all APIExportEndpointSlices inside a workspace.
type aPIExportEndpointSliceScopedLister struct {
	indexer cache.Indexer
}

// List lists all APIExportEndpointSlices in the indexer for a workspace.
func (s *aPIExportEndpointSliceScopedLister) List(selector labels.Selector) (ret []*apisv1alpha1.APIExportEndpointSlice, err error) {
	err = cache.ListAll(s.indexer, selector, func(i interface{}) {
		ret = append(ret, i.(*apisv1alpha1.APIExportEndpointSlice))
	})
	return ret, err
}

// Get retrieves the APIExportEndpointSlice from the indexer for a given workspace and name.
func (s *aPIExportEndpointSliceScopedLister) Get(name string) (*apisv1alpha1.APIExportEndpointSlice, error) {
	key := name
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(apisv1alpha1.Resource("apiexportendpointslices"), name)
	}
	return obj.(*apisv1alpha1.APIExportEndpointSlice), nil
}
