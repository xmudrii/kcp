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

package apibinding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"

	kcpinitializers "github.com/kcp-dev/kcp/pkg/admission/initializers"
	"github.com/kcp-dev/kcp/pkg/authorization/delegated"
	"github.com/kcp-dev/kcp/pkg/indexers"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha2"
	"github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha2/permissionclaims"
	"github.com/kcp-dev/kcp/sdk/apis/core"
	kcpinformers "github.com/kcp-dev/kcp/sdk/client/informers/externalversions"
)

const (
	PluginName = "apis.kcp.io/APIBinding"
)

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName,
		func(_ io.Reader) (admission.Interface, error) {
			p := &apiBindingAdmission{
				Handler:          admission.NewHandler(admission.Create, admission.Update),
				createAuthorizer: delegated.NewDelegatedAuthorizer,
			}
			p.getAPIExport = func(path logicalcluster.Path, name string) (*apisv1alpha2.APIExport, error) {
				return indexers.ByPathAndNameWithFallback[*apisv1alpha2.APIExport](apisv1alpha2.Resource("apiexports"), p.apiExportIndexer, p.cacheAPIExportIndexer, path, name)
			}

			return p, nil
		})
}

type apiBindingAdmission struct {
	*admission.Handler

	getAPIExport func(path logicalcluster.Path, name string) (*apisv1alpha2.APIExport, error)

	apiExportIndexer      cache.Indexer
	cacheAPIExportIndexer cache.Indexer

	deepSARClient    kcpkubernetesclientset.ClusterInterface
	createAuthorizer delegated.DelegatedAuthorizerFactory
}

// Ensure that the required admission interfaces are implemented.
var (
	_ = admission.ValidationInterface(&apiBindingAdmission{})
	_ = admission.MutationInterface(&apiBindingAdmission{})
	_ = admission.InitializationValidator(&apiBindingAdmission{})
	_ = kcpinitializers.WantsDeepSARClient(&apiBindingAdmission{})
	_ = kcpinitializers.WantsKcpInformers(&apiBindingAdmission{})
)

func (o *apiBindingAdmission) Admit(ctx context.Context, a admission.Attributes, _ admission.ObjectInterfaces) error {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	if a.GetResource().GroupResource() != apisv1alpha2.Resource("apibindings") {
		return nil
	}

	u, ok := a.GetObject().(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected type %T", a.GetObject())
	}

	apiBinding := &apisv1alpha2.APIBinding{}
	switch a.GetKind().GroupVersion().Version {
	case apisv1alpha1.SchemeGroupVersion.Version:
		// v1alpha1 is deprecated, but we still need to support it for a while
		// for backward compatibility.

		ab := &apisv1alpha1.APIBinding{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ab); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}

		// Before we convert to v1alpha2, we need to validate the annotations overhanging:
		if err := validateOverhangingPermissionClaims(ctx, a, ab); err != nil {
			return admission.NewForbidden(a, err)
		}

		err := apisv1alpha2.Convert_v1alpha1_APIBinding_To_v1alpha2_APIBinding(ab, apiBinding, nil)
		if err != nil {
			return fmt.Errorf("failed to convert v1alpha1 APIBinding to v1alpha2: %w", err)
		}
	case apisv1alpha2.SchemeGroupVersion.Version:
		// v1alpha2 is the current version.
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, apiBinding); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}
	default:
		return admission.NewForbidden(a,
			field.Invalid(
				field.NewPath("apiVersion"),
				a.GetKind().GroupVersion().String(),
				fmt.Sprintf("unsupported API version %s", a.GetKind().GroupVersion().String())))
	}

	if apiBinding.Spec.Reference.Export == nil {
		// should not happen due to validation.
		return nil
	}

	var oldAPIBinding *apisv1alpha2.APIBinding
	if a.GetOperation() == admission.Update {
		u, ok := a.GetOldObject().(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("unexpected type %T", a.GetObject())
		}

		oldAPIBinding = &apisv1alpha2.APIBinding{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, oldAPIBinding); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}
	}

	switch {
	case a.GetOperation() == admission.Create,
		a.GetOperation() == admission.Update && !reflect.DeepEqual(apiBinding.Spec.Reference, oldAPIBinding.Spec.Reference),
		a.GetOperation() == admission.Update && apiBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey] != oldAPIBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey]:

		// unified forbidden error that does not leak workspace existence
		action := "create"
		if a.GetOperation() == admission.Update {
			action = "update"
		}
		forbidden := admission.NewForbidden(a, fmt.Errorf("unable to %s APIBinding: no permission to bind to export %s", action,
			logicalcluster.NewPath(apiBinding.Spec.Reference.Export.Path).Join(apiBinding.Spec.Reference.Export.Name).String()))

		// get cluster name of export
		var exportClusterName logicalcluster.Name
		if apiBinding.Spec.Reference.Export.Path == "" {
			exportClusterName = clusterName
		} else if apiBinding.Spec.Reference.Export.Path == core.RootCluster.String() {
			// special case to allow bootstrapping
			exportClusterName = core.RootCluster
		} else {
			path := logicalcluster.NewPath(apiBinding.Spec.Reference.Export.Path)
			export, err := o.getAPIExport(path, apiBinding.Spec.Reference.Export.Name)
			if err != nil {
				return forbidden
			}
			exportClusterName = logicalcluster.From(export)
		}

		// set labels
		if apiBinding.Labels == nil {
			apiBinding.Labels = make(map[string]string)
		}
		apiBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey] = permissionclaims.ToAPIBindingExportLabelValue(
			exportClusterName,
			apiBinding.Spec.Reference.Export.Name,
		)
	}

	// convert to original version and write back
	switch a.GetKind().GroupVersion().Version {
	case apisv1alpha1.SchemeGroupVersion.Version:
		v1 := &apisv1alpha1.APIBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apisv1alpha1.SchemeGroupVersion.String(),
				Kind:       "APIBinding",
			},
		}
		if err := apisv1alpha2.Convert_v1alpha2_APIBinding_To_v1alpha1_APIBinding(apiBinding, v1, nil); err != nil {
			return fmt.Errorf("failed to convert v1alpha2 APIBinding to v1alpha1: %w", err)
		}

		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v1)
		if err != nil {
			return err
		}
		u.Object = raw
	case apisv1alpha2.SchemeGroupVersion.Version:
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(apiBinding)
		if err != nil {
			return err
		}
		u.Object = raw
	default: // should not happen as we validated above, but to be on the safe side
		return admission.NewForbidden(a,
			field.Invalid(
				field.NewPath("apiVersion"),
				a.GetKind().GroupVersion().String(),
				fmt.Sprintf("unsupported API version %s", a.GetKind().GroupVersion().String())))
	}

	return nil
}

// Validate validates the creation and updating of APIBinding resources. It also performs a SubjectAccessReview
// making sure the user is allowed to use the 'bind' verb with the referenced APIExport.
func (o *apiBindingAdmission) Validate(ctx context.Context, a admission.Attributes, _ admission.ObjectInterfaces) error {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	if a.GetResource().GroupResource() != apisv1alpha2.Resource("apibindings") {
		return nil
	}

	u, ok := a.GetObject().(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected type %T", a.GetObject())
	}

	apiBinding := &apisv1alpha2.APIBinding{}
	switch a.GetKind().GroupVersion().Version {
	case apisv1alpha1.SchemeGroupVersion.Version:
		// v1alpha1 is deprecated, but we still need to support it for a while
		// for backward compatibility.

		ab := &apisv1alpha1.APIBinding{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ab); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}

		// Before we convert to v1alpha2, we need to validate the annotations overhanging:
		if err := validateOverhangingPermissionClaims(ctx, a, ab); err != nil {
			return admission.NewForbidden(a, err)
		}

		err := apisv1alpha2.Convert_v1alpha1_APIBinding_To_v1alpha2_APIBinding(ab, apiBinding, nil)
		if err != nil {
			return fmt.Errorf("failed to convert v1alpha1 APIBinding to v1alpha2: %w", err)
		}
	case apisv1alpha2.SchemeGroupVersion.Version:
		// v1alpha2 is the current version.
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, apiBinding); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}
	default:
		return admission.NewForbidden(a,
			field.Invalid(
				field.NewPath("apiVersion"),
				a.GetKind().GroupVersion().String(),
				fmt.Sprintf("unsupported API version %s", a.GetKind().GroupVersion().String())))
	}

	// Object validation
	var errs field.ErrorList
	var oldAPIBinding *apisv1alpha2.APIBinding
	switch a.GetOperation() {
	case admission.Create:
		errs = ValidateAPIBinding(apiBinding)
	case admission.Update:
		u, ok = a.GetOldObject().(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("unexpected type %T", a.GetOldObject())
		}
		oldAPIBinding = &apisv1alpha2.APIBinding{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, oldAPIBinding); err != nil {
			return fmt.Errorf("failed to convert unstructured to APIBinding: %w", err)
		}

		errs = ValidateAPIBindingUpdate(oldAPIBinding, apiBinding)
	}
	if len(errs) > 0 {
		return admission.NewForbidden(a, fmt.Errorf("%v", errs))
	}

	switch {
	case a.GetOperation() == admission.Create,
		a.GetOperation() == admission.Update && !reflect.DeepEqual(apiBinding.Spec.Reference, oldAPIBinding.Spec.Reference),
		a.GetOperation() == admission.Update && apiBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey] != oldAPIBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey]:

		// unified forbidden error that does not leak workspace existence
		action := "create"
		if a.GetOperation() == admission.Update {
			action = "update"
		}
		forbidden := admission.NewForbidden(a, fmt.Errorf("unable to %s APIBinding: no permission to bind to export %s", action,
			logicalcluster.NewPath(apiBinding.Spec.Reference.Export.Path).Join(apiBinding.Spec.Reference.Export.Name).String()))

		// get cluster name of export
		var exportClusterName logicalcluster.Name
		if apiBinding.Spec.Reference.Export.Path == "" {
			exportClusterName = clusterName
		} else if apiBinding.Spec.Reference.Export.Path == core.RootCluster.String() {
			// special case to allow bootstrapping
			exportClusterName = core.RootCluster
		} else {
			path := logicalcluster.NewPath(apiBinding.Spec.Reference.Export.Path)
			export, err := o.getAPIExport(path, apiBinding.Spec.Reference.Export.Name)
			if err != nil {
				return forbidden
			}
			exportClusterName = logicalcluster.From(export)
		}

		// Access check
		if err := o.checkAPIExportAccess(ctx, a.GetUserInfo(), exportClusterName, apiBinding.Spec.Reference.Export.Name); err != nil {
			return forbidden
		}

		// Verify the labels
		value := apiBinding.Labels[apisv1alpha1.InternalAPIBindingExportLabelKey]
		if expected := permissionclaims.ToAPIBindingExportLabelValue(
			exportClusterName,
			apiBinding.Spec.Reference.Export.Name,
		); value != expected {
			return admission.NewForbidden(a, field.Invalid(field.NewPath("metadata").Child("labels").Key(apisv1alpha1.InternalAPIBindingExportLabelKey), value, fmt.Sprintf("must be set to %q", expected)))
		}
	}

	return nil
}

func (o *apiBindingAdmission) checkAPIExportAccess(ctx context.Context, user user.Info, apiExportClusterName logicalcluster.Name, apiExportName string) error {
	logger := klog.FromContext(ctx)
	authz, err := o.createAuthorizer(apiExportClusterName, o.deepSARClient, delegated.Options{})
	if err != nil {
		// Logging a more specific error for the operator
		logger.Error(err, "error creating authorizer from delegating authorizer config")
		// Returning a less specific error to the end user
		return errors.New("unable to authorize request")
	}
	return CheckAPIExportAccess(ctx, user, apiExportName, authz)
}

// ValidateInitialization ensures the required injected fields are set.
func (o *apiBindingAdmission) ValidateInitialization() error {
	if o.deepSARClient == nil {
		return fmt.Errorf(PluginName + " plugin needs a deepSARClient")
	}
	if o.apiExportIndexer == nil {
		return fmt.Errorf(PluginName + " plugin needs an APIExport indexer")
	}
	if o.cacheAPIExportIndexer == nil {
		return fmt.Errorf(PluginName + " plugin needs a cache APIExport indexer")
	}
	return nil
}

// SetDeepSARClient is an admission plugin initializer function that injects a client capable of deep SAR requests into
// this admission plugin.
func (o *apiBindingAdmission) SetDeepSARClient(client kcpkubernetesclientset.ClusterInterface) {
	o.deepSARClient = client
}

func (o *apiBindingAdmission) SetKcpInformers(local, global kcpinformers.SharedInformerFactory) {
	apiExportsReady := local.Apis().V1alpha2().APIExports().Informer().HasSynced
	cacheAPIExportsReady := local.Apis().V1alpha2().APIExports().Informer().HasSynced
	o.SetReadyFunc(func() bool {
		return apiExportsReady() && cacheAPIExportsReady()
	})
	o.apiExportIndexer = local.Apis().V1alpha2().APIExports().Informer().GetIndexer()
	o.cacheAPIExportIndexer = global.Apis().V1alpha2().APIExports().Informer().GetIndexer()

	indexers.AddIfNotPresentOrDie(local.Tenancy().V1alpha1().WorkspaceTypes().Informer().GetIndexer(), cache.Indexers{
		indexers.ByLogicalClusterPathAndName: indexers.IndexByLogicalClusterPathAndName,
	})
	indexers.AddIfNotPresentOrDie(global.Tenancy().V1alpha1().WorkspaceTypes().Informer().GetIndexer(), cache.Indexers{
		indexers.ByLogicalClusterPathAndName: indexers.IndexByLogicalClusterPathAndName,
	})
}

func validateOverhangingPermissionClaims(_ context.Context, _ admission.Attributes, ab *apisv1alpha1.APIBinding) error {
	// TODO(xmudrii): Remove this once we are sure that all APIExport objects are
	// converted to v1alpha2.
	if _, ok := ab.Annotations[apisv1alpha2.PermissionClaimsAnnotation]; ok {
		// validate if we can decode overhanging permission claims. If not, we will fail.
		var overhanging []apisv1alpha2.PermissionClaim
		if err := json.Unmarshal([]byte(ab.Annotations[apisv1alpha2.PermissionClaimsAnnotation]), &overhanging); err != nil {
			return field.Invalid(field.NewPath("metadata").Child("annotations").Key(apisv1alpha2.PermissionClaimsAnnotation), ab.Annotations[apisv1alpha2.PermissionClaimsAnnotation], "failed to decode overhanging permission claims")
		}

		// validate mismatches. We could have mismatches between the spec and the annotation
		// (e.g. a resource present in the annotation, but not in the spec).
		// We convert to v2 to check for mismatches.
		v2Claims := make([]apisv1alpha2.PermissionClaim, len(ab.Spec.PermissionClaims))
		for i, v1pc := range ab.Spec.PermissionClaims {
			var v2pc apisv1alpha2.PermissionClaim
			err := apisv1alpha2.Convert_v1alpha1_PermissionClaim_To_v1alpha2_PermissionClaim(&v1pc.PermissionClaim, &v2pc, nil)
			if err != nil {
				return field.Invalid(field.NewPath("spec").Child("permissionClaims").Index(i), ab.Spec.PermissionClaims, "failed to convert spec.PermissionClaims")
			}
			v2Claims = append(v2Claims, v2pc)
		}

		for _, o := range overhanging {
			var found bool
			for _, pc := range v2Claims {
				if pc.EqualGRI(o) {
					found = true

					break
				}
			}
			if !found {
				return field.Invalid(field.NewPath("metadata").Child("annotations").Key(apisv1alpha2.PermissionClaimsAnnotation), ab.Annotations[apisv1alpha2.PermissionClaimsAnnotation], "permission claims defined in annotation do not match permission claims defined in spec")
			}
		}
	}
	return nil
}
