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

package apis

import (
	"flag"
	"math/rand"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/api/apitesting/roundtrip"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	kubeconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha2"
	sdktesting "github.com/kcp-dev/kcp/sdk/testing"
)

// in non-race-detection mode, a higher number of iterations is reasonable.
const defaultFuzzIters = 200

var FuzzIters = flag.Int("fuzz-iters", defaultFuzzIters, "How many fuzzing iterations to do.")

var groups = []runtime.SchemeBuilder{
	apisv1alpha2.SchemeBuilder,
	apisv1alpha1.SchemeBuilder,
}

// external -> json/protobuf -> external.
func TestRoundTripTypes(t *testing.T) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	for _, builder := range groups {
		require.NoError(t, builder.AddToScheme(scheme))
	}
	seed := rand.Int63()
	fuzzer := fuzzer.FuzzerFor(sdktesting.FuzzerFuncs, rand.NewSource(seed), codecs)
	roundtrip.RoundTripExternalTypesWithoutProtobuf(t, scheme, codecs, fuzzer, nil)
}

type ConversionTests struct {
	name string
	v1   runtime.Object
	v2   runtime.Object
	toV2 func(in, out runtime.Object, s kubeconversion.Scope) error
	toV1 func(in, out runtime.Object, s kubeconversion.Scope) error
}

func TestConversion(t *testing.T) {
	tests := []ConversionTests{
		{
			name: "APIExport",
			v1:   &apisv1alpha1.APIExport{},
			v2:   &apisv1alpha2.APIExport{},
			toV2: func(in, out runtime.Object, s kubeconversion.Scope) error {
				return apisv1alpha2.Convert_v1alpha1_APIExport_To_v1alpha2_APIExport(in.(*apisv1alpha1.APIExport), out.(*apisv1alpha2.APIExport), s)
			},
			toV1: func(in, out runtime.Object, s kubeconversion.Scope) error {
				return apisv1alpha2.Convert_v1alpha2_APIExport_To_v1alpha1_APIExport(in.(*apisv1alpha2.APIExport), out.(*apisv1alpha1.APIExport), s)
			},
		},
		{
			name: "APIExportList",
			v1:   &apisv1alpha1.APIExportList{},
			v2:   &apisv1alpha2.APIExportList{},
			toV2: func(in, out runtime.Object, s kubeconversion.Scope) error {
				return apisv1alpha2.Convert_v1alpha1_APIExportList_To_v1alpha2_APIExportList(in.(*apisv1alpha1.APIExportList), out.(*apisv1alpha2.APIExportList), s)
			},
			toV1: func(in, out runtime.Object, s kubeconversion.Scope) error {
				return apisv1alpha2.Convert_v1alpha2_APIExportList_To_v1alpha1_APIExportList(in.(*apisv1alpha2.APIExportList), out.(*apisv1alpha1.APIExportList), s)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for range *FuzzIters {
				testGenericConversion(t, test.v1.DeepCopyObject, test.v2.DeepCopyObject, test.toV1, test.toV2)
			}
		})
	}
}

func testGenericConversion[V1 runtime.Object, V2 runtime.Object](
	t *testing.T,
	v1Factory func() V1,
	v2Factory func() V2,
	toV1 func(in, out runtime.Object, s kubeconversion.Scope) error,
	toV2 func(in, out runtime.Object, s kubeconversion.Scope) error,
) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	for _, builder := range groups {
		require.NoError(t, builder.AddToScheme(scheme))
	}
	seed := rand.Int63()
	fuzzer := fuzzer.FuzzerFor(sdktesting.FuzzerFuncs, rand.NewSource(seed), codecs)

	t.Run("V1->V2->V1", func(t *testing.T) {
		original := v1Factory()
		result := v1Factory()
		fuzzInternalObject(t, fuzzer, original)
		originalCopy := original.DeepCopyObject()

		intermediate := v2Factory()

		err := toV2(original, intermediate, nil)
		require.NoError(t, err)

		err = toV1(intermediate, result, nil)
		require.NoError(t, err)

		require.True(t, apiequality.Semantic.DeepEqual(original, result), "expects original to equal result")
		require.True(t, apiequality.Semantic.DeepEqual(originalCopy, original), "expects originalCopy to equal original")
	})

	t.Run("V2->V1->V2", func(t *testing.T) {
		original := v2Factory()
		result := v2Factory()
		fuzzInternalObject(t, fuzzer, original)
		originalCopy := original.DeepCopyObject()

		intermediate := v1Factory()

		err := toV1(original, intermediate, nil)
		require.NoError(t, err)

		err = toV2(intermediate, result, nil)
		require.NoError(t, err)

		require.True(t, apiequality.Semantic.DeepEqual(original, result), "expects original to equal result")
		require.True(t, apiequality.Semantic.DeepEqual(originalCopy, original), "expects originalCopy to equal original")
	})
}

// fuzzInternalObject fuzzes an arbitrary runtime object using the appropriate
// fuzzer registered with the apitesting package.
func fuzzInternalObject(t *testing.T, fuzzer *fuzz.Fuzzer, object runtime.Object) runtime.Object {
	fuzzer.Fuzz(object)

	j, err := apimeta.TypeAccessor(object)
	if err != nil {
		t.Fatalf("Unexpected error %v for %#v", err, object)
	}
	j.SetKind("")
	j.SetAPIVersion("")

	return object
}
