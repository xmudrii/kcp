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

	v1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/client/applyconfiguration/apis/v1alpha1"
	typedapisv1alpha1 "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/typed/apis/v1alpha1"
)

// fakeAPIExports implements APIExportInterface
type fakeAPIExports struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.APIExport, *v1alpha1.APIExportList, *apisv1alpha1.APIExportApplyConfiguration]
	Fake *FakeApisV1alpha1
}

func newFakeAPIExports(fake *FakeApisV1alpha1) typedapisv1alpha1.APIExportInterface {
	return &fakeAPIExports{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.APIExport, *v1alpha1.APIExportList, *apisv1alpha1.APIExportApplyConfiguration](
			fake.Fake,
			"",
			v1alpha1.SchemeGroupVersion.WithResource("apiexports"),
			v1alpha1.SchemeGroupVersion.WithKind("APIExport"),
			func() *v1alpha1.APIExport { return &v1alpha1.APIExport{} },
			func() *v1alpha1.APIExportList { return &v1alpha1.APIExportList{} },
			func(dst, src *v1alpha1.APIExportList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.APIExportList) []*v1alpha1.APIExport { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.APIExportList, items []*v1alpha1.APIExport) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
