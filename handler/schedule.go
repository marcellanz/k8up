package handler

import (
	"fmt"

	"github.com/imdario/mergo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8upv1alpha1 "github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/job"
	"github.com/vshn/k8up/scheduler"
)

// ScheduleHandler handles the reconciles for the schedules. Schedules are a special
// type of k8up objects as they will only trigger jobs indirectly.
type ScheduleHandler struct {
	schedule          *k8upv1alpha1.Schedule
	effectiveSchedule *k8upv1alpha1.EffectiveSchedule
	job.Config
	requireStatusUpdate bool
}

// NewScheduleHandler will return a new ScheduleHandler.
func NewScheduleHandler(config job.Config, schedule *k8upv1alpha1.Schedule, effectiveSchedule *k8upv1alpha1.EffectiveSchedule) *ScheduleHandler {
	return &ScheduleHandler{
		schedule:          schedule,
		effectiveSchedule: effectiveSchedule,
		Config:            config,
	}
}

// Handle handles the schedule management. It's responsible for adding and removing the
// jobs from the internal cron library.
func (s *ScheduleHandler) Handle() error {

	namespacedName := types.NamespacedName{Name: s.schedule.GetName(), Namespace: s.schedule.GetNamespace()}

	if s.schedule.GetDeletionTimestamp() != nil {
		controllerutil.RemoveFinalizer(s.schedule, k8upv1alpha1.ScheduleFinalizerName)
		scheduler.GetScheduler().RemoveSchedules(namespacedName)

		return s.updateSchedule()
	}

	var err error

	jobList := s.createJobList()

	err = scheduler.GetScheduler().SyncSchedules(jobList)
	if err != nil {
		s.SetConditionFalseWithMessage(k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonFailed, "cannot add to cron: %v", err.Error())
		return s.updateStatus()
	}

	s.updateEffectiveSchedule()

	if !controllerutil.ContainsFinalizer(s.schedule, k8upv1alpha1.ScheduleFinalizerName) {
		controllerutil.AddFinalizer(s.schedule, k8upv1alpha1.ScheduleFinalizerName)
		return s.updateSchedule()
	}

	return s.updateStatus()
}

func (s *ScheduleHandler) createJobList() scheduler.JobList {
	jobList := scheduler.JobList{
		Config: s.Config,
		Jobs:   make([]scheduler.Job, 0),
	}

	if archive := s.schedule.Spec.Archive; archive != nil {
		jobTemplate := archive.DeepCopy()
		s.mergeWithDefaults(&jobTemplate.RunnableSpec)
		jobType := k8upv1alpha1.ArchiveType
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, jobTemplate.Schedule),
			Object:   jobTemplate.ArchiveSpec,
		})
		s.cleanupEffectiveSchedules(jobType, jobTemplate.Schedule)
	} else {
		s.cleanupEffectiveSchedules(k8upv1alpha1.ArchiveType, "")
	}
	if backup := s.schedule.Spec.Backup; backup != nil {
		backupTemplate := backup.DeepCopy()
		s.mergeWithDefaults(&backupTemplate.RunnableSpec)
		jobType := k8upv1alpha1.BackupType
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, backupTemplate.Schedule),
			Object:   backupTemplate.BackupSpec,
		})
		s.cleanupEffectiveSchedules(jobType, backupTemplate.Schedule)
	} else {
		s.cleanupEffectiveSchedules(k8upv1alpha1.BackupType, "")
	}
	if check := s.schedule.Spec.Check; check != nil {
		checkTemplate := check.DeepCopy()
		s.mergeWithDefaults(&checkTemplate.RunnableSpec)
		jobType := k8upv1alpha1.CheckType
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, checkTemplate.Schedule),
			Object:   checkTemplate.CheckSpec,
		})
		s.cleanupEffectiveSchedules(jobType, checkTemplate.Schedule)
	} else {
		s.cleanupEffectiveSchedules(k8upv1alpha1.CheckType, "")
	}
	if restore := s.schedule.Spec.Restore; restore != nil {
		restoreTemplate := restore.DeepCopy()
		s.mergeWithDefaults(&restoreTemplate.RunnableSpec)
		jobType := k8upv1alpha1.RestoreType
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, restoreTemplate.Schedule),
			Object:   restoreTemplate.RestoreSpec,
		})
		s.cleanupEffectiveSchedules(jobType, restoreTemplate.Schedule)
	} else {
		s.cleanupEffectiveSchedules(k8upv1alpha1.RestoreType, "")
	}
	if prune := s.schedule.Spec.Prune; prune != nil {
		pruneTemplate := prune.DeepCopy()
		s.mergeWithDefaults(&pruneTemplate.RunnableSpec)
		jobType := k8upv1alpha1.PruneType
		jobList.Jobs = append(jobList.Jobs, scheduler.Job{
			JobType:  jobType,
			Schedule: s.getEffectiveSchedule(jobType, pruneTemplate.Schedule),
			Object:   pruneTemplate.PruneSpec,
		})
		s.cleanupEffectiveSchedules(jobType, pruneTemplate.Schedule)
	} else {
		s.cleanupEffectiveSchedules(k8upv1alpha1.PruneType, "")
	}

	return jobList
}

func (s *ScheduleHandler) mergeWithDefaults(specInstance *k8upv1alpha1.RunnableSpec) {
	s.mergeResourcesWithDefaults(specInstance)
	s.mergeBackendWithDefaults(specInstance)
}

func (s *ScheduleHandler) mergeResourcesWithDefaults(specInstance *k8upv1alpha1.RunnableSpec) {
	resources := &specInstance.Resources

	if err := mergo.Merge(resources, s.schedule.Spec.ResourceRequirementsTemplate); err != nil {
		s.Log.Info("could not merge specific resources with schedule defaults", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
	if err := mergo.Merge(resources, cfg.Config.GetGlobalDefaultResources()); err != nil {
		s.Log.Info("could not merge specific resources with global defaults", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
}

func (s *ScheduleHandler) mergeBackendWithDefaults(specInstance *k8upv1alpha1.RunnableSpec) {
	if specInstance.Backend == nil {
		specInstance.Backend = s.schedule.Spec.Backend.DeepCopy()
		return
	}

	if err := mergo.Merge(specInstance.Backend, s.schedule.Spec.Backend); err != nil {
		s.Log.Info("could not merge the schedule's backend with the resource's backend", "err", err.Error(), "schedule", s.Obj.GetMetaObject().GetName(), "namespace", s.Obj.GetMetaObject().GetNamespace())
	}
}

func (s *ScheduleHandler) updateSchedule() error {
	if err := s.Client.Update(s.CTX, s.schedule); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error updating resource %s/%s: %w", s.schedule.Namespace, s.schedule.Name, err)
	}
	return nil
}

func (s *ScheduleHandler) updateStatus() error {
	err := s.Client.Status().Update(s.CTX, s.schedule)
	if err != nil {
		s.Log.Error(err, "Could not update SyncConfig.", "name", s.schedule)
		return err
	}
	s.Log.Info("Updated SyncConfig status.")
	return nil
}

func (s *ScheduleHandler) getEffectiveSchedule(jobType k8upv1alpha1.JobType, originalSchedule k8upv1alpha1.ScheduleDefinition) k8upv1alpha1.ScheduleDefinition {

	if existingSchedule, found := s.findExistingSchedule(jobType); found {
		return existingSchedule
	}

	isStandardOrNotRandom := !originalSchedule.IsNonStandard() || !originalSchedule.IsRandom()
	if isStandardOrNotRandom {
		return originalSchedule
	}

	randomizedSchedule, err := s.createRandomSchedule(jobType, originalSchedule)
	if err != nil {
		s.Log.Info("Could not randomize schedule, continuing with original schedule", "schedule", originalSchedule, "error", err.Error())
		return originalSchedule
	}
	s.setEffectiveSchedule(jobType, randomizedSchedule)
	return randomizedSchedule
}

func (s *ScheduleHandler) findExistingSchedule(jobType k8upv1alpha1.JobType) (k8upv1alpha1.ScheduleDefinition, bool) {
	if s.effectiveSchedule == nil {
		return "", false
	}
	for _, ref := range s.effectiveSchedule.Spec.EffectiveSchedules {
		if !s.schedule.IsReferencedBy(ref) {
			continue
		}
		if ref.JobType == jobType {
			return ref.GeneratedSchedule, true
		}
	}
	return "", false
}

func (s *ScheduleHandler) createRandomSchedule(jobType k8upv1alpha1.JobType, originalSchedule k8upv1alpha1.ScheduleDefinition) (k8upv1alpha1.ScheduleDefinition, error) {
	seed := s.createSeed(s.schedule, jobType)
	randomizedSchedule, err := randomizeSchedule(seed, originalSchedule)
	if err != nil {
		return originalSchedule, err
	}

	s.Log.V(1).Info("Randomized schedule", "seed", seed, "from_schedule", originalSchedule, "effective_schedule", randomizedSchedule)
	return randomizedSchedule, nil
}

func (s *ScheduleHandler) setEffectiveSchedule(jobType k8upv1alpha1.JobType, schedule k8upv1alpha1.ScheduleDefinition) {
	if s.effectiveSchedule == nil {
		s.createNewEffectiveScheduleObj()
	}
	schedules := s.effectiveSchedule.Spec.EffectiveSchedules
	schedules = append(schedules, k8upv1alpha1.JobRef{
		Name:              s.schedule.Name,
		Namespace:         s.schedule.Namespace,
		JobType:           jobType,
		GeneratedSchedule: schedule,
	})
	s.effectiveSchedule.Spec.EffectiveSchedules = schedules
	s.requireStatusUpdate = true
}

func (s *ScheduleHandler) createNewEffectiveScheduleObj() {
	newSchedule := &k8upv1alpha1.EffectiveSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cfg.Config.OperatorNamespace,
			Name:      rand.String(32),
		},
	}
	s.requireStatusUpdate = true
	s.Log.Info("Creating new EffectiveSchedule", "name", k8upv1alpha1.GetNamespacedName(newSchedule))
	err := s.Client.Create(s.CTX, newSchedule)
	if err != nil && !errors.IsAlreadyExists(err) {
		s.Log.Error(err, "could not persist effective schedules", "name", newSchedule.Name)
		s.SetConditionFalseWithMessage(k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonCreationFailed, "could not persist effective schedules: %s", err.Error())
	} else {
		s.SetConditionTrueWithMessage(k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonReady, "effective schedule created: '%s'", k8upv1alpha1.GetNamespacedName(newSchedule))
		s.effectiveSchedule = newSchedule
	}
}

func (s *ScheduleHandler) updateEffectiveSchedule() {
	if s.effectiveSchedule == nil {
		return
	}
	if len(s.effectiveSchedule.Spec.EffectiveSchedules) <= 0 {
		s.deleteEffectiveSchedule()
		return
	}
	err := s.Client.Update(s.CTX, s.effectiveSchedule)
	if err != nil && !errors.IsNotFound(err) {
		s.Log.Error(err, "could not update effective schedules", "name", s.effectiveSchedule.Name)
		s.SetConditionFalseWithMessage(k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonUpdateFailed,
			"could not update effective schedule '%s': %s", k8upv1alpha1.GetNamespacedName(s.schedule), err.Error())
	}
}

// cleanupEffectiveSchedules removes elements in the EffectiveSchedule list that match the job type, but aren't randomized.
// This is needed in case the schedule spec has changed from randomized to standard cron syntax.
func (s *ScheduleHandler) cleanupEffectiveSchedules(jobType k8upv1alpha1.JobType, newSchedule k8upv1alpha1.ScheduleDefinition) {
	if s.effectiveSchedule == nil {
		return
	}
	var schedules []k8upv1alpha1.JobRef
	for _, ref := range s.effectiveSchedule.Spec.EffectiveSchedules {
		if s.schedule.IsReferencedBy(ref) && ref.JobType == jobType && !newSchedule.IsRandom() {
			s.Log.V(1).Info("cleaning out from effective schedule", "type", jobType, "name", k8upv1alpha1.GetNamespacedName(s.schedule))
			continue
		}
		schedules = append(schedules, ref)
	}
	s.effectiveSchedule.Spec.EffectiveSchedules = schedules
}

func (s *ScheduleHandler) deleteEffectiveSchedule() {
	if s.effectiveSchedule == nil {
		return
	}
	s.Log.Info("deleting effective schedule", "name", k8upv1alpha1.GetNamespacedName(s.effectiveSchedule))
	err := s.Client.Delete(s.CTX, s.effectiveSchedule)
	if err != nil {
		s.Log.Info("could not delete effective schedule", "error", err.Error())
	}
	s.SetConditionTrue(k8upv1alpha1.ConditionReady, k8upv1alpha1.ReasonReady)
}
