package patterns

// Webhook Pattern
//
// This file shows patterns for implementing admission webhooks
// Webhooks allow you to validate and mutate resources before they are persisted

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// VALIDATION WEBHOOK PATTERN
// ==========================

// MyResourceValidator validates MyResource objects
type MyResourceValidator struct {
	Client  client.Client
	Decoder *admission.Decoder
}

// Handle handles admission requests for validation
func (v *MyResourceValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := log.FromContext(ctx)

	// Decode the object
	instance := &MyResource{}
	err := v.Decoder.Decode(req, instance)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Validate the object
	if err := v.validateMyResource(ctx, instance); err != nil {
		log.Error(err, "Validation failed for MyResource", "name", instance.Name)
		return admission.Denied(err.Error())
	}

	log.Info("Validation passed for MyResource", "name", instance.Name)
	return admission.Allowed("")
}

// validateMyResource contains the validation logic
func (v *MyResourceValidator) validateMyResource(ctx context.Context, instance *MyResource) error {
	// Example: Validate replicas
	if instance.Spec.Replicas < 0 || instance.Spec.Replicas > 100 {
		return fmt.Errorf("replicas must be between 0 and 100, got %d", instance.Spec.Replicas)
	}

	// Example: Validate image is not empty
	if instance.Spec.Image == "" {
		return fmt.Errorf("image must be specified")
	}

	// Example: Validate image format
	if !isValidImageReference(instance.Spec.Image) {
		return fmt.Errorf("invalid image reference: %s", instance.Spec.Image)
	}

	// Example: Validate that referenced ConfigMap exists
	if instance.Spec.ConfigMapName != "" {
		configMap := &corev1.ConfigMap{}
		err := v.Client.Get(ctx, types.NamespacedName{
			Name:      instance.Spec.ConfigMapName,
			Namespace: instance.Namespace,
		}, configMap)
		if err != nil {
			return fmt.Errorf("configmap %s not found: %w", instance.Spec.ConfigMapName, err)
		}
	}

	// Example: Validate parameters
	for key, value := range instance.Spec.Parameters {
		if key == "" {
			return fmt.Errorf("parameter key cannot be empty")
		}
		if value == "" {
			return fmt.Errorf("parameter value for key %s cannot be empty", key)
		}
	}

	return nil
}

// DEFAULTING WEBHOOK PATTERN
// ==========================

// MyResourceDefaulter sets default values for MyResource objects
type MyResourceDefaulter struct {
	Decoder *admission.Decoder
}

// Handle handles admission requests for defaulting
func (d *MyResourceDefaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := log.FromContext(ctx)

	// Decode the object
	instance := &MyResource{}
	err := d.Decoder.Decode(req, instance)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Set defaults
	d.setDefaults(instance)

	// Marshal the updated object
	marshaled, err := json.Marshal(instance)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	log.Info("Set defaults for MyResource", "name", instance.Name)
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

// setDefaults sets default values for the resource
func (d *MyResourceDefaulter) setDefaults(instance *MyResource) {
	// Example: Set default replicas
	if instance.Spec.Replicas == 0 {
		instance.Spec.Replicas = 1
	}

	// Example: Set default image
	if instance.Spec.Image == "" {
		instance.Spec.Image = "nginx:latest"
	}

	// Example: Set default parameters
	if instance.Spec.Parameters == nil {
		instance.Spec.Parameters = make(map[string]string)
	}

	// Example: Set default label
	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}
	if _, exists := instance.Labels["app"]; !exists {
		instance.Labels["app"] = instance.Name
	}
}

// WEBHOOK REGISTRATION
// ====================

// SetupWebhookWithManager registers the webhooks with the manager
// Add this to your main.go
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Validation webhook
	validator := &MyResourceValidator{
		Client:  mgr.GetClient(),
		Decoder: admission.NewDecoder(mgr.GetScheme()),
	}

	// Defaulting webhook
	defaulter := &MyResourceDefaulter{
		Decoder: admission.NewDecoder(mgr.GetScheme()),
	}

	// Register validation webhook
	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&MyResource{}).
		WithValidator(validator).
		Complete(); err != nil {
		return err
	}

	// Register defaulting webhook
	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&MyResource{}).
		WithDefaulter(defaulter).
		Complete(); err != nil {
		return err
	}

	return nil
}

// IMPORTANT: Add these markers to your API types to enable webhooks
// +kubebuilder:webhook:path=/validate-mygroup-my-domain-v1-myresource,mutating=false,failurePolicy=fail,sideEffects=None,groups=mygroup.my.domain,resources=myresources,verbs=create;update,versions=v1,name=vmyresource.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/mutate-mygroup-my-domain-v1-myresource,mutating=true,failurePolicy=fail,sideEffects=None,groups=mygroup.my.domain,resources=myresources,verbs=create;update,versions=v1,name=myresource.kb.io,admissionReviewVersions=v1

// HELPER FUNCTIONS
// ================

// isValidImageReference checks if a string is a valid container image reference
func isValidImageReference(image string) bool {
	// Basic validation - implement proper image reference validation
	// For production, use docker/reference package or similar
	return len(image) > 0 && len(image) < 512
}
