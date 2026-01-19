package patterns

// CRD (Custom Resource Definition) Pattern
//
// This file shows the standard pattern for defining Kubernetes Custom Resources.
// Replace "MyResource" with your resource name and "mygroup" with your API group.

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MyResourceSpec defines the desired state of MyResource
// Add fields here that represent the desired state
type MyResourceSpec struct {
	// IMPORTANT: Add kubebuilder validation markers as comments above fields
	// Example:
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Replicas is the desired number of replicas
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:MinLength=1
	// Image is the container image to deploy
	Image string `json:"image"`

	// +kubebuilder:validation:Optional
	// ConfigMapName is an optional ConfigMap reference
	ConfigMapName string `json:"configMapName,omitempty"`

	// +kubebuilder:validation:Optional
	// Parameters for custom configuration
	Parameters map[string]string `json:"parameters,omitempty"`
}

// MyResourceStatus defines the observed state of MyResource
// Add fields here that represent the current state
type MyResourceStatus struct {
	// IMPORTANT: Always use Conditions for complex status
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`

	// ObservedGeneration is the most recent generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Add other status fields here
	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// LastUpdated is the last time the status was updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mr
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="REPLICAS",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// MyResource is the Schema for the myresources API
type MyResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyResourceSpec   `json:"spec,omitempty"`
	Status MyResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MyResourceList contains a list of MyResource
type MyResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyResource{}, &MyResourceList{})
}

// Helper functions for working with conditions
func (r *MyResource) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Find existing condition
	found := false
	for i, condition := range r.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != newCondition.Status {
				condition.LastTransitionTime = newCondition.LastTransitionTime
			}
			r.Status.Conditions[i] = newCondition
			found = true
			break
		}
	}

	if !found {
		r.Status.Conditions = append(r.Status.Conditions, newCondition)
	}
}

func (r *MyResource) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range r.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// IsReady returns true if the Ready condition is True
func (r *MyResource) IsReady() bool {
	if condition := r.GetCondition("Ready"); condition != nil {
		return condition.Status == metav1.ConditionTrue
	}
	return false
}
