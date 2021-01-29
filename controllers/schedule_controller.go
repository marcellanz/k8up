package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/handler"
	"github.com/vshn/k8up/job"
)

// ScheduleReconciler reconciles a Schedule object
type ScheduleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.appuio.ch,resources=schedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=schedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.appuio.ch,resources=effectiveschedules/finalizers,verbs=update

// Reconcile is the entrypoint to manage the given resource.
func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("schedule", req.NamespacedName)

	schedule := &k8upv1alpha1.Schedule{}
	err := r.Client.Get(ctx, req.NamespacedName, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	effectiveSchedule, err := r.FetchEffectiveScheduleResource(ctx, schedule)
	if err != nil {
		r.Log.Info("could not retrieve list of effective schedules, try later again", "error", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: 60 * time.Minute}, err
	}

	repository := cfg.GetGlobalRepository()
	if schedule.Spec.Backend != nil {
		repository = schedule.Spec.Backend.String()
	}
	config := job.NewConfig(ctx, r.Client, log, schedule, r.Scheme, repository)

	return ctrl.Result{}, handler.NewScheduleHandler(config, schedule, effectiveSchedule).Handle()
}

// SetupWithManager configures the reconciler.
func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8upv1alpha1.Schedule{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// FetchEffectiveScheduleResource retrieves a list of EffectiveSchedules and filter the one that matches the given schedule.
// Returns an error if the listing failed, but nil when no matching EffectiveSchedule object was found.
func (r *ScheduleReconciler) FetchEffectiveScheduleResource(ctx context.Context, schedule *k8upv1alpha1.Schedule) (*k8upv1alpha1.EffectiveSchedule, error) {
	list := k8upv1alpha1.EffectiveScheduleList{}
	err := r.Client.List(ctx, &list, client.InNamespace(cfg.Config.OperatorNamespace))
	if err != nil {
		return nil, err
	}
	return filterEffectiveSchedule(list, schedule), nil
}

func filterEffectiveSchedule(list k8upv1alpha1.EffectiveScheduleList, schedule *k8upv1alpha1.Schedule) *k8upv1alpha1.EffectiveSchedule {
	for _, effectiveSchedule := range list.Items {
		if effectiveSchedule.GetDeletionTimestamp() != nil {
			continue
		}
		for _, jobRef := range effectiveSchedule.Spec.EffectiveSchedules {
			if schedule.IsReferencedBy(jobRef) {
				return &effectiveSchedule
			}
		}
	}
	return nil
}
