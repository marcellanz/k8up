package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
)

func Test_filterEffectiveSchedule(t *testing.T) {
	tests := map[string]struct {
		givenEffectiveSchedules   k8upv1alpha1.EffectiveScheduleList
		givenSchedule             *k8upv1alpha1.Schedule
		expectedEffectiveSchedule *k8upv1alpha1.EffectiveSchedule
	}{
		"GivenNoSchedules_WhenFilter_ThenReturnNil": {
			expectedEffectiveSchedule: nil,
		},
		"GivenNonMatchingSchedules_WhenFilter_ThenReturnNil": {
			givenEffectiveSchedules: createListWithScheduleRef("not matching", "foreign"),
			givenSchedule: &k8upv1alpha1.Schedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "schedule",
					Namespace: "test",
				},
			},
			expectedEffectiveSchedule: nil,
		},
		"GivenMatchingSchedules_WhenFilter_ThenReturnMatch": {
			givenEffectiveSchedules: createListWithScheduleRef("schedule", "test"),
			givenSchedule: &k8upv1alpha1.Schedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "schedule",
					Namespace: "test",
				},
			},
			expectedEffectiveSchedule: createEffectiveScheduleWithScheduleRef("schedule", "test"),
		},
		"GivenMatchingSchedulesWithDeletion_WhenFilter_ThenReturnNil": {
			givenEffectiveSchedules: k8upv1alpha1.EffectiveScheduleList{
				Items: []k8upv1alpha1.EffectiveSchedule{*createEffectiveScheduleWithScheduleRefAndDeletionDate("schedule", "test")},
			},
			givenSchedule: &k8upv1alpha1.Schedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "schedule",
					Namespace: "test",
				},
			},
			expectedEffectiveSchedule: nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterEffectiveSchedule(tt.givenEffectiveSchedules, tt.givenSchedule)
			assert.Equal(t, tt.expectedEffectiveSchedule, result)
		})
	}
}

func createListWithScheduleRef(name, namespace string) k8upv1alpha1.EffectiveScheduleList {
	return k8upv1alpha1.EffectiveScheduleList{
		Items: []k8upv1alpha1.EffectiveSchedule{
			*createEffectiveScheduleWithScheduleRef(name, namespace),
		},
	}
}

func createEffectiveScheduleWithScheduleRef(name, namespace string) *k8upv1alpha1.EffectiveSchedule {
	return &k8upv1alpha1.EffectiveSchedule{
		Spec: k8upv1alpha1.EffectiveScheduleSpec{
			EffectiveSchedules: []k8upv1alpha1.JobRef{
				{Name: name, Namespace: namespace},
			},
		},
	}
}

func createEffectiveScheduleWithScheduleRefAndDeletionDate(name, namespace string) *k8upv1alpha1.EffectiveSchedule {
	es := createEffectiveScheduleWithScheduleRef(name, namespace)
	time := metav1.Now()
	es.DeletionTimestamp = &time
	return es
}
