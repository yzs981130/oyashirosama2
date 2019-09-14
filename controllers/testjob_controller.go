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
	"log"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	schedulev1 "openi.cn/api/v1"
)

// TestjobReconciler reconciles a Testjob object
type TestjobReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=schedule.openi.cn,resources=testjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=schedule.openi.cn,resources=testjobs/status,verbs=get;update;patch

func (r *TestjobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("testjob", req.NamespacedName)

	// your logic here
	testjob := &schedulev1.Testjob{}
	if err := r.Get(ctx, req.NamespacedName, testjob); err != nil {
		log.Println("unable to fetch testjob")
	} else {
		fmt.Println(testjob.Spec.CPU, testjob.Spec.Memory)
	}

	testjob.Status.State = schedulev1.JobRun
	if err := r.Status().Update(ctx, testjob); err != nil {
		fmt.Println(err)
		log.Println("unable to update testjob status")
	}

	return ctrl.Result{}, nil
}

func (r *TestjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulev1.Testjob{}).
		Complete(r)
}
