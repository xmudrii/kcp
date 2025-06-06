/*
Copyright 2022 The KCP Authors.

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

package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.com/kcp-dev/logicalcluster/v3"

	dynamiccontext "github.com/kcp-dev/kcp/pkg/virtual/framework/dynamic/context"
	apisv1alpha2 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha2"
)

func TestMaximalPermissionPolicyAuthorizer(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		attr                    authorizer.Attributes
		apidomainKey            string
		getAPIExport            func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error)
		getAPIExportsByIdentity func(identityHash string) ([]*apisv1alpha2.APIExport, error)
		newDeepSARAuthorizer    func(clusterName logicalcluster.Name) (authorizer.Authorizer, error)

		expectedErr      string
		expectedDecision authorizer.Decision
		expectedReason   string
	}{
		{
			name:             "invalid domain key",
			attr:             &authorizer.AttributesRecord{User: &user.DefaultInfo{}},
			apidomainKey:     "",
			expectedDecision: authorizer.DecisionNoOpinion,
			expectedErr:      "invalid API domain key",
		},
		{
			name:         "no claimed identities",
			attr:         &authorizer.AttributesRecord{User: &user.DefaultInfo{}},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
				}, nil
			},

			expectedDecision: authorizer.DecisionAllow,
			expectedReason:   `unclaimed resource in API export: "fooExport", workspace :"someWorkspace"`,
		},
		{
			name: "claimed identity without identity hash",
			attr: &authorizer.AttributesRecord{
				User:     &user.DefaultInfo{},
				APIGroup: "claimedGroup",
				Resource: "claimedResource",
			},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
					Spec: apisv1alpha2.APIExportSpec{
						PermissionClaims: []apisv1alpha2.PermissionClaim{
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "someGroup",
									Resource: "someResource",
								},
							},
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "claimedGroup",
									Resource: "claimedResource",
								},
							},
						},
					},
				}, nil
			},

			expectedDecision: authorizer.DecisionAllow,
			expectedReason:   `unclaimable resource, identity hash not set in claiming API export: "fooExport", workspace :"someWorkspace"`,
		},
		{
			name: "claimed identity without api export",
			attr: &authorizer.AttributesRecord{
				User:     &user.DefaultInfo{},
				APIGroup: "claimedGroup",
				Resource: "claimedResource",
			},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
					Spec: apisv1alpha2.APIExportSpec{
						PermissionClaims: []apisv1alpha2.PermissionClaim{
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "someGroup",
									Resource: "someResource",
								},
							},
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "claimedGroup",
									Resource: "claimedResource",
								},
								IdentityHash: "123",
							},
						},
					},
				}, nil
			},
			getAPIExportsByIdentity: func(identityHash string) ([]*apisv1alpha2.APIExport, error) {
				return []*apisv1alpha2.APIExport{}, nil
			},

			expectedDecision: authorizer.DecisionDeny,
			expectedReason:   `no API export providing claimed resources found for identity hash: "123"`,
		},
		{
			name: "claimed identity with api export having no maximum permission policy",
			attr: &authorizer.AttributesRecord{
				User:     &user.DefaultInfo{},
				APIGroup: "claimedGroup",
				Resource: "claimedResource",
			},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
					Spec: apisv1alpha2.APIExportSpec{
						PermissionClaims: []apisv1alpha2.PermissionClaim{
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "someGroup",
									Resource: "someResource",
								},
							},
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "claimedGroup",
									Resource: "claimedResource",
								},
								IdentityHash: "123",
							},
						},
					},
				}, nil
			},
			getAPIExportsByIdentity: func(identityHash string) ([]*apisv1alpha2.APIExport, error) {
				return []*apisv1alpha2.APIExport{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
					},
				}, nil
			},

			expectedDecision: authorizer.DecisionAllow,
			expectedReason:   `all claimed API exports granted access`,
		},
		{
			name: "claimed identity with api export having maximum permission policy granting access",
			attr: &authorizer.AttributesRecord{
				User:     &user.DefaultInfo{},
				APIGroup: "claimedGroup",
				Resource: "claimedResource",
			},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
					Spec: apisv1alpha2.APIExportSpec{
						PermissionClaims: []apisv1alpha2.PermissionClaim{
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "someGroup",
									Resource: "someResource",
								},
							},
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "claimedGroup",
									Resource: "claimedResource",
								},
								IdentityHash: "123",
							},
						},
					},
				}, nil
			},
			getAPIExportsByIdentity: func(identityHash string) ([]*apisv1alpha2.APIExport, error) {
				return []*apisv1alpha2.APIExport{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec: apisv1alpha2.APIExportSpec{
							MaximalPermissionPolicy: &apisv1alpha2.MaximalPermissionPolicy{Local: &apisv1alpha2.LocalAPIExportPolicy{}},
						},
					},
				}, nil
			},
			newDeepSARAuthorizer: func(clusterName logicalcluster.Name) (authorizer.Authorizer, error) {
				return authorizer.AuthorizerFunc(func(ctx context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
					return authorizer.DecisionAllow, "", nil
				}), nil
			},

			expectedDecision: authorizer.DecisionAllow,
			expectedReason:   `all claimed API exports granted access`,
		},
		{
			name: "claimed identity with api export having maximum permission policy denying access",
			attr: &authorizer.AttributesRecord{
				User:     &user.DefaultInfo{},
				APIGroup: "claimedGroup",
				Resource: "claimedResource",
			},
			apidomainKey: "foo/bar",
			getAPIExport: func(clusterName, apiExportName string) (*apisv1alpha2.APIExport, error) {
				return &apisv1alpha2.APIExport{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fooExport",
						Annotations: map[string]string{
							logicalcluster.AnnotationKey: "someWorkspace",
						},
					},
					Spec: apisv1alpha2.APIExportSpec{
						PermissionClaims: []apisv1alpha2.PermissionClaim{
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "someGroup",
									Resource: "someResource",
								},
							},
							{
								GroupResource: apisv1alpha2.GroupResource{
									Group:    "claimedGroup",
									Resource: "claimedResource",
								},
								IdentityHash: "123",
							},
						},
					},
				}, nil
			},
			getAPIExportsByIdentity: func(identityHash string) ([]*apisv1alpha2.APIExport, error) {
				return []*apisv1alpha2.APIExport{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "fooExport",
							Annotations: map[string]string{
								logicalcluster.AnnotationKey: "someWorkspace",
							},
						},
						Spec: apisv1alpha2.APIExportSpec{
							MaximalPermissionPolicy: &apisv1alpha2.MaximalPermissionPolicy{Local: &apisv1alpha2.LocalAPIExportPolicy{}},
						},
					},
				}, nil
			},
			newDeepSARAuthorizer: func(clusterName logicalcluster.Name) (authorizer.Authorizer, error) {
				return authorizer.AuthorizerFunc(func(ctx context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
					return authorizer.DecisionDeny, "access denied", nil
				}), nil
			},

			expectedDecision: authorizer.DecisionNoOpinion,
			expectedReason:   `API export: "fooExport", workspace: "someWorkspace" RBAC decision: access denied`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := dynamiccontext.WithAPIDomainKey(context.Background(), dynamiccontext.APIDomainKey(tc.apidomainKey))
			auth := &maximalPermissionAuthorizer{
				getAPIExport:            tc.getAPIExport,
				getAPIExportsByIdentity: tc.getAPIExportsByIdentity,
				newDeepSARAuthorizer:    tc.newDeepSARAuthorizer,
			}
			dec, reason, err := auth.Authorize(ctx, tc.attr)
			errString := ""
			if err != nil {
				errString = err.Error()
			}
			require.Equal(t, errString, tc.expectedErr)
			require.Equal(t, tc.expectedDecision, dec)
			require.Equal(t, tc.expectedReason, reason)
		})
	}
}
