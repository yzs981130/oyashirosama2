/*

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

package controllers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	schedulev1 "openi.cn/api/v1"
)

// TestjobReconciler reconciles a Testjob object
type TestjobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Clock
}

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		fmt.Println("Not Found")
		return nil
	}
	return err
}

// +kubebuilder:rbac:groups=schedule.openi.cn,resources=testjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=schedule.openi.cn,resources=testjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=schedule,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=schedule,resources=jobs/status,verbs=get

var (
	scheduledTimeAnnotation = "openi.cn/scheduled-at"
)

func (r *TestjobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("testjob", req.NamespacedName)

	// get
	testjob := &schedulev1.Testjob{}
	if err := r.Get(ctx, req.NamespacedName, testjob); err != nil {
		//log.Println("unable to fetch testjob")
		log.Error(err, "unable to fetch testJob")
		return ctrl.Result{}, ignoreNotFound(err)
	} else {
		fmt.Println("CPU: ", testjob.Spec.CPU, "Memory: ", testjob.Spec.Memory)
	}

	//list
	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingField(jobOwnerKey, req.Name)); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	//update
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time

	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	getScheduledTimeForJob := func(job *kbatch.Job) (*time.Time, error) {
		timeRaw := job.Annotations[scheduledTimeAnnotation]
		if len(timeRaw) == 0 {
			return nil, nil
		}

		timeParsed, err := time.Parse(time.RFC3339, timeRaw)
		if err != nil {
			return nil, err
		}
		return &timeParsed, nil
	}

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	if mostRecentTime != nil {
		testjob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		testjob.Status.LastScheduleTime = nil
	}
	testjob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		testjob.Status.Active = append(testjob.Status.Active, *jobRef)
	}
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	//testjob.Status.State = schedulev1.JobRun
	if err := r.Status().Update(ctx, testjob); err != nil {
		//fmt.Println(err)
		log.Error(err, "unable to update testjob status")
		return ctrl.Result{}, nil
	}

	//delete
	if testjob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if err := r.Delete(ctx, job); err != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			}
			if int32(i) >= *testjob.Spec.FailedJobsHistoryLimit {
				break
			}
		}
	}

	if testjob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if err := r.Delete(ctx, job); err != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			}
			if int32(i) >= *testjob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
		}
	}

	//get next run
	getNextSchedule := func(testjob *schedulev1.Testjob, now time.Time) (lastMissed *time.Time, next time.Time, err error) {
		sched, err := cron.ParseStandard(testjob.Spec.Schedule)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", testjob.Spec.Schedule, err)
		}

		var earliestTime time.Time
		if testjob.Status.LastScheduleTime != nil {
			earliestTime = testjob.Status.LastScheduleTime.Time
		} else {
			earliestTime = testjob.ObjectMeta.CreationTimestamp.Time
		}
		if testjob.Spec.StartingDeadlineSeconds != nil {
			// controller is not going to schedule anything below this point
			schedulingDeadline := now.Add(-time.Second * time.Duration(*testjob.Spec.StartingDeadlineSeconds))

			if schedulingDeadline.After(earliestTime) {
				earliestTime = schedulingDeadline
			}
		}
		if earliestTime.After(now) {
			return nil, sched.Next(now), nil
		}

		starts := 0
		for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
			lastMissed = &t
			starts++
			if starts > 100 {
				// We can't get the most recent times so just return an empty slice
				return nil, time.Time{}, fmt.Errorf("Too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
			}
		}
		return lastMissed, sched.Next(now), nil
	}

	missedRun, nextRun, err := getNextSchedule(testjob, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		return ctrl.Result{}, nil
	}
	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}
	log = log.WithValues("now", r.Now(), "next run", nextRun)

	//run
	if missedRun == nil {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	log = log.WithValues("current run", missedRun)
	tooLate := false
	if testjob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*testjob.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		return scheduledResult, nil
	}

	if testjob.Spec.ConcurrencyPolicy == schedulev1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}

	if testjob.Spec.ConcurrencyPolicy == schedulev1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			if err := r.Delete(ctx, activeJob); ignoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

	constructJobForTestJob := func(testjob *schedulev1.Testjob, scheduledTime time.Time) (*kbatch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("%s-%d", testjob.Name, scheduledTime.Unix())

		job := &kbatch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   testjob.Namespace,
			},
			Spec: *testjob.Spec.JobTemplate.Spec.DeepCopy(),
		}
		for k, v := range testjob.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
		for k, v := range testjob.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		if err := ctrl.SetControllerReference(testjob, job, r.Scheme); err != nil {
			return nil, err
		}

		return job, nil
	}

	job, err := constructJobForTestJob(testjob, *missedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return scheduledResult, nil
	}

	// ...and create it on the cluster
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for TestJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for TestJob run", "job", job)

	return scheduledResult, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = schedulev1.GroupVersion.String()
)

func (r *TestjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(&kbatch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		job := rawObj.(*kbatch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Testjob
		if owner.APIVersion != apiGVStr || owner.Kind != "Testjob" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulev1.Testjob{}).
		Owns(&kbatch.Job{}).
		Complete(r)
}
