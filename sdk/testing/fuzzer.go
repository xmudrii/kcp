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

package testing

import (
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"

	apisfuzzer "github.com/kcp-dev/kcp/sdk/apis/apis/fuzzer"
)

// FuzzerFuncs is a list of fuzzer functions.
var FuzzerFuncs = fuzzer.MergeFuzzerFuncs(
	apisfuzzer.Funcs,
)
