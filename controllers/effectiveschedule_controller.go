package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/job"
	"github.com/vshn/k8up/scheduler"
)

type (
	// EffectiveScheduleReconciler reconciles a EffectiveSchedule object
	EffectiveScheduleReconciler struct {
		client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}
	EffectiveScheduleReconciliationContext struct {
		Ctx               context.Context
		EffectiveSchedule *k8upv1alpha1.EffectiveSchedule
	}
)

// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules/finalizers,verbs=update

// Reconcile handles reconcile requests for EffectiveSchedule.
func (r *EffectiveScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("effectiveschedule", req.NamespacedName)

	schedule := &k8upv1alpha1.EffectiveSchedule{}
	rc := EffectiveScheduleReconciliationContext{
		Ctx:               ctx,
		EffectiveSchedule: schedule,
	}

	err := r.Client.Get(rc.Ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, rc.EffectiveSchedule)
	if err != nil && !apierrors.IsNotFound(err) {
		r.Log.Info("Failed to fetch resource")
		return ctrl.Result{}, err
	}

	controllerutil.AddFinalizer(rc.EffectiveSchedule, k8upv1alpha1.EffectiveScheduleFinalizer)

	err = r.Client.Update(rc.Ctx, rc.EffectiveSchedule)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(rc.EffectiveSchedule.Spec.EffectiveSchedules) > 0 {
		scheduler.GetScheduler().SyncSchedules(r.createJobList(rc))
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EffectiveScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8upv1alpha1.EffectiveSchedule{}).
		Complete(r)
}

func (r *EffectiveScheduleReconciler) createJobList(rc EffectiveScheduleReconciliationContext) scheduler.JobList {


	list := scheduler.JobList{
		Config: job.NewConfig(rc.Ctx, r.Client, r.Log, rc.EffectiveSchedule, r.Scheme, ""),
	}


	for _, ref := range rc.EffectiveSchedule.Spec.EffectiveSchedules {
		scheduleSpec := &k8upv1alpha1.Schedule{}
		err := r.Client.Get(rc.Ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, scheduleSpec)
		if err != nil {
			r.Log.Info("schedule not found", "error", err.Error())
			continue
		}
		var obj scheduler.ObjectCreator

		switch ref.JobType {
		case k8upv1alpha1.PruneType:
			obj = scheduleSpec.Spec.Prune
		case k8upv1alpha1.CheckType:
			obj = scheduleSpec.Spec.Check
		default:
			continue
		}
		list.Jobs = append(list.Jobs, scheduler.Job{
			JobType: ref.JobType,
			Schedule: ref.Schedule,
			Object: obj,
		})
	}
	return list
}
