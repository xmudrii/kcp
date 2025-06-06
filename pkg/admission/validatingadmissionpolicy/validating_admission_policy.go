/*
Copyright 2023 The KCP Authors.

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

package validatingadmissionpolicy

import (
	"context"
	"io"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/admission/plugin/policy/generic"
	"k8s.io/apiserver/pkg/admission/plugin/policy/validating"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"

	kcpdynamic "github.com/kcp-dev/client-go/dynamic"
	kcpkubernetesinformers "github.com/kcp-dev/client-go/informers"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"

	"github.com/kcp-dev/kcp/pkg/admission/initializers"
	"github.com/kcp-dev/kcp/pkg/admission/kubequota"
	kcpinformers "github.com/kcp-dev/kcp/sdk/client/informers/externalversions"
	corev1alpha1informers "github.com/kcp-dev/kcp/sdk/client/informers/externalversions/core/v1alpha1"
)

const PluginName = "KCPValidatingAdmissionPolicy"

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(config io.Reader) (admission.Interface, error) {
			return NewKubeValidatingAdmissionPolicy(), nil
		},
	)
}

func NewKubeValidatingAdmissionPolicy() *KubeValidatingAdmissionPolicy {
	return &KubeValidatingAdmissionPolicy{
		Handler:   admission.NewHandler(admission.Connect, admission.Create, admission.Delete, admission.Update),
		delegates: make(map[logicalcluster.Name]*stoppableValidatingAdmissionPolicy),
	}
}

type KubeValidatingAdmissionPolicy struct {
	*admission.Handler

	// Injected/set via initializers
	logicalClusterInformer          corev1alpha1informers.LogicalClusterClusterInformer
	kubeClusterClient               kcpkubernetesclientset.ClusterInterface
	dynamicClusterClient            kcpdynamic.ClusterInterface
	localKubeSharedInformerFactory  kcpkubernetesinformers.SharedInformerFactory
	globalKubeSharedInformerFactory kcpkubernetesinformers.SharedInformerFactory
	serverDone                      <-chan struct{}
	authorizer                      authorizer.Authorizer

	lock      sync.RWMutex
	delegates map[logicalcluster.Name]*stoppableValidatingAdmissionPolicy

	logicalClusterDeletionMonitorStarter sync.Once
}

var _ admission.ValidationInterface = &KubeValidatingAdmissionPolicy{}
var _ = initializers.WantsKubeClusterClient(&KubeValidatingAdmissionPolicy{})
var _ = initializers.WantsKubeInformers(&KubeValidatingAdmissionPolicy{})
var _ = initializers.WantsServerShutdownChannel(&KubeValidatingAdmissionPolicy{})
var _ = initializers.WantsDynamicClusterClient(&KubeValidatingAdmissionPolicy{})
var _ = initializer.WantsAuthorizer(&KubeValidatingAdmissionPolicy{})
var _ = admission.InitializationValidator(&KubeValidatingAdmissionPolicy{})

func (k *KubeValidatingAdmissionPolicy) SetKubeClusterClient(kubeClusterClient kcpkubernetesclientset.ClusterInterface) {
	k.kubeClusterClient = kubeClusterClient
}

func (k *KubeValidatingAdmissionPolicy) SetKcpInformers(local, global kcpinformers.SharedInformerFactory) {
	k.logicalClusterInformer = local.Core().V1alpha1().LogicalClusters()
}

func (k *KubeValidatingAdmissionPolicy) SetKubeInformers(local, global kcpkubernetesinformers.SharedInformerFactory) {
	k.localKubeSharedInformerFactory = local
	k.globalKubeSharedInformerFactory = global
}

func (k *KubeValidatingAdmissionPolicy) SetServerShutdownChannel(ch <-chan struct{}) {
	k.serverDone = ch
}

func (k *KubeValidatingAdmissionPolicy) SetDynamicClusterClient(c kcpdynamic.ClusterInterface) {
	k.dynamicClusterClient = c
}

func (k *KubeValidatingAdmissionPolicy) SetAuthorizer(authz authorizer.Authorizer) {
	k.authorizer = authz
}

func (k *KubeValidatingAdmissionPolicy) ValidateInitialization() error {
	return nil
}

func (k *KubeValidatingAdmissionPolicy) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	k.logicalClusterDeletionMonitorStarter.Do(func() {
		m := kubequota.NewLogicalClusterDeletionMonitor("kubequota-logicalcluster-deletion-monitor", k.logicalClusterInformer, k.logicalClusterDeleted)
		go m.Start(k.serverDone)
	})

	cluster, err := genericapirequest.ValidClusterFrom(ctx)
	if err != nil {
		return err
	}

	delegate, err := k.getOrCreateDelegate(cluster.Name)
	if err != nil {
		return err
	}

	return delegate.Validate(ctx, a, o)
}

// getOrCreateDelegate creates an actual plugin for clusterName.
func (k *KubeValidatingAdmissionPolicy) getOrCreateDelegate(clusterName logicalcluster.Name) (*stoppableValidatingAdmissionPolicy, error) {
	k.lock.RLock()
	delegate := k.delegates[clusterName]
	k.lock.RUnlock()

	if delegate != nil {
		return delegate, nil
	}

	k.lock.Lock()
	defer k.lock.Unlock()

	delegate = k.delegates[clusterName]
	if delegate != nil {
		return delegate, nil
	}

	// Set up a context that is cancelable and that is bounded by k.serverDone
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Wait for either the context or the server to be done. If it's the server, cancel the context.
		select {
		case <-ctx.Done():
		case <-k.serverDone:
			cancel()
		}
	}()

	plugin := validating.NewPlugin(nil)

	delegate = &stoppableValidatingAdmissionPolicy{
		Plugin: plugin,
		stop:   cancel,
	}

	plugin.SetNamespaceInformer(k.localKubeSharedInformerFactory.Core().V1().Namespaces().Cluster(clusterName))
	plugin.SetExternalKubeClientSet(k.kubeClusterClient.Cluster(clusterName.Path()))

	// TODO(ncdc): this is super inefficient to do per workspace
	discoveryClient := memory.NewMemCacheClient(k.kubeClusterClient.Cluster(clusterName.Path()).Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	plugin.SetRESTMapper(restMapper)

	plugin.SetDynamicClient(k.dynamicClusterClient.Cluster(clusterName.Path()))
	plugin.SetDrainedNotification(ctx.Done())
	plugin.SetAuthorizer(k.authorizer)
	plugin.SetClusterName(clusterName)
	plugin.SetSourceFactory(func(_ informers.SharedInformerFactory, client kubernetes.Interface, dynamicClient dynamic.Interface, restMapper meta.RESTMapper, clusterName logicalcluster.Name) generic.Source[validating.PolicyHook] {
		return generic.NewPolicySource(
			k.globalKubeSharedInformerFactory.Admissionregistration().V1().ValidatingAdmissionPolicies().Informer().Cluster(clusterName),
			k.globalKubeSharedInformerFactory.Admissionregistration().V1().ValidatingAdmissionPolicyBindings().Informer().Cluster(clusterName),
			validating.NewValidatingAdmissionPolicyAccessor,
			validating.NewValidatingAdmissionPolicyBindingAccessor,
			validating.CompilePolicy,
			nil,
			dynamicClient,
			restMapper,
			clusterName,
		)
	})

	if err := plugin.ValidateInitialization(); err != nil {
		cancel()
		return nil, err
	}

	k.delegates[clusterName] = delegate

	return delegate, nil
}

func (k *KubeValidatingAdmissionPolicy) logicalClusterDeleted(clusterName logicalcluster.Name) {
	k.lock.Lock()
	defer k.lock.Unlock()

	delegate := k.delegates[clusterName]

	logger := klog.Background().WithValues("clusterName", clusterName)

	if delegate == nil {
		logger.V(3).Info("received event to stop validating admission policy for logical cluster, but it wasn't in the map")
		return
	}

	logger.V(2).Info("stopping validating admission policy for logical cluster")

	delete(k.delegates, clusterName)
	delegate.stop()
}

type stoppableValidatingAdmissionPolicy struct {
	*validating.Plugin
	stop func()
}
