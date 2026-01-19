package patterns

// Advanced Reconciler Patterns
//
// This file demonstrates advanced patterns for building production-ready Kubernetes operators.
// These patterns go beyond basic reconciliation to handle real-world scenarios.

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// MyResourceReconciler reconciles a MyResource object
type MyResourceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// ==============================================================================
// PATTERN 1: Controller Options and Leader Election
// ==============================================================================

// SetupWithManagerWithOptions demonstrates configuring controller with advanced options
func (r *MyResourceReconciler) SetupWithManagerWithOptions(mgr ctrl.Manager) error {
	// Configure controller with advanced options
	return ctrl.NewControllerManagedBy(mgr).
		For(&MyResource{}).
		// OPTIONS 1: Max Concurrent Reconciles
		// Controls how many reconciles can happen in parallel
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 4, // Allow up to 4 concurrent reconciliations
			// CacheSyncTimeout: time.Minute * 5, // How long to wait for cache to sync
			// RecoverPanic: true, // Recover from panics and log them instead of crashing
		}).
		// OPTIONS 2: Event Filter
		// Only process events that match specific criteria
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		// OPTIONS 3: Filter by Annotation
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			// Skip reconciliation if paused annotation is set
			return !obj.GetAnnotations()["my.domain/paused"] == "true"
		})).
		Complete(r)
}

// ==============================================================================
// PATTERN 2: Complex Watch Configurations
// ==============================================================================

// SetupWithManagerWatches demonstrates watching related resources
func (r *MyResourceReconciler) SetupWithManagerWatches(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&MyResource{}).
		// WATCH 1: Watch owned resources (automatic reconciliation)
		// When Deployment changes, trigger reconciliation of owner MyResource
		// Owns(&appsv1.Deployment{}).

		// WATCH 2: Watch ConfigMaps referenced in spec
		// Uses EnqueueRequestsFromMapFunc to find affected MyResources
		Watches(
			&source.Kind{Type: &v1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			// Optional: Add predicate to filter events
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		// WATCH 3: Watch Secrets referenced in spec
		Watches(
			&source.Kind{Type: &v1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
		).
		Complete(r)
}

// findObjectsForConfigMap finds MyResources that reference a ConfigMap
func (r *MyResourceReconciler) findObjectsForConfigMap(ctx context.Context, o client.Object) []reconcile.Request {
	configMap := o.(*v1.ConfigMap)
	log := log.FromContext(ctx)

	// List all MyResources in the same namespace
	var list MyResourceList
	if err := r.List(ctx, &list, client.InNamespace(configMap.Namespace)); err != nil {
		log.Error(err, "Failed to list MyResources")
		return nil
	}

	// Find MyResources that reference this ConfigMap
	var requests []reconcile.Request
	for _, item := range list.Items {
		if item.Spec.ConfigMapName == configMap.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.Name,
					Namespace: item.Namespace,
				},
			})
		}
	}

	return requests
}

// findObjectsForSecret finds MyResources that reference a Secret
func (r *MyResourceReconciler) findObjectsForSecret(ctx context.Context, o client.Object) []reconcile.Request {
	secret := o.(*v1.Secret)
	log := log.FromContext(ctx)

	var list MyResourceList
	if err := r.List(ctx, &list, client.InNamespace(secret.Namespace)); err != nil {
		log.Error(err, "failed to list MyResources")
		return nil
	}

	var requests []reconcile.Request
	for _, item := range list.Items {
		if item.Spec.SecretName == secret.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.Name,
					Namespace: item.Namespace,
				},
			})
		}
	}

	return requests
}

// ==============================================================================
// PATTERN 3: Selective Reconciliation
// ==============================================================================

// ReconcileWithSkip demonstrates skipping reconciliation based on conditions
func (r *MyResourceReconciler) ReconcileWithSkip(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	instance := &MyResource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// SKIP 1: Check for paused annotation
	if instance.Annotations["my.domain/paused"] == "true" {
		log.Info("Reconciliation paused via annotation")
		r.Recorder.Event(instance, v1.EventTypeNormal, "Paused", "Reconciliation is paused")
		return ctrl.Result{}, nil
	}

	// SKIP 2: Check if already in desired state
	if instance.Status.ObservedGeneration == instance.Generation && instance.IsReady() {
		log.Info("Resource is up-to-date and ready, skipping reconciliation")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil // Recheck in 5 minutes
	}

	// SKIP 3: Check if deletion timestamp is set
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance)
	}

	// Proceed with normal reconciliation
	return r.reconcileNormal(ctx, instance)
}

// ==============================================================================
// PATTERN 4: Retry with Exponential Backoff
// ==============================================================================

// ReconcileWithRetry demonstrates retry logic with exponential backoff
func (r *MyResourceReconciler) ReconcileWithRetry(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	instance := &MyResource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Track retry count in annotation
	retryCount := 0
	if rc, ok := instance.Annotations["my.domain/retryCount"]; ok {
		fmt.Sscanf(rc, "%d", &retryCount)
	}

	// Attempt reconciliation
	err := r.reconcileLogic(ctx, instance)
	if err != nil {
		// Calculate backoff: 2^retryCount seconds, max 5 minutes
		backoff := time.Duration(1<<uint(retryCount)) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}

		// Increment retry count
		retryCount++
		if instance.Annotations == nil {
			instance.Annotations = make(map[string]string)
		}
		instance.Annotations["my.domain/retryCount"] = fmt.Sprintf("%d", retryCount)
		if updateErr := r.Update(ctx, instance); updateErr != nil {
			log.Error(updateErr, "failed to update retry count")
		}

		// Check if we should give up
		if retryCount > 10 {
			log.Error(err, "max retries exceeded, giving up")
			r.Recorder.Event(instance, v1.EventTypeWarning, "ReconciliationFailed", "Max retries exceeded")
			return ctrl.Result{}, err
		}

		log.Error(err, "reconciliation failed, retrying", "retry", retryCount, "backoff", backoff)
		return ctrl.Result{RequeueAfter: backoff}, nil
	}

	// Success - reset retry count
	if retryCount > 0 {
		instance.Annotations["my.domain/retryCount"] = "0"
		_ = r.Update(ctx, instance)
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// ==============================================================================
// PATTERN 5: Patch Strategy for Status Updates
// ==============================================================================

// UpdateStatusWithPatch demonstrates using patch for status updates to avoid conflicts
func (r *MyResourceReconciler) UpdateStatusWithPatch(ctx context.Context, instance *MyResource) error {
	// Create a patch for status updates
	patch := client.MergeFrom(instance.DeepCopy())

	// Update status fields
	instance.Status.Phase = "Ready"
	instance.Status.ObservedGeneration = instance.Generation
	instance.SetCondition("Available", metav1.ConditionTrue, "Ready", "Resource is ready")

	// Use Status().Patch() instead of Status().Update() to avoid conflicts
	if err := r.Status().Patch(ctx, instance, patch); err != nil {
		return fmt.Errorf("failed to patch status: %w", err)
	}

	return nil
}

// ==============================================================================
// PATTERN 6: Conflict Resolution
// ==============================================================================

// ReconcileWithConflictHandling demonstrates proper conflict handling
func (r *MyResourceReconciler) ReconcileWithConflictHandling(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	instance := &MyResource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Attempt update with conflict handling
	err := r.UpdateWithRetry(ctx, instance, func() error {
		// Make changes to instance
		instance.Annotations["my.domain/lastReconcile"] = time.Now().Format(time.RFC3339)
		return nil
	})

	if err != nil {
		if errors.IsConflict(err) {
			// Conflict occurred - requeue immediately
			log.Info("conflict detected, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}
		// Other error - return with error to trigger retry
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// UpdateWithRetry retries updates on conflict
func (r *MyResourceReconciler) UpdateWithRetry(ctx context.Context, obj client.Object, mutate func() error) error {
	return utilretry.RetryOnConflict(utilretry.DefaultRetry, func() error {
		// Get latest version
		if err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return err
		}

		// Apply mutations
		if err := mutate(); err != nil {
			return err
		}

		// Update
		return r.Update(ctx, obj)
	})
}

// ==============================================================================
// PATTERN 7: Rate Limiting and Work Queue
// ==============================================================================

// SetupWithRateLimiter demonstrates custom rate limiting
func (r *MyResourceReconciler) SetupWithRateLimiter(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&MyResource{}).
		// Custom rate limiter: exponential with max 1000s base
		WithControllerControllerOptions(controller.Options{
			RateLimiter: controller.NewControllerRateLimiter(
				// Items: exponentially increase from 5ms to 1000s
				workqueue.NewExponentialFastSlowRateLimiter(5*time.Millisecond, 1000*time.Second),
			),
		}).
		Complete(r)
}

// ==============================================================================
// PATTERN 8: Event Recording
// ==============================================================================

// ReconcileWithEvents demonstrates using event recorder
func (r *MyResourceReconciler) ReconcileWithEvents(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	instance := &MyResource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Record normal event
	r.Recorder.Event(instance, v1.EventTypeNormal, "Reconciling", "Starting reconciliation")

	// Perform reconciliation
	if err := r.reconcileLogic(ctx, instance); err != nil {
		// Record warning event
		r.Recorder.Event(instance, v1.EventTypeWarning, "ReconciliationFailed", err.Error())
		return ctrl.Result{}, err
	}

	// Record success event
	r.Recorder.Event(instance, v1.EventTypeNormal, "ReconciliationSucceeded", "Resource reconciled successfully")
	log.Info("reconciled successfully", "name", instance.Name)

	return ctrl.Result{}, nil
}

// ==============================================================================
// PATTERN 9: Status Aggregation from Multiple Sources
// ==============================================================================

// AggregateStatus demonstrates aggregating status from multiple sources
func (r *MyResourceReconciler) AggregateStatus(ctx context.Context, instance *MyResource) error {
	// Get deployment status
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployment); err != nil {
		return err
	}

	// Get service status
	service := &v1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service); err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Aggregate conditions
	ready := true
	reasons := []string{}

	// Check deployment
	if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
		ready = false
		reasons = append(reasons, fmt.Sprintf("Deployment not ready: %d/%d replicas", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas))
	}

	// Check service
	if service != nil && !service.Status.Conditions.IsTrue() {
		ready = false
		reasons = append(reasons, "Service not ready")
	}

	// Update aggregated status
	if ready {
		instance.SetCondition("Ready", metav1.ConditionTrue, "AllComponentsReady", "All components are ready")
	} else {
		reason := "ComponentsNotReady"
		message := fmt.Sprintf("Waiting for components: %v", reasons)
		instance.SetCondition("Ready", metav1.ConditionFalse, reason, message)
	}

	return r.Status().Update(ctx, instance)
}

// ==============================================================================
// PATTERN 10: Custom Predicate Filtering
// ==============================================================================

// SetupWithCustomPredicate demonstrates custom event filtering
func (r *MyResourceReconciler) SetupWithCustomPredicate(mgr ctrl.Manager) error {
	// Create custom predicate
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Process all create events
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only process if spec changed
			oldObj := e.ObjectOld.(*MyResource)
			newObj := e.ObjectNew.(*MyResource)
			return !oldObj.Spec.Equal(&newObj.Spec)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Process all delete events
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Process all generic events
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&MyResource{}).
		WithEventFilter(pred).
		Complete(r)
}

// ==============================================================================
// Helper Functions
// ==============================================================================

func (r *MyResourceReconciler) reconcileDelete(ctx context.Context, instance *MyResource) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling delete", "name", instance.Name)

	finalizerName := "myresource.my.domain/finalizer"

	if controllerutil.ContainsFinalizer(instance, finalizerName) {
		if err := r.cleanupExternalResources(ctx, instance); err != nil {
			log.Error(err, "cleanup failed")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(instance, finalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MyResourceReconciler) reconcileNormal(ctx context.Context, instance *MyResource) (ctrl.Result, error) {
	// Normal reconciliation logic
	return ctrl.Result{}, nil
}

func (r *MyResourceReconciler) cleanupExternalResources(ctx context.Context, instance *MyResource) error {
	return nil
}

func (r *MyResourceReconciler) reconcileLogic(ctx context.Context, instance *MyResource) error {
	return nil
}

// ==============================================================================
// NOTES:
//
// 1. Leader Election: Enable in main.go with --leader-elect flag
// 2. Metrics: Available at :8080/metrics by default
// 3. Health Probes: Available at :8081/healthz and :8081/readyz
// 4. Event Recorder: Injected into reconciler, events appear in kubectl describe
// 5. Work Queue: Automatically managed by controller-runtime
// 6. Caching: Watched resources are cached for performance
//
// WHEN TO USE EACH PATTERN:
//
// - Controller Options: Tuning performance and concurrency
// - Complex Watches: When you need to reconcile based on related resources
// - Selective Reconciliation: Skip unnecessary work for efficiency
// - Retry with Backoff: Handle transient failures gracefully
// - Patch Strategy: Avoid conflicts in high-update scenarios
// - Conflict Resolution: Handle concurrent modifications
// - Rate Limiting: Control reconciliation rate for external API calls
// - Event Recording: Provide visibility into reconciliation
// - Status Aggregation: When managing multiple child resources
// - Custom Predicates: Filter events to reduce unnecessary reconciliations
//
// ==============================================================================

// Import statements needed (not imported to avoid circular dependencies in this example):
// import (
// 	"sigs.k8s.io/controller-runtime/pkg/manager"
// 	"sigs.k8s.io/controller-runtime/pkg/controller"
// 	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
// 	"sigs.k8s.io/controller-runtime/pkg/event"
// 	"sigs.k8s.io/controller-runtime/pkg/handler"
// 	"sigs.k8s.io/controller-runtime/pkg/predicate"
// 	"sigs.k8s.io/controller-runtime/pkg/source"
// 	"sigs.k8s.io/controller-runtime/pkg/reconcile"
// 	"k8s.io/client-go/util/retry"
// 	"k8s.io/apimachinery/pkg/util/wait"
// 	"k8s.io/client-go/util/workqueue"
// 	appsv1 "k8s.io/api/apps/v1"
// )
