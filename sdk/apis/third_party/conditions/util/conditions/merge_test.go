/*
Copyright 2020 The Kubernetes Authors.

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

package conditions

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	conditionsapi "github.com/kcp-dev/kcp/sdk/apis/third_party/conditions/apis/conditions/v1alpha1"

	. "github.com/onsi/gomega"
)

func TestNewConditionsGroup(t *testing.T) {
	g := NewWithT(t)

	conditions := []*conditionsapi.Condition{nil1, true1, true1, falseInfo1, falseWarning1, falseWarning1, falseError1, unknown1}

	got := getConditionGroups(conditionsWithSource(&conditioned{}, conditions...))

	g.Expect(got).ToNot(BeNil())
	g.Expect(got).To(HaveLen(5))

	// The top group should be False/Error and it should have one condition
	g.Expect(got.TopGroup().status).To(Equal(corev1.ConditionFalse))
	g.Expect(got.TopGroup().severity).To(Equal(conditionsapi.ConditionSeverityError))
	g.Expect(got.TopGroup().conditions).To(HaveLen(1))

	// The true group should be true and it should have two conditions
	g.Expect(got.TrueGroup().status).To(Equal(corev1.ConditionTrue))
	g.Expect(got.TrueGroup().severity).To(Equal(conditionsapi.ConditionSeverityNone))
	g.Expect(got.TrueGroup().conditions).To(HaveLen(2))

	// The error group should be False/Error and it should have one condition
	g.Expect(got.ErrorGroup().status).To(Equal(corev1.ConditionFalse))
	g.Expect(got.ErrorGroup().severity).To(Equal(conditionsapi.ConditionSeverityError))
	g.Expect(got.ErrorGroup().conditions).To(HaveLen(1))

	// The warning group should be False/Warning and it should have two conditions
	g.Expect(got.WarningGroup().status).To(Equal(corev1.ConditionFalse))
	g.Expect(got.WarningGroup().severity).To(Equal(conditionsapi.ConditionSeverityWarning))
	g.Expect(got.WarningGroup().conditions).To(HaveLen(2))

	// got[0] should be False/Error and it should have one condition
	g.Expect(got[0].status).To(Equal(corev1.ConditionFalse))
	g.Expect(got[0].severity).To(Equal(conditionsapi.ConditionSeverityError))
	g.Expect(got[0].conditions).To(HaveLen(1))

	// got[1] should be False/Warning and it should have two conditions
	g.Expect(got[1].status).To(Equal(corev1.ConditionFalse))
	g.Expect(got[1].severity).To(Equal(conditionsapi.ConditionSeverityWarning))
	g.Expect(got[1].conditions).To(HaveLen(2))

	// got[2] should be False/Info and it should have one condition
	g.Expect(got[2].status).To(Equal(corev1.ConditionFalse))
	g.Expect(got[2].severity).To(Equal(conditionsapi.ConditionSeverityInfo))
	g.Expect(got[2].conditions).To(HaveLen(1))

	// got[3] should be True and it should have two conditions
	g.Expect(got[3].status).To(Equal(corev1.ConditionTrue))
	g.Expect(got[3].severity).To(Equal(conditionsapi.ConditionSeverityNone))
	g.Expect(got[3].conditions).To(HaveLen(2))

	// got[4] should be Unknown and it should have one condition
	g.Expect(got[4].status).To(Equal(corev1.ConditionUnknown))
	g.Expect(got[4].severity).To(Equal(conditionsapi.ConditionSeverityNone))
	g.Expect(got[4].conditions).To(HaveLen(1))

	// nil conditions are ignored
}

func TestMergeRespectPriority(t *testing.T) {
	tests := []struct {
		name       string
		conditions []*conditionsapi.Condition
		want       *conditionsapi.Condition
	}{
		{
			name:       "aggregate nil list return nil",
			conditions: nil,
			want:       nil,
		},
		{
			name:       "aggregate empty list return nil",
			conditions: []*conditionsapi.Condition{},
			want:       nil,
		},
		{
			name:       "When there is false/error it returns false/error",
			conditions: []*conditionsapi.Condition{falseError1, falseWarning1, falseInfo1, unknown1, true1},
			want:       FalseCondition("foo", "reason falseError1", conditionsapi.ConditionSeverityError, "message falseError1"),
		},
		{
			name:       "When there is false/warning and no false/error, it returns false/warning",
			conditions: []*conditionsapi.Condition{falseWarning1, falseInfo1, unknown1, true1},
			want:       FalseCondition("foo", "reason falseWarning1", conditionsapi.ConditionSeverityWarning, "message falseWarning1"),
		},
		{
			name:       "When there is false/info and no false/error or false/warning, it returns false/info",
			conditions: []*conditionsapi.Condition{falseInfo1, unknown1, true1},
			want:       FalseCondition("foo", "reason falseInfo1", conditionsapi.ConditionSeverityInfo, "message falseInfo1"),
		},
		{
			name:       "When there is true and no false/*, it returns info",
			conditions: []*conditionsapi.Condition{unknown1, true1},
			want:       TrueCondition("foo"),
		},
		{
			name:       "When there is unknown and no true or false/*, it returns unknown",
			conditions: []*conditionsapi.Condition{unknown1},
			want:       UnknownCondition("foo", "reason unknown1", "message unknown1"),
		},
		{
			name:       "nil conditions are ignored",
			conditions: []*conditionsapi.Condition{nil1, nil1, nil1},
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got := merge(conditionsWithSource(&conditioned{}, tt.conditions...), "foo", &mergeOptions{})

			if tt.want == nil {
				g.Expect(got).To(BeNil())
				return
			}
			g.Expect(got).To(HaveSameStateOf(tt.want))
		})
	}
}

func conditionsWithSource(obj Setter, conditions ...*conditionsapi.Condition) []localizedCondition {
	obj.SetConditions(conditionList(conditions...))

	ret := []localizedCondition{}
	for i := range conditions {
		ret = append(ret, localizedCondition{
			Condition: conditions[i],
			Getter:    obj,
		})
	}

	return ret
}
