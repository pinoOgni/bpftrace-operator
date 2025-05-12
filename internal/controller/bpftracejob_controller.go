/*
Copyright 2025 pinoOgni.

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

package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tracingv1alpha1 "github.com/pinoOgni/bpftrace-operator/api/v1alpha1"
)

const (
	DefaultAction  = `printf("comm: %s, probe: %s\n", comm, probe);`
	PhaseRunning   = "Running"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)

var runningJobs sync.Map // maps NamespacedName.String() to cancel functions

// BpfTraceJobReconciler reconciles a BpfTraceJob object
type BpfTraceJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tracing.dev.local,resources=bpftracejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tracing.dev.local,resources=bpftracejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tracing.dev.local,resources=bpftracejobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BpfTraceJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *BpfTraceJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	key := req.String()
	bpfTraceJob := tracingv1alpha1.BpfTraceJob{}
	var script string

	err := r.Get(ctx, req.NamespacedName, &bpfTraceJob)

	if err != nil {
		if apierrors.IsNotFound(err) {
			if cancel, ok := runningJobs.Load(key); ok {
				cancel.(context.CancelFunc)()
				runningJobs.Delete(key)
			}
		}
		l.Info("BpfTraceJob is deleted. We can stop the related bpftrace process", "namespaceName", req.String())
		// Some other error
		return ctrl.Result{}, nil
	}
	l.Info("BpfTraceJob Hook", "hook", bpfTraceJob.Spec.Hook)

	if _, exists := runningJobs.Load(key); !exists {
		ctxBpf, cancel := context.WithCancel(context.Background())
		runningJobs.Store(key, cancel)

		// Update status to "Running"
		r.updateStatusPhase(ctx, &bpfTraceJob, PhaseRunning)

		go func(job tracingv1alpha1.BpfTraceJob, key string, namespacedName types.NamespacedName) {
			if job.Spec.Action != "" {
				script = generateScript(job.Spec.Hook, job.Spec.Action)
			} else {
				script = generateScript(job.Spec.Hook, DefaultAction)
			}
			err := runBpftrace(ctxBpf, script)
			// Fetch the latest CR instance before updating status
			var freshJob tracingv1alpha1.BpfTraceJob
			if getErr := r.Get(context.Background(), namespacedName, &freshJob); getErr != nil {
				if !apierrors.IsNotFound(getErr) {
					log.FromContext(ctx).Error(getErr, "failed to get fresh CR for status update")
				}
				return
			}

			if err != nil {
				r.updateStatusPhase(context.Background(), &freshJob, PhaseFailed)
			}

			// Clean up
			runningJobs.Delete(key)

		}(bpfTraceJob, key, req.NamespacedName)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BpfTraceJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tracingv1alpha1.BpfTraceJob{}).
		Named("bpftracejob").
		Complete(r)
}

func generateScript(hook, action string) string {
	return fmt.Sprintf("%s {\n%s\n}\n", hook, action)
}

func runBpftrace(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "bpftrace", "-e", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *BpfTraceJobReconciler) updateStatusPhase(ctx context.Context, job *tracingv1alpha1.BpfTraceJob, phase string) {
	if job.Status.Phase != phase {
		job.Status.Phase = phase
		err := r.Status().Update(ctx, job)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to update status")
		}
	}
}
