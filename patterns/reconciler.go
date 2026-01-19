package patterns

// Reconciler Pattern
//
// This file shows the complete pattern for implementing a Kubernetes controller reconciler.
// The reconciler is responsible for making the actual state match the desired state.

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MyResourceReconciler reconciles a MyResource object
type MyResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// IMPORTANT: Add RBAC markers for permissions needed
// The reconciler needs these permissions to operate

// +kubebuilder:rbac:groups=mygroup.my.domain,resources=myresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mygroup.my.domain,resources=myresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mygroup.my.domain,resources=myresources/finalizers,verbs=update

// Add permissions for resources you manage
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop
// IMPORTANT: This method is called for every event on watched resources
func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// STEP 1: Fetch the MyResource instance
	instance := &MyResource{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			log.Info("MyResource resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get MyResource")
		return ctrl.Result{}, err
	}

	// STEP 2: Check if the object is being deleted
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		return r.reconcileDelete(ctx, instance)
	}

	// STEP 3: Add finalizer if not present
	// IMPORTANT: This ensures we can clean up external resources before deletion
	finalizerName := "myresource.my.domain/finalizer"
	if !controllerutil.ContainsFinalizer(instance, finalizerName) {
		controllerutil.AddFinalizer(instance, finalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// STEP 4: Reconcile the actual state with desired state
	// This is where your business logic goes
	log.Info("Reconciling MyResource", "name", instance.Name)

	// Update observed generation
	instance.Status.ObservedGeneration = instance.Generation

	// Call the main reconcile logic
	if err := r.reconcileLogic(ctx, instance); err != nil {
		log.Error(err, "Failed to reconcile MyResource")
		r.updateStatus(ctx, instance, metav1.ConditionFalse, "ReconcileError", err.Error())
		return ctrl.Result{}, err
	}

	// STEP 5: Update status to indicate success
	r.updateStatus(ctx, instance, metav1.ConditionTrue, "Ready", "MyResource is ready")

	// STEP 6: Determine if we should requeue
	// Return with RequeueAfter for periodic reconciliation (e.g., polling external systems)
	// Return without requeue if everything is stable
	return ctrl.Result{RequeueAfter: r.getRequeueInterval(instance)}, nil
}

// reconcileDelete handles deletion of the resource
func (r *MyResourceReconciler) reconcileDelete(ctx context.Context, instance *MyResource) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling delete for MyResource", "name", instance.Name)

	finalizerName := "myresource.my.domain/finalizer"

	if controllerutil.ContainsFinalizer(instance, finalizerName) {
		// Clean up external resources
		if err := r.cleanupExternalResources(ctx, instance); err != nil {
			log.Error(err, "Failed to clean up external resources")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(instance, finalizerName)
		if err := r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}

// reconcileLogic contains the main business logic
func (r *MyResourceReconciler) reconcileLogic(ctx context.Context, instance *MyResource) error {
	log := log.FromContext(ctx)

	// Example: Create or update a Deployment
	deployment := r.constructDeployment(instance)

	// Create the Deployment if it doesn't exist
	op, err := controllerutil.CreateOrPatch(ctx, r.Client, deployment, func() error {
		// Update deployment with desired spec
		deployment.Spec = *constructDeploymentSpec(instance)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update deployment: %w", err)
	}

	log.Info("Deployment reconciled", "operation", op)

	// Example: Update status based on deployment state
	instance.Status.ReadyReplicas = deployment.Status.ReadyReplicas

	return nil
}

// updateStatus updates the status of the resource
func (r *MyResourceReconciler) updateStatus(ctx context.Context, instance *MyResource, status metav1.ConditionStatus, reason, message string) {
	instance.SetCondition("Ready", status, reason, message)

	now := metav1.Now()
	instance.Status.LastUpdated = &now

	if err := r.Status().Update(ctx, instance); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update status")
	}
}

// getRequeueInterval returns the duration before next reconciliation
func (r *MyResourceReconciler) getRequeueInterval(instance *MyResource) time.Duration {
	// Example: Requeue more frequently if not ready
	if instance.IsReady() {
		return 5 * time.Minute
	}
	return 30 * time.Second
}

// constructDeployment creates a Deployment object from the MyResource spec
func (r *MyResourceReconciler) constructDeployment(instance *MyResource) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}
		controllerutil.SetControllerReference(instance, dep, r.Scheme)
	return dep
}

// constructDeploymentSpec creates the Deployment spec from MyResource
func constructDeploymentSpec(instance *MyResource) *appsv1.DeploymentSpec {
	return &appsv1.DeploymentSpec{
		Replicas: &instance.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": instance.Name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": instance.Name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: instance.Spec.Image,
					},
				},
			},
		},
	}
}

// cleanupExternalResources cleans up external resources created by the operator
func (r *MyResourceReconciler) cleanupExternalResources(ctx context.Context, instance *MyResource) error {
	log := log.FromContext(ctx)
	log.Info("Cleaning up external resources for MyResource", "name", instance.Name)

	// Example: Delete owned resources
	// Kubernetes garbage collector will handle resources with owner references
	// But you might need to manually clean up external resources (API calls, cloud resources, etc.)

	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *MyResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&MyResource{}).
		// Watch for changes in owned resources
		// Example: Watch Deployments created by this controller
		// Owns(&appsv1.Deployment{}).
		// Watch for changes in dependent resources
		// Example: Watch ConfigMaps referenced in spec
		// Watches(
		// 	&source.Kind{Type: &corev1.ConfigMap{}},
		// 	handler.EnqueueRequestsFromMapFunc(r.findConfigMaps),
		// ).
		Complete(r)
}

// findConfigMaps finds MyResources that reference a ConfigMap
func (r *MyResourceReconciler) findConfigMaps(ctx context.Context, o client.Object) []reconcile.Request {
	configMap := o.(*corev1.ConfigMap)
	log := log.FromContext(ctx)

	// Find all MyResources that reference this ConfigMap
	var list MyResourceList
	if err := r.List(ctx, &list, client.InNamespace(configMap.Namespace)); err != nil {
		log.Error(err, "Failed to list MyResources")
		return nil
	}

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
