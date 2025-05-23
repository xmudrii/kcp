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

package fake

import (
	kcpgentype "github.com/kcp-dev/client-go/third_party/k8s.io/client-go/gentype"
	kcptesting "github.com/kcp-dev/client-go/third_party/k8s.io/client-go/testing"
	"github.com/kcp-dev/logicalcluster/v3"

	topologyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/topology/v1alpha1"
	kcpv1alpha1 "github.com/kcp-dev/kcp/sdk/client/applyconfiguration/topology/v1alpha1"
	typedkcptopologyv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster/typed/topology/v1alpha1"
	typedtopologyv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/typed/topology/v1alpha1"
)

// partitionSetClusterClient implements PartitionSetClusterInterface
type partitionSetClusterClient struct {
	*kcpgentype.FakeClusterClientWithList[*topologyv1alpha1.PartitionSet, *topologyv1alpha1.PartitionSetList]
	Fake *kcptesting.Fake
}

func newFakePartitionSetClusterClient(fake *TopologyV1alpha1ClusterClient) typedkcptopologyv1alpha1.PartitionSetClusterInterface {
	return &partitionSetClusterClient{
		kcpgentype.NewFakeClusterClientWithList[*topologyv1alpha1.PartitionSet, *topologyv1alpha1.PartitionSetList](
			fake.Fake,
			topologyv1alpha1.SchemeGroupVersion.WithResource("partitionsets"),
			topologyv1alpha1.SchemeGroupVersion.WithKind("PartitionSet"),
			func() *topologyv1alpha1.PartitionSet { return &topologyv1alpha1.PartitionSet{} },
			func() *topologyv1alpha1.PartitionSetList { return &topologyv1alpha1.PartitionSetList{} },
			func(dst, src *topologyv1alpha1.PartitionSetList) { dst.ListMeta = src.ListMeta },
			func(list *topologyv1alpha1.PartitionSetList) []*topologyv1alpha1.PartitionSet {
				return kcpgentype.ToPointerSlice(list.Items)
			},
			func(list *topologyv1alpha1.PartitionSetList, items []*topologyv1alpha1.PartitionSet) {
				list.Items = kcpgentype.FromPointerSlice(items)
			},
		),
		fake.Fake,
	}
}

func (c *partitionSetClusterClient) Cluster(cluster logicalcluster.Path) typedtopologyv1alpha1.PartitionSetInterface {
	return newFakePartitionSetClient(c.Fake, cluster)
}

// partitionSetScopedClient implements PartitionSetInterface
type partitionSetScopedClient struct {
	*kcpgentype.FakeClientWithListAndApply[*topologyv1alpha1.PartitionSet, *topologyv1alpha1.PartitionSetList, *kcpv1alpha1.PartitionSetApplyConfiguration]
	Fake        *kcptesting.Fake
	ClusterPath logicalcluster.Path
}

func newFakePartitionSetClient(fake *kcptesting.Fake, clusterPath logicalcluster.Path) typedtopologyv1alpha1.PartitionSetInterface {
	return &partitionSetScopedClient{
		kcpgentype.NewFakeClientWithListAndApply[*topologyv1alpha1.PartitionSet, *topologyv1alpha1.PartitionSetList, *kcpv1alpha1.PartitionSetApplyConfiguration](
			fake,
			clusterPath,
			"",
			topologyv1alpha1.SchemeGroupVersion.WithResource("partitionsets"),
			topologyv1alpha1.SchemeGroupVersion.WithKind("PartitionSet"),
			func() *topologyv1alpha1.PartitionSet { return &topologyv1alpha1.PartitionSet{} },
			func() *topologyv1alpha1.PartitionSetList { return &topologyv1alpha1.PartitionSetList{} },
			func(dst, src *topologyv1alpha1.PartitionSetList) { dst.ListMeta = src.ListMeta },
			func(list *topologyv1alpha1.PartitionSetList) []*topologyv1alpha1.PartitionSet {
				return kcpgentype.ToPointerSlice(list.Items)
			},
			func(list *topologyv1alpha1.PartitionSetList, items []*topologyv1alpha1.PartitionSet) {
				list.Items = kcpgentype.FromPointerSlice(items)
			},
		),
		fake,
		clusterPath,
	}
}
