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

	tenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	kcpv1alpha1 "github.com/kcp-dev/kcp/sdk/client/applyconfiguration/tenancy/v1alpha1"
	typedkcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster/typed/tenancy/v1alpha1"
	typedtenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/typed/tenancy/v1alpha1"
)

// workspaceTypeClusterClient implements WorkspaceTypeClusterInterface
type workspaceTypeClusterClient struct {
	*kcpgentype.FakeClusterClientWithList[*tenancyv1alpha1.WorkspaceType, *tenancyv1alpha1.WorkspaceTypeList]
	Fake *kcptesting.Fake
}

func newFakeWorkspaceTypeClusterClient(fake *TenancyV1alpha1ClusterClient) typedkcptenancyv1alpha1.WorkspaceTypeClusterInterface {
	return &workspaceTypeClusterClient{
		kcpgentype.NewFakeClusterClientWithList[*tenancyv1alpha1.WorkspaceType, *tenancyv1alpha1.WorkspaceTypeList](
			fake.Fake,
			tenancyv1alpha1.SchemeGroupVersion.WithResource("workspacetypes"),
			tenancyv1alpha1.SchemeGroupVersion.WithKind("WorkspaceType"),
			func() *tenancyv1alpha1.WorkspaceType { return &tenancyv1alpha1.WorkspaceType{} },
			func() *tenancyv1alpha1.WorkspaceTypeList { return &tenancyv1alpha1.WorkspaceTypeList{} },
			func(dst, src *tenancyv1alpha1.WorkspaceTypeList) { dst.ListMeta = src.ListMeta },
			func(list *tenancyv1alpha1.WorkspaceTypeList) []*tenancyv1alpha1.WorkspaceType {
				return kcpgentype.ToPointerSlice(list.Items)
			},
			func(list *tenancyv1alpha1.WorkspaceTypeList, items []*tenancyv1alpha1.WorkspaceType) {
				list.Items = kcpgentype.FromPointerSlice(items)
			},
		),
		fake.Fake,
	}
}

func (c *workspaceTypeClusterClient) Cluster(cluster logicalcluster.Path) typedtenancyv1alpha1.WorkspaceTypeInterface {
	return newFakeWorkspaceTypeClient(c.Fake, cluster)
}

// workspaceTypeScopedClient implements WorkspaceTypeInterface
type workspaceTypeScopedClient struct {
	*kcpgentype.FakeClientWithListAndApply[*tenancyv1alpha1.WorkspaceType, *tenancyv1alpha1.WorkspaceTypeList, *kcpv1alpha1.WorkspaceTypeApplyConfiguration]
	Fake        *kcptesting.Fake
	ClusterPath logicalcluster.Path
}

func newFakeWorkspaceTypeClient(fake *kcptesting.Fake, clusterPath logicalcluster.Path) typedtenancyv1alpha1.WorkspaceTypeInterface {
	return &workspaceTypeScopedClient{
		kcpgentype.NewFakeClientWithListAndApply[*tenancyv1alpha1.WorkspaceType, *tenancyv1alpha1.WorkspaceTypeList, *kcpv1alpha1.WorkspaceTypeApplyConfiguration](
			fake,
			clusterPath,
			"",
			tenancyv1alpha1.SchemeGroupVersion.WithResource("workspacetypes"),
			tenancyv1alpha1.SchemeGroupVersion.WithKind("WorkspaceType"),
			func() *tenancyv1alpha1.WorkspaceType { return &tenancyv1alpha1.WorkspaceType{} },
			func() *tenancyv1alpha1.WorkspaceTypeList { return &tenancyv1alpha1.WorkspaceTypeList{} },
			func(dst, src *tenancyv1alpha1.WorkspaceTypeList) { dst.ListMeta = src.ListMeta },
			func(list *tenancyv1alpha1.WorkspaceTypeList) []*tenancyv1alpha1.WorkspaceType {
				return kcpgentype.ToPointerSlice(list.Items)
			},
			func(list *tenancyv1alpha1.WorkspaceTypeList, items []*tenancyv1alpha1.WorkspaceType) {
				list.Items = kcpgentype.FromPointerSlice(items)
			},
		),
		fake,
		clusterPath,
	}
}
