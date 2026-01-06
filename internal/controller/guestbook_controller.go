/*
Copyright 2026.

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
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	webappv1 "com.tomludwig/guestbook/api/v1"
	"github.com/tom-ludwig/operator-scaling/sharding"
)

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Orchestrator *sharding.ShardOrchestrator
}

// +kubebuilder:rbac:groups=webapp.com.tomludwig,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.com.tomludwig,resources=guestbooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapp.com.tomludwig,resources=guestbooks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *GuestbookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Get the pod name from environment variable (set in deployment)
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "unknown"
	}

	// Fetch the Guestbook instance
	var guestbook webappv1.Guestbook
	if err := r.Get(ctx, req.NamespacedName, &guestbook); err != nil {
		// Ignore not-found errors, we'll get a new notification when the object is created
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update status with current reconcile info
	now := metav1.Now()
	guestbook.Status.LastReconcileTime = &now
	guestbook.Status.ProcessedBy = podName
	guestbook.Status.ReconcileCount++

	log.Info("Reconciling Guestbook",
		"name", guestbook.Name,
		"namespace", guestbook.Namespace,
		"processedBy", podName,
		"reconcileCount", guestbook.Status.ReconcileCount,
	)

	// Update the status
	if err := r.Status().Update(ctx, &guestbook); err != nil {
		log.Error(err, "Failed to update Guestbook status")
		return ctrl.Result{}, err
	}

	// Requeue after 30 seconds to demonstrate continuous reconciliation
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Guestbook{}).
		WithEventFilter(predicate.And(
			// GenerationChangedPredicate ignores status-only updates (prevents infinite reconcile loop)
			predicate.GenerationChangedPredicate{},
			// ShardingPredicate ensures only the owning replica processes this resource
			sharding.NewShardingPredicate(r.Orchestrator, nil),
		)).
		Named("guestbook").
		Complete(r)
}
