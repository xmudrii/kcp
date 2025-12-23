/*
Copyright 2025 The KCP Authors.

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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conditionsv1alpha1 "github.com/kcp-dev/sdk/apis/third_party/conditions/apis/conditions/v1alpha1"
)

// +crd
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=kcp,path=filteredapiexportendpointslices,singular=filteredapiexportendpointslice
// +kubebuilder:printcolumn:name="Export",type="string",JSONPath=".spec.export.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// FilteredAPIExportEndpointSlice is a sink for the endpoints of an APIExport. These endpoints can be filtered by a Partition.
// They get consumed by the managers to start controllers and informers for the respective APIExport services.
type FilteredAPIExportEndpointSlice struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec holds the desired state:
	// - the targeted APIExport
	// - the label-based object selector
	Spec FilteredAPIExportEndpointSliceSpec `json:"spec,omitempty"`

	// status communicates the observed state:
	// the filtered list of endpoints for the APIExport service.
	// +optional
	Status FilteredAPIExportEndpointSliceStatus `json:"status,omitempty"`
}

// FilteredAPIExportEndpointSliceSpec defines the desired state of the FilteredAPIExportEndpointSlice.
type FilteredAPIExportEndpointSliceSpec struct {
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="APIExport reference must not be changed"

	// export points to the API export.
	APIExport ExportBindingReference `json:"export"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="APIExport reference must not be changed"

	// objectSelector TBD
	ObjectSelector FilteredAPIExportObjectSelector `json:"objectSelector,omitempty"`
}

type FilteredAPIExportObjectSelector struct {
	// Label selector used to filter objects.
	metav1.LabelSelector `json:",inline"`
}

// APIExportEndpointSliceStatus defines the observed state of APIExportEndpointSlice.
type FilteredAPIExportEndpointSliceStatus struct {
	// +optional

	// conditions is a list of conditions that apply to the FilteredAPIExportEndpointSlice.
	Conditions conditionsv1alpha1.Conditions `json:"conditions,omitempty"`

	// endpoints contains all the URLs of the APIExport service.
	//
	// +optional
	// +listType=map
	// +listMapKey=url
	APIExportEndpoints []FilteredAPIExportEndpoint `json:"endpoints"`
}

// Using a struct provides an extension point

// APIExportEndpoint contains the endpoint information of an APIExport service for a specific shard.
type FilteredAPIExportEndpoint struct {

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:format:URL
	// +required

	// url is an APIExport virtual workspace URL.
	URL string `json:"url"`
}

func (in *FilteredAPIExportEndpointSlice) GetConditions() conditionsv1alpha1.Conditions {
	return in.Status.Conditions
}

func (in *FilteredAPIExportEndpointSlice) SetConditions(conditions conditionsv1alpha1.Conditions) {
	in.Status.Conditions = conditions
}

// These are valid conditions of FilteredAPIExportEndpointSlice in addition to
// APIExportValid and related reasons defined with the APIBinding type.
const (
	// PartitionValid is a condition for FilteredAPIExportEndpointSlice that reflects the validity of the referenced Partition.
	PartitionValid conditionsv1alpha1.ConditionType = "PartitionValid"

	// EndpointURLsReady is a condition for FilteredAPIExportEndpointSlice that reflects the readiness of the URLs.
	// Deprecated: This condition is deprecated and will be removed in a future release.
	APIExportEndpointSliceURLsReady conditionsv1alpha1.ConditionType = "EndpointURLsReady"

	// PartitionInvalidReferenceReason is a reason for the PartitionValid condition of FilteredAPIExportEndpointSlice that the
	// Partition reference is invalid.
	PartitionInvalidReferenceReason = "PartitionInvalidReference"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FilteredAPIExportEndpointSliceList is a list of FilteredAPIExportEndpointSlice resources.
type FilteredAPIExportEndpointSliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FilteredAPIExportEndpointSlice `json:"items"`
}
