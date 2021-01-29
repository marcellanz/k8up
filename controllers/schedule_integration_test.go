// +build integration

package controllers_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/controllers"
)

type (
	ScheduleControllerTestSuite struct {
		EnvTestSuite
		reconciler    *controllers.ScheduleReconciler
		givenSchedule *k8upv1alpha1.Schedule
	}
)

func Test_Schedule(t *testing.T) {
	suite.Run(t, new(ScheduleControllerTestSuite))
}

func (ts *ScheduleControllerTestSuite) BeforeTest(suiteName, testName string) {
	ts.reconciler = &controllers.ScheduleReconciler{
		Client: ts.Client,
		Log:    ts.Logger,
		Scheme: ts.Scheme,
	}
}

func (ts *ScheduleControllerTestSuite) Test_GivenScheduleWithRandomSchedules_WhenChangingToStandardSchedule_ThenCleanupEffectiveSchedule() {
	ts.givenScheduleResource()

	ts.whenReconcile(ts.givenSchedule)

	ts.assertEffectiveSchedule()

	resultSchedule := &k8upv1alpha1.Schedule{}
	ts.FetchResource(k8upv1alpha1.GetNamespacedName(ts.givenSchedule), resultSchedule)
	ts.assertCondition(resultSchedule, k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonReady, "effective schedule created")

	// Change schedule spec
	resultSchedule.Spec.Backup.Schedule = "* * * * *"
	ts.UpdateResources(resultSchedule)

	ts.whenReconcile(resultSchedule)

	resultSchedule = &k8upv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.givenSchedule.Name,
			Namespace: ts.NS,
		},
	}
	ts.FetchResource(k8upv1alpha1.GetNamespacedName(resultSchedule), resultSchedule)
	ts.assertCondition(resultSchedule, k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonReady, "resource is ready")

	newEffectiveSchedules := &k8upv1alpha1.EffectiveScheduleList{}
	ts.FetchResources(newEffectiveSchedules, client.InNamespace(ts.NS))
	ts.Assert().Len(newEffectiveSchedules.Items, 0)
}

func (ts *ScheduleControllerTestSuite) assertEffectiveSchedule() {
	// Assert effective Schedule
	ref := ts.getEffectiveScheduleRef()
	ts.Assert().Equal(ts.givenSchedule.Name, ref.Name)
	ts.Assert().Equal(ts.givenSchedule.Namespace, ref.Namespace)
	ts.Assert().Equal(k8upv1alpha1.BackupType, ref.JobType)
	ts.Assert().False(ref.GeneratedSchedule.IsNonStandard())
}

func (ts *ScheduleControllerTestSuite) assertCondition(resultSchedule *k8upv1alpha1.Schedule, condition k8upv1alpha1.ConditionType, reason k8upv1alpha1.ConditionReason, containsMessage string) {
	c := meta.FindStatusCondition(resultSchedule.Status.Conditions, condition.String())
	ts.Assert().NotNil(c)
	ts.Assert().Equal(reason.String(), c.Reason)
	ts.Assert().Contains(c.Message, containsMessage)
}

func (ts *ScheduleControllerTestSuite) givenScheduleResource() {
	cfg.Config.OperatorNamespace = ts.NS
	givenSchedule := &k8upv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: ts.NS},
		Spec: k8upv1alpha1.ScheduleSpec{
			Backup: &k8upv1alpha1.BackupSchedule{
				ScheduleCommon: &k8upv1alpha1.ScheduleCommon{
					Schedule: "@daily-random",
				},
			},
		},
	}
	ts.EnsureResources(givenSchedule)
	ts.givenSchedule = givenSchedule
}

func (ts *ScheduleControllerTestSuite) whenReconcile(givenSchedule *k8upv1alpha1.Schedule) {
	newResult, err := ts.reconciler.Reconcile(ts.Ctx, ts.ToRequest(givenSchedule))
	ts.Assert().NoError(err)
	ts.Assert().False(newResult.Requeue)
}

func (ts *ScheduleControllerTestSuite) getEffectiveScheduleRef() k8upv1alpha1.JobRef {
	effectiveSchedules := &k8upv1alpha1.EffectiveScheduleList{}
	ts.FetchResources(effectiveSchedules, client.InNamespace(ts.NS))
	ts.Assert().Len(effectiveSchedules.Items, 1)

	return effectiveSchedules.Items[0].Spec.EffectiveSchedules[0]
}
