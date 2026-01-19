package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CocktailSpec defines the desired state of Cocktail
type CocktailSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// Size is the number of cocktail servings to prepare
	Size int32 `json:"size"`

	// +kubebuilder:validation:Enum=Mojito;Margarita;OldFashioned;Cosmopolitan
	// Recipe is the type of cocktail to prepare
	Recipe string `json:"recipe"`

	// +kubebuilder:validation:Optional
	// Garnish indicates whether to add garnish
	Garnish bool `json:"garnish,omitempty"`

	// +kubebuilder:validation:Optional
	// Instructions are custom preparation instructions
	Instructions string `json:"instructions,omitempty"`
}

// CocktailStatus defines the observed state of Cocktail
type CocktailStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// Phase indicates the current state of cocktail preparation
	Phase string `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	// ServingsReady is the number of servings currently ready
	ServingsReady int32 `json:"servingsReady,omitempty"`

	// +kubebuilder:validation:Optional
	// LastPrepared is the timestamp when the cocktail was last prepared
	LastPrepared *metav1.Time `json:"lastPrepared,omitempty"`

	// +kubebuilder:validation:Optional
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=cocktail
//+kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.servingsReady`
//+kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Cocktail is the Schema for the cocktails API
type Cocktail struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CocktailSpec   `json:"spec,omitempty"`
	Status CocktailStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CocktailList contains a list of Cocktail
type CocktailList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cocktail `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cocktail{}, &CocktailList{})
}

// SetCondition sets a condition on the Cocktail status
func (c *Cocktail) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Find existing condition
	found := false
	for i, condition := range c.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != newCondition.Status {
				condition.LastTransitionTime = newCondition.LastTransitionTime
			}
			c.Status.Conditions[i] = newCondition
			found = true
			break
		}
	}

	if !found {
		c.Status.Conditions = append(c.Status.Conditions, newCondition)
	}
}

// GetCondition gets a condition from the Cocktail status
func (c *Cocktail) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range c.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// IsReady returns true if the Cocktail is ready
func (c *Cocktail) IsReady() bool {
	if condition := c.GetCondition("Ready"); condition != nil {
		return condition.Status == metav1.ConditionTrue
	}
	return false
}
