package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	// +kubebuilder:object:root=true
	// +kubebuilder:subresource:status
	// +kubebuilder:printcolumn:name="Schedule Namespace",type="string",JSONPath=`.spec.effectiveSchedules[0].namespace`,description="Schedule Namespace"
	// +kubebuilder:printcolumn:name="Schedule Name",type="string",JSONPath=`.spec.effectiveSchedules[0].name`,description="Schedule Name"
	// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

	// EffectiveSchedule is the Schema to persist schedules generated from Randomized schedules.
	EffectiveSchedule struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   EffectiveScheduleSpec   `json:"spec,omitempty"`
		Status EffectiveScheduleStatus `json:"status,omitempty"`
	}

	// +kubebuilder:object:root=true

	// EffectiveScheduleList contains a list of EffectiveSchedule
	EffectiveScheduleList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []EffectiveSchedule `json:"items"`
	}

	// EffectiveScheduleSpec defines the desired state of EffectiveSchedule
	EffectiveScheduleSpec struct {

		// EffectiveSchedules holds a list of effective schedules. The list may omit entries that aren't generated from
		// smart schedules.
		EffectiveSchedules []JobRef `json:"effectiveSchedules,omitempty"`
	}

	// JobRef represents a reference to a job in a Schedule object
	JobRef struct {
		Name              string             `json:"name,omitempty"`
		Namespace         string             `json:"namespace,omitempty"`
		JobType           JobType            `json:"jobType,omitempty"`
		EffectiveSchedule ScheduleDefinition `json:"effectiveSchedule,omitempty"`
	}

	// EffectiveScheduleStatus defines the observed state of EffectiveSchedule
	EffectiveScheduleStatus struct {
		// Conditions provide a standard mechanism for higher-level status reporting from a controller.
		// They are an extension mechanism which allows tools and other controllers to collect summary information about
		// resources without needing to understand resource-specific status details.
		Conditions []metav1.Condition `json:"conditions,omitempty"`
	}
)

func init() {
	SchemeBuilder.Register(&EffectiveSchedule{}, &EffectiveScheduleList{})
}
