package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	barv1 "your.domain/project/api/v1"
)

const cocktailFinalizer = "cocktails.bar.my.domain/finalizer"

// CocktailReconciler reconciles a Cocktail object
type CocktailReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=bar.my.domain,resources=cocktails,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bar.my.domain,resources=cocktails/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bar.my.domain,resources=cocktails/finalizers,verbs=update

// Reconcile is the main reconciliation loop for Cocktail resources
func (r *CocktailReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Cocktail instance
	cocktail := &barv1.Cocktail{}
	err := r.Get(ctx, req.NamespacedName, cocktail)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			log.Info("Cocktail resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Cocktail")
		return ctrl.Result{}, err
	}

	// Check if the object is being deleted
	if !cocktail.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, cocktail)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cocktail, cocktailFinalizer) {
		controllerutil.AddFinalizer(cocktail, cocktailFinalizer)
		if err := r.Update(ctx, cocktail); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the cocktail
	log.Info("Reconciling Cocktail", "name", cocktail.Name, "recipe", cocktail.Spec.Recipe)

	// Update observed generation
	cocktail.Status.ObservedGeneration = cocktail.Generation

	// Prepare the cocktail
	if err := r.prepareCocktail(ctx, cocktail); err != nil {
		log.Error(err, "Failed to prepare Cocktail")
		r.updateStatus(ctx, cocktail, "Failed", "PreparationError", err.Error())
		return ctrl.Result{}, err
	}

	// Update status to indicate success
	r.updateStatus(ctx, cocktail, "Ready", "Prepared", "Cocktail is ready to serve")

	// Requeue after 5 minutes for freshness check
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// reconcileDelete handles deletion of Cocktail resources
func (r *CocktailReconciler) reconcileDelete(ctx context.Context, cocktail *barv1.Cocktail) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling delete for Cocktail", "name", cocktail.Name)

	if controllerutil.ContainsFinalizer(cocktail, cocktailFinalizer) {
		// Clean up resources (e.g., wash glasses, clean up)
		if err := r.cleanupCocktail(ctx, cocktail); err != nil {
			log.Error(err, "Failed to clean up Cocktail")
			return ctrl.Result{}, err
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(cocktail, cocktailFinalizer)
		if err := r.Update(ctx, cocktail); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}

// prepareCocktail contains the main logic for preparing a cocktail
func (r *CocktailReconciler) prepareCocktail(ctx context.Context, cocktail *barv1.Cocktail) error {
	log := log.FromContext(ctx)

	// Update phase to "Preparing"
	cocktail.Status.Phase = "Preparing"
	r.Status().Update(ctx, cocktail)

	// Simulate preparation time based on recipe
	recipe := cocktail.Spec.Recipe
	preparationTime := r.getPreparationTime(recipe)

	log.Info("Preparing cocktail", "recipe", recipe, "size", cocktail.Spec.Size, "time", preparationTime)

	// In a real operator, you would:
	// 1. Fetch ingredients from inventory
	// 2. Mix components according to recipe
	// 3. Add garnish if requested
	// 4. Verify quality
	// 5. Update status

	// Update status
	cocktail.Status.Phase = "Ready"
	cocktail.Status.ServingsReady = cocktail.Spec.Size
	now := metav1.Now()
	cocktail.Status.LastPrepared = &now

	return nil
}

// getPreparationTime returns the time needed to prepare a cocktail
func (r *CocktailReconciler) getPreparationTime(recipe string) time.Duration {
	switch recipe {
	case "Mojito":
		return time.Minute * 2
	case "Margarita":
		return time.Minute * 3
	case "OldFashioned":
		return time.Minute * 4
	case "Cosmopolitan":
		return time.Minute * 2
	default:
		return time.Minute * 1
	}
}

// cleanupCocktail cleans up resources when a cocktail is deleted
func (r *CocktailReconciler) cleanupCocktail(ctx context.Context, cocktail *barv1.Cocktail) error {
	log := log.FromContext(ctx)
	log.Info("Cleaning up Cocktail", "name", cocktail.Name)

	// In a real operator, you would:
	// 1. Consume remaining cocktail
	// 2. Wash glass and equipment
	// 3. Update inventory

	return nil
}

// updateStatus updates the status of the Cocktail resource
func (r *CocktailReconciler) updateStatus(ctx context.Context, cocktail *barv1.Cocktail, phase string, conditionStatus, reason, message string) {
	// Update phase
	cocktail.Status.Phase = phase

	// Update condition
	cocktail.SetCondition("Ready", metav1.ConditionStatus(conditionStatus), reason, message)

	// Update status
	if err := r.Status().Update(ctx, cocktail); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update status")
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *CocktailReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&barv1.Cocktail{}).
		Complete(r)
}
