/*
Copyright 2021.

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
	"time"

	"github.com/CodeMonk123/dlpipe-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DLpipeJobReconciler reconciles a DLpipeJob object
type DLpipeJobReconciler struct {
	client.Client
	KubeClient kubernetes.Interface
	Scheme     *runtime.Scheme
}

//+kubebuilder:rbac:groups=dlpipe.github.com,resources=dlpipejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dlpipe.github.com,resources=dlpipejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dlpipe.github.com,resources=dlpipejobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DLpipeJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *DLpipeJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Enter Reconcile")
	dlpipeJob := v1alpha1.DLpipeJob{}
	err := r.Client.Get(ctx, req.NamespacedName, &dlpipeJob)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// set default status and requeue
	if dlpipeJob.SetDefaultStatus() {
		if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
			logger.Error(err, "update status to \"JobPending\"error")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// check phase and schedule
	if dlpipeJob.Status.JobPhase == v1alpha1.JobPending {
		dlpipeJob.Status.JobPhase = v1alpha1.WaitingForMaster
		if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
			logger.Error(err, "update status to \"WaingForMaster\" error")
			return ctrl.Result{}, err
		}

		err := r.CreateMaster(ctx, &dlpipeJob)
		if err != nil {
			logger.Error(err, "create master error")
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Duration(3 * time.Second)}, nil
	}

	if dlpipeJob.Status.JobPhase == v1alpha1.WaitingForMaster {
		err, ready := r.IsMasterReady(ctx, &dlpipeJob)
		if err != nil {
			logger.Error(err, "error when check master's status")
			return ctrl.Result{}, err
		}
		if !ready {
			return ctrl.Result{RequeueAfter: time.Duration(2 * time.Second)}, nil
		} else {
			dlpipeJob.Status.JobPhase = v1alpha1.MasterIsReady
			err, masterAddr := r.GetMasterAddr(ctx, &dlpipeJob)
			if err != nil {
				logger.Error(err, "error when get master's addr")
				return ctrl.Result{}, err
			}
			dlpipeJob.Status.MasterAddr = masterAddr
			if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
				logger.Error(err, "update status to \"MasterIsReady\" error")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Duration(2 * time.Second)}, nil
		}
	}

	if dlpipeJob.Status.JobPhase == v1alpha1.MasterIsReady {
		dlpipeJob.Status.JobPhase = v1alpha1.JobRunning
		if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
			logger.Error(err, "update status to \"JobRunning\" error")
			return ctrl.Result{}, err
		}

		for i := 1; i < int(*dlpipeJob.Spec.WorldSize); i++ {
			err := r.CreateWorker(ctx, &dlpipeJob, i)
			if err != nil {
				logger.Error(err, fmt.Sprintf("create worker[%d] error", i))
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if dlpipeJob.Status.JobPhase == v1alpha1.JobRunning {
		if r.IsJobFailed(ctx, &dlpipeJob) {
			dlpipeJob.Status.JobPhase = v1alpha1.Failed
			if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
				logger.Error(err, "update status to \"Failed\" error")
				return ctrl.Result{}, err
			}
			// r.DeleteAllPod(ctx, &dlpipeJob)
			return ctrl.Result{}, nil
		}

		if r.IsJobSucceeded(ctx, &dlpipeJob) {
			dlpipeJob.Status.JobPhase = v1alpha1.Completed
			if err := r.Status().Update(ctx, &dlpipeJob); err != nil {
				logger.Error(err, "update status to \"Succeeded\" error")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		return ctrl.Result{RequeueAfter: time.Duration(5 * time.Second)}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DLpipeJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c := ctrl.NewControllerManagedBy(mgr)
	return c.For(&v1alpha1.DLpipeJob{}).Owns(&corev1.Pod{}).Complete(r)
}
