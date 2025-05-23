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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	gentype "k8s.io/client-go/gentype"

	v1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"
	corev1alpha1 "github.com/kcp-dev/kcp/sdk/client/applyconfiguration/core/v1alpha1"
	typedcorev1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/typed/core/v1alpha1"
)

// fakeShards implements ShardInterface
type fakeShards struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.Shard, *v1alpha1.ShardList, *corev1alpha1.ShardApplyConfiguration]
	Fake *FakeCoreV1alpha1
}

func newFakeShards(fake *FakeCoreV1alpha1) typedcorev1alpha1.ShardInterface {
	return &fakeShards{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.Shard, *v1alpha1.ShardList, *corev1alpha1.ShardApplyConfiguration](
			fake.Fake,
			"",
			v1alpha1.SchemeGroupVersion.WithResource("shards"),
			v1alpha1.SchemeGroupVersion.WithKind("Shard"),
			func() *v1alpha1.Shard { return &v1alpha1.Shard{} },
			func() *v1alpha1.ShardList { return &v1alpha1.ShardList{} },
			func(dst, src *v1alpha1.ShardList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ShardList) []*v1alpha1.Shard { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.ShardList, items []*v1alpha1.Shard) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
