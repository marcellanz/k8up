// +build integration

package controllers_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/controllers"
)

type (
	ScheduleControllerTestSuite struct {
		EnvTestSuite
		reconciler *controllers.ScheduleReconciler
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
	ts.NS = ts.SanitizeNameForNS(testName)
	ts.Require().NoError(ts.CreateNS(ts.NS))
}

func (ts *ScheduleControllerTestSuite) Test_FetchEffectiveSchedule_ReturnMatch() {
	cfg.Config.OperatorNamespace = ts.NS
	schedule1 := ts.getEffectiveSchedule("something else")
	schedule2 := ts.getEffectiveSchedule("schedule")
	givenSchedule := &k8upv1alpha1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "schedule",
			Namespace: "foreign",
		},
	}
	ts.CreateResources(schedule1, schedule2)

	result, err := ts.reconciler.FetchEffectiveScheduleResource(ts.Ctx, givenSchedule)
	ts.Assert().NoError(err)
	ts.Assert().NotNil(result)
	ts.Assert().Equal(schedule2.Spec.EffectiveSchedules, result.Spec.EffectiveSchedules)
}

func (ts *ScheduleControllerTestSuite) getEffectiveSchedule(scheduleRefName string) *k8upv1alpha1.EffectiveSchedule {
	return &k8upv1alpha1.EffectiveSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(5),
			Namespace: ts.NS,
		},
		Spec: k8upv1alpha1.EffectiveScheduleSpec{
			EffectiveSchedules: []k8upv1alpha1.JobRef{
				{Name: scheduleRefName, Namespace: "foreign"},
			},
		},
	}
}
