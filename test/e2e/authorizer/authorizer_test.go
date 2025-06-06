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
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	controlplaneapiserver "k8s.io/kubernetes/pkg/controlplane/apiserver"

	kcpdiscovery "github.com/kcp-dev/client-go/discovery"
	kcpdynamic "github.com/kcp-dev/client-go/dynamic"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"

	confighelpers "github.com/kcp-dev/kcp/config/helpers"
	"github.com/kcp-dev/kcp/pkg/authorization"
	kcptesting "github.com/kcp-dev/kcp/sdk/testing"
	kcptestinghelpers "github.com/kcp-dev/kcp/sdk/testing/helpers"
	"github.com/kcp-dev/kcp/test/e2e/framework"
)

//go:embed testdata/*.yaml
var embeddedResources embed.FS

func TestAuthorizer(t *testing.T) {
	t.Parallel()
	framework.Suite(t, "control-plane")

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	server := kcptesting.SharedKcpServer(t)
	cfg := server.BaseConfig(t)
	rootShardCfg := server.RootShardSystemMasterBaseConfig(t)

	kubeClusterClient, err := kcpkubernetesclientset.NewForConfig(cfg)
	require.NoError(t, err)
	kubeDiscoveryClient, err := kcpdiscovery.NewForConfig(cfg)
	require.NoError(t, err)
	dynamicClusterClient, err := kcpdynamic.NewForConfig(cfg)
	require.NoError(t, err)

	org1, _ := framework.NewOrganizationFixture(t, server, kcptesting.WithNamePrefix("org1")) //nolint:staticcheck // TODO: switch to NewWorkspaceFixture.
	org2, _ := framework.NewRootShardOrganizationFixture(t, server, kcptesting.PrivilegedWorkspaceOption(kcptesting.WithNamePrefix("org2")), framework.WithRequiredGroups("empty-group"))

	kcptesting.NewWorkspaceFixture(t, server, org1, kcptesting.WithName("workspace1"))
	kcptesting.NewWorkspaceFixture(t, server, org1, kcptesting.WithName("workspace2"), kcptesting.WithRootShard())                      // on root for system:admin ClusterRole test
	_, org2Workspace1 := kcptesting.NewWorkspaceFixture(t, server, org2, kcptesting.WithName("workspace1"), kcptesting.WithRootShard()) // on root for deep SAR test
	kcptesting.NewWorkspaceFixture(t, server, org2, kcptesting.WithName("workspace2"))

	createResources(ctx, t, dynamicClusterClient, kubeDiscoveryClient, org1.Join("workspace1"), "testdata/workspace1-resources.yaml")
	createResources(ctx, t, dynamicClusterClient, kubeDiscoveryClient, org2.Join("workspace1"), "testdata/workspace1-resources.yaml")

	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org1, []string{"user-1", "user-2", "user-3", "user-4", "user-5"}, nil, false)

	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org2.Join("workspace1"), []string{"user-1"}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org2.Join("workspace1"), []string{"user-2"}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org2.Join("workspace2"), []string{"user-3"}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org2.Join("workspace2"), []string{"user-2"}, nil, true)

	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org1.Join("workspace1"), []string{"user-1"}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org1.Join("workspace1"), []string{"user-2"}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org1.Join("workspace2"), []string{"user-3", "user-4", "user-5"}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClusterClient, org1.Join("workspace2"), []string{"user-2"}, nil, true) // last, to be used for priming below

	user1KubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-1", cfg))
	require.NoError(t, err)
	user1KubeDiscoveryClient, err := kcpdiscovery.NewForConfig(framework.StaticTokenUserConfig("user-1", cfg))
	require.NoError(t, err)
	user2KubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-2", cfg))
	require.NoError(t, err)
	user3KubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-3", cfg))
	require.NoError(t, err)
	user4KubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-4", cfg))
	require.NoError(t, err)
	user5KubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-5", cfg))
	require.NoError(t, err)

	t.Logf("Priming the authorization cache")
	require.Eventually(t, func() bool {
		// test *last* of the admitted permissions
		_, err := user2KubeClusterClient.Cluster(org1.Join("workspace2")).CoreV1().Secrets("default").List(ctx, metav1.ListOptions{})
		return err == nil
	}, 2*wait.ForeverTestTimeout, 100*time.Millisecond)

	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"as org member, workspace admin user-1 can access everything", func(t *testing.T) {
			_, err := user1KubeClusterClient.Cluster(org1.Join("workspace1")).CoreV1().ConfigMaps("default").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			_, err = user1KubeClusterClient.Cluster(org1.Join("workspace1")).CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, metav1.CreateOptions{})
			require.NoError(t, err)
			_, err = user1KubeClusterClient.Cluster(org1.Join("workspace1")).CoreV1().ConfigMaps("test").Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, metav1.CreateOptions{})
			require.NoError(t, err)
		}},
		{"with org access, workspace1 non-admin user-2 can access according to local policy", func(t *testing.T) {
			_, err := user2KubeClusterClient.Cluster(org1.Join("workspace1")).CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, metav1.CreateOptions{})
			require.Errorf(t, err, "user-2 should not be able to create namespace in %s", org1.Join("workspace1"))
			_, err = user2KubeClusterClient.Cluster(org1.Join("workspace1")).CoreV1().Secrets("default").List(ctx, metav1.ListOptions{})
			require.NoErrorf(t, err, "user-2 should be able to list secrets in %s as defined in the local policy", org1.Join("workspace1"))
		}},
		{"with org access, workspace1 non-admin user-2 can access /healthz, /livez, /readyz etc", func(t *testing.T) {
			cl := user2KubeClusterClient.RESTClient()
			requestPath := org1.RequestPath()
			{
				for endpoint := range sets.New[string]("/healthz", "/readyz", "/livez") {
					req := cl.Get().AbsPath(requestPath + endpoint)
					respBytes, err := req.DoRaw(ctx)
					require.NoError(t, err, "should be able to GET %s", endpoint)
					require.Equal(t, "ok", string(respBytes))
				}
			}
			{
				endpoint := "/version"
				req := cl.Get().AbsPath(requestPath + endpoint)
				respBytes, err := req.DoRaw(ctx)
				require.NoError(t, err, "should be able to GET %s", endpoint)
				require.NotEmpty(t, string(respBytes))
				version := new(apimachineryversion.Info)
				require.NoError(t, json.Unmarshal(respBytes, version))
			}
		}},
		{"without org access, org1 workspace1 admin user-1 cannot access org2, not even discovery", func(t *testing.T) {
			_, err := user1KubeClusterClient.Cluster(org2.Join("workspace1")).CoreV1().ConfigMaps("default").List(ctx, metav1.ListOptions{})
			require.Errorf(t, err, "user-1 should not be able to list configmaps in a different org (%s)", org2.Join("workspace1"))
			_, err = user1KubeDiscoveryClient.Cluster(org2.Join("workspace1")).ServerResourcesForGroupVersion("rbac.authorization.k8s.io/v1") // can't be core because that always returns nil
			require.Errorf(t, err, "user-1 should not be able to list server resources in a different org (%s)", org2.Join("workspace1"))
		}},
		{"as org member, workspace1 admin user-1 cannot access workspace2, not even discovery", func(t *testing.T) {
			_, err := user1KubeClusterClient.Cluster(org1.Join("workspace2")).CoreV1().ConfigMaps("default").List(ctx, metav1.ListOptions{})
			require.Errorf(t, err, "user-1 should not be able to list configmaps in a different workspace (%s)", org1.Join("workspace2"))
			_, err = user1KubeDiscoveryClient.Cluster(org2.Join("workspace1")).ServerResourcesForGroupVersion("rbac.authorization.k8s.io/v1") // can't be core because that always returns nil
			require.Errorf(t, err, "user-1 should not be able to list server resources in a different workspace (%s)", org1.Join("workspace2"))
		}},
		{"with org access, workspace2 admin user-2 can access workspace2", func(t *testing.T) {
			_, err := user2KubeClusterClient.Cluster(org1.Join("workspace2")).CoreV1().ConfigMaps("default").List(ctx, metav1.ListOptions{})
			require.NoError(t, err, "user-2 should be able to list configmaps in workspace2 (%s)", org1.Join("workspace2"))
		}},
		{"cluster admins can use wildcard clusters, non-cluster admin cannot", func(t *testing.T) {
			// create client talking directly to root shard to test wildcard requests
			rootKubeClusterClient, err := kcpkubernetesclientset.NewForConfig(rootShardCfg)
			require.NoError(t, err)
			user1RootKubeClusterClient, err := kcpkubernetesclientset.NewForConfig(framework.StaticTokenUserConfig("user-1", rootShardCfg))
			require.NoError(t, err)

			_, err = rootKubeClusterClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			_, err = user1RootKubeClusterClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			require.Error(t, err, "Only cluster admins can use all clusters at once")
		}},
		{"with system:admin permissions, workspace2 non-admin user-3 can list Namespaces with a bootstrap ClusterRole", func(t *testing.T) {
			t.Logf("User-3 cannot access namespaces in %s", org1.Join("workspace2"))
			_, err = user3KubeClusterClient.Cluster(org1.Join("workspace2")).CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			require.Error(t, err, "User-3 shouldn't be able to list Namespaces")

			bootstrapClusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("kcp-authorizer-test-namespace-lister-%d", rand.Uint32()),
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
					},
				},
			}

			localAuthorizerClusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "user-3-k8s-ref",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "User",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     "user-3",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     bootstrapClusterRole.Name,
				},
			}

			t.Logf("Creating ClusterRoleBinding %s in %s", localAuthorizerClusterRoleBinding.Name, org1.Join("workspace2"))
			_, err = user2KubeClusterClient.Cluster(org1.Join("workspace2")).RbacV1().ClusterRoleBindings().Create(ctx, localAuthorizerClusterRoleBinding, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Logf("Creating matching ClusterRole %s in %s", bootstrapClusterRole.Name, controlplaneapiserver.LocalAdminCluster)
			shardKubeClusterClient, err := kcpkubernetesclientset.NewForConfig(rootShardCfg)
			require.NoError(t, err)
			_, err = shardKubeClusterClient.Cluster(controlplaneapiserver.LocalAdminCluster.Path()).RbacV1().ClusterRoles().Create(ctx, bootstrapClusterRole, metav1.CreateOptions{})
			require.NoError(t, err)

			kcptestinghelpers.Eventually(t, func() (bool, string) {
				if _, err := user3KubeClusterClient.Cluster(org1.Join("workspace2")).CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("kcp-authorizer-test-namespace-%d", rand.Uint32()),
					},
				}, metav1.CreateOptions{}); err != nil {
					return false, fmt.Sprintf("failed to create test namespace: %v", err)
				}
				return true, ""
			}, wait.ForeverTestTimeout, time.Millisecond*100, "User-3 should now be able to list Namespaces in %s", org1.Join("workspace2"))
		}},
		{"without org access, a deep SAR with user-1 against org2 succeeds even without org access for user-1", func(t *testing.T) {
			t.Logf("try to list ConfigMap as user-1 in %q without access, should fail", org2.Join("workspace1"))
			_, err := user1KubeClusterClient.Cluster(org2.Join("workspace1")).CoreV1().ConfigMaps("default").List(ctx, metav1.ListOptions{})
			require.Errorf(t, err, "user-1 should not be able to list configmaps in %q", org2.Join("workspace1"))

			sar := &authorizationv1.SubjectAccessReview{
				Spec: authorizationv1.SubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{Namespace: "default", Verb: "list", Version: "v1", Resource: "configmaps"},
					User:               "user-1",
					Groups:             []string{"team-1"},
				},
			}

			t.Logf("ask with normal SAR that user-1 cannot access %q because it has no access", org2.Join("workspace1"))
			resp, err := kubeClusterClient.Cluster(org2.Join("workspace1")).AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
			require.NoError(t, err)
			require.Equalf(t, "access denied", resp.Status.Reason, "SAR should answer that user-1 has no workspace access in %q", org2.Join("workspace1"))
			require.Falsef(t, resp.Status.Allowed, "SAR should correctly answer that user-1 CANNOT list configmaps in %q because it has no access to it", org2.Join("workspace1"))

			t.Logf("ask with normal SAR that user-1 can access %q because it has access", org1.Join("workspace1"))
			resp, err = kubeClusterClient.Cluster(org1.Join("workspace1")).AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
			require.NoError(t, err)
			require.Truef(t, resp.Status.Allowed, "SAR should correctly answer that user-1 CAN list configmaps in %q because it has access to %q", org2.Join("workspace1"), org1.Join("workspace1"))

			t.Logf("ask with deep SAR that user-1 hypothetically could list configmaps in %q if it had access", org2.Join("workspace1"))
			deepSARClient, err := kcpkubernetesclientset.NewForConfig(authorization.WithDeepSARConfig(rest.CopyConfig(server.RootShardSystemMasterBaseConfig(t))))
			require.NoError(t, err)
			kcptestinghelpers.Eventually(t, func() (bool, string) {
				resp, err = deepSARClient.Cluster(logicalcluster.NewPath(org2Workspace1.Spec.Cluster)).AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
				if err != nil {
					return false, fmt.Sprintf("failed to create SAR: %v", err)
				}
				return resp.Status.Allowed, resp.Status.Reason
			}, wait.ForeverTestTimeout, time.Millisecond*100, "SAR should answer hypothetically that user-1 could list configmaps in %q if it had access", org2.Join("workspace1"))
		}},
		{"default SelfSubjectRulesReview for non-admin user should be a fixed list of rules", func(t *testing.T) {
			ws := org1.Join("workspace2")

			// user shouldn't be able to do a SelfSubjectRulesReview in a workspace they don't have access to.
			t.Logf("verifying that user-4 cannot create SelfSubjectRulesReview in %s", org1.Join("workspace1"))
			_, err := user4KubeClusterClient.Cluster(org1.Join("workspace1")).AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &authorizationv1.SelfSubjectRulesReview{Spec: authorizationv1.SelfSubjectRulesReviewSpec{Namespace: corev1.NamespaceDefault}}, metav1.CreateOptions{})
			require.Error(t, err)

			// the user has been admitted to workspace2, they should have the basic set of rules returned on a SelfSubjectRulesReview.
			t.Logf("creating SelfSubjectRulesReview as user-4 in %s", ws)
			review, err := user4KubeClusterClient.Cluster(ws).AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &authorizationv1.SelfSubjectRulesReview{Spec: authorizationv1.SelfSubjectRulesReviewSpec{Namespace: corev1.NamespaceDefault}}, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Logf("checking SelfSubjectRulesReview response in %s to be the expected set of default rules", org1.Join("workspace2"))
			require.ElementsMatch(t, []authorizationv1.ResourceRule{
				{
					Verbs:     []string{"create"},
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"selfsubjectaccessreviews", "selfsubjectrulesreviews"},
				},
				{
					Verbs:     []string{"create"},
					APIGroups: []string{"authentication.k8s.io"},
					Resources: []string{"selfsubjectreviews"},
				},
			}, review.Status.ResourceRules)

			require.ElementsMatch(t, []authorizationv1.NonResourceRule{
				{
					Verbs:           []string{"get"},
					NonResourceURLs: []string{"/api", "/api/*", "/apis", "/apis/*", "/healthz", "/livez", "/openapi", "/openapi/*", "/readyz", "/version", "/version/"},
				},
				{
					Verbs:           []string{"get"},
					NonResourceURLs: []string{"/healthz", "/livez", "/readyz", "/version", "/version/"},
				},
				{
					Verbs:           []string{"access"},
					NonResourceURLs: []string{"/"},
				},
			}, review.Status.NonResourceRules)

			// no authorizer in the chain should mark this as incomplete.
			require.False(t, review.Status.Incomplete)
		}},
		{"SelfSubjectRulesReview includes rules coming from system:admin ClusterRole", func(t *testing.T) {
			ws := org1.Join("workspace2")

			rootShardSystemMasterClient, err := kcpkubernetesclientset.NewForConfig(rest.CopyConfig(server.RootShardSystemMasterBaseConfig(t)))
			require.NoError(t, err)

			clusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("kcp-bootstrap-namespace-%d", rand.Uint32()),
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
					},
				},
			}
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s:user-5", clusterRole.Name),
				},
				Subjects: []rbacv1.Subject{{
					Kind: "User",
					Name: "user-5",
				}},
				RoleRef: rbacv1.RoleRef{
					Kind: "ClusterRole",
					Name: clusterRole.Name,
				},
			}

			t.Logf("creating ClusterRole %s in workspace %s", clusterRole.Name, controlplaneapiserver.LocalAdminCluster)
			_, err = rootShardSystemMasterClient.Cluster(controlplaneapiserver.LocalAdminCluster.Path()).RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
			require.NoError(t, err)

			t.Logf("creating ClusterRoleBinding %s in workspace %s", clusterRoleBinding.Name, ws)
			_, err = kubeClusterClient.Cluster(ws).RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
			require.NoError(t, err)

			kcptestinghelpers.Eventually(t, func() (bool, string) {
				review, err := user5KubeClusterClient.Cluster(ws).AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &authorizationv1.SelfSubjectRulesReview{Spec: authorizationv1.SelfSubjectRulesReviewSpec{Namespace: corev1.NamespaceDefault}}, metav1.CreateOptions{})
				if err != nil {
					return false, fmt.Sprintf("failed to create SelfSubjectRulesReview: %v", err)
				}

				t.Logf("checking if review response includes permissions granted by system:admin ClusterRole bound in %s", ws)
				return assert.Contains(t, review.Status.ResourceRules, authorizationv1.ResourceRule{Verbs: []string{"*"}, APIGroups: []string{""}, Resources: []string{"namespaces"}}), "returned resource rules do not contain '*' verb for namespaces."
			}, wait.ForeverTestTimeout, time.Millisecond*100, "SelfSubjectRulesReview response in %s should contain resource rule for namespaces", org2.Join("workspace1"))
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.run(t)
		})
	}
}

func createResources(ctx context.Context, t *testing.T, dynamicClusterClient kcpdynamic.ClusterInterface, discoveryClusterClient kcpdiscovery.DiscoveryClusterInterface, clusterName logicalcluster.Path, fileName string) {
	t.Helper()
	t.Logf("Create resources in %s", clusterName)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClusterClient.Cluster(clusterName)))
	require.Eventually(t, func() bool {
		if err := confighelpers.CreateResourceFromFS(ctx, dynamicClusterClient.Cluster(clusterName), mapper, nil, fileName, embeddedResources); err != nil {
			t.Logf("failed to create resources: %v", err)
			return false
		}
		return true
	}, wait.ForeverTestTimeout, time.Millisecond*100, "failed to create resources")
}
