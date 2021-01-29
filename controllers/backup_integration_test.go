// +build integration

package controllers_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8upv1a1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/controllers"
)

type BackupTestSuite struct {
	EnvTestSuite

	GivenPreBackupPod       *k8upv1a1.PreBackupPod
	GivenBackup             *k8upv1a1.Backup
	BackupName              string
	DeploymentName          string
	DeploymentContainerName string
	PreBackupPodName        string
}

func Test_Backup(t *testing.T) {
	suite.Run(t, new(BackupTestSuite))
}

func (r *BackupTestSuite) Test_GivenBackup_ExpectDeployment() {
	r.givenABackupExists()

	result := r.whenReconciling()
	assert.GreaterOrEqual(r.T(), result.RequeueAfter, 30*time.Second)

	r.expectABackupJobEventually()
}

func (r *BackupTestSuite) Test_GivenPreBackupPods_ExpectDeployment() {
	r.givenAPreBackupPodExists()
	r.givenABackupExists()

	result := r.whenReconciling()

	assert.GreaterOrEqual(r.T(), result.RequeueAfter, 30*time.Second)
	r.expectADeploymentEventually()
}

func (r *BackupTestSuite) Test_GivenPreBackupPods_WhenRestarting() {
	// r.givenAPreBackupPodExists()
	//
	// _ = r.whenReconciling()
	// r.whenCancellingTheContext()
	//
	// r.afterDeploymentIsFinished()
	// r.whenRerunningPreBackup()
	//
	// r.expectABackupContainer()
}

func (r *BackupTestSuite) SetupTest() {
	r.BackupName = "restore-integration-test"
	r.DeploymentName = "pre-backup-pod-deployment"
	r.DeploymentContainerName = "pre-backup-pod-container"
	r.PreBackupPodName = "pre-backup-pod-cr"
}

func (r *BackupTestSuite) newPreBackupPod() *k8upv1a1.PreBackupPod {
	return &k8upv1a1.PreBackupPod{
		Spec: k8upv1a1.PreBackupPodSpec{
			BackupCommand: "/bin/true",
			Pod: &k8upv1a1.Pod{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      r.DeploymentName,
						Namespace: r.NS,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    r.DeploymentContainerName,
								Image:   "alpine",
								Command: []string{"/bin/true"},
							},
						},
					},
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.PreBackupPodName,
			Namespace: r.NS,
		},
	}
}

func (r *BackupTestSuite) newBackup() *k8upv1a1.Backup {
	return &k8upv1a1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.BackupName,
			Namespace: r.NS,
		},
		Spec: k8upv1a1.BackupSpec{
			RunnableSpec: k8upv1a1.RunnableSpec{},
		},
	}
}

func (r *BackupTestSuite) givenAPreBackupPodExists() {
	if r.GivenPreBackupPod == nil {
		r.GivenPreBackupPod = r.newPreBackupPod()
	}

	err := r.Client.Create(r.Ctx, r.GivenPreBackupPod)
	require.NoError(r.T(), err)
}

func (r *BackupTestSuite) givenABackupExists() {
	if r.GivenBackup == nil {
		r.GivenBackup = r.newBackup()
	}

	err := r.Client.Create(r.Ctx, r.GivenBackup)
	require.NoError(r.T(), err)
}

func (r *BackupTestSuite) whenReconciling() controllerruntime.Result {

	controller := controllers.BackupReconciler{
		Client: r.Client,
		Log:    r.Logger,
		Scheme: scheme.Scheme,
	}

	key := types.NamespacedName{
		Namespace: r.NS,
		Name:      r.BackupName,
	}
	request := controllerruntime.Request{
		NamespacedName: key,
	}

	result, err := controller.Reconcile(r.Ctx, request)
	require.NoError(r.T(), err)

	return result
}

func (r *BackupTestSuite) expectABackupJobEventually() {
	r.RepeatedAssert(3*time.Second, time.Second, "Jobs not found", func(timedCtx context.Context) (done bool, err error) {
		jobs := new(batchv1.JobList)
		err = r.Client.List(timedCtx, jobs, &client.ListOptions{Namespace: r.NS})
		require.NoError(r.T(), err)

		jobsLen := len(jobs.Items)
		r.T().Logf("%d Jobs found", jobsLen)

		if jobsLen > 0 {
			assert.Len(r.T(), jobs.Items, 1)
			return true, err
		}

		return
	})
}

func (r *BackupTestSuite) expectADeploymentEventually() {
	r.RepeatedAssert(5*time.Second, time.Second, "Deployments not found", func(timedCtx context.Context) (done bool, err error) {
		deployments := new(appsv1.DeploymentList)
		err = r.Client.List(timedCtx, deployments, &client.ListOptions{Namespace: r.NS})
		require.NoError(r.T(), err)

		jobsLen := len(deployments.Items)
		r.T().Logf("%d Deployments found", jobsLen)

		if jobsLen > 0 {
			assert.Len(r.T(), deployments.Items, 1)
			return true, err
		}

		return
	})
}
