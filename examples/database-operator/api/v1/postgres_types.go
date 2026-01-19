package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// Replicas is the number of database instances
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:MinLength=1
	// Image is the database container image
	Image string `json:"image"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100000
	// Storage is the size of the persistent volume claim (in MiB)
	Storage int32 `json:"storage"`

	// +kubebuilder:validation:Optional
	// DatabaseName is the name of the database to create
	DatabaseName string `json:"databaseName,omitempty"`

	// +kubebuilder:validation:Optional
	// UserName is the database user name
	UserName string `json:"userName,omitempty"`

	// +kubebuilder:validation:Optional
	// PasswordSecretName is the name of the secret containing the database password
	PasswordSecretName string `json:"passwordSecretName,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigMapName is the name of the configmap with additional settings
	ConfigMapName string `json:"configMapName,omitempty"`

	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +kubebuilder:validation:Optional
	// ServiceType is the Kubernetes service type
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// +kubebuilder:validation:Optional
	// StorageClass is the storage class to use
	StorageClass string `json:"storageClass,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// +kubebuilder:validation:Optional
	// Phase is the current phase of the database
	Phase string `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	// ObservedGeneration is the generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Optional
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	// ServiceName is the name of the created service
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	// DeploymentName is the name of the created deployment
	DeploymentName string `json:"deploymentName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=db
//+kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.readyReplicas`
//+kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}

// SetCondition sets a condition on the Database status
func (d *Database) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	found := false
	for i, condition := range d.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != newCondition.Status {
				condition.LastTransitionTime = newCondition.LastTransitionTime
			}
			d.Status.Conditions[i] = newCondition
			found = true
			break
		}
	}

	if !found {
		d.Status.Conditions = append(d.Status.Conditions, newCondition)
	}
}

// GetCondition gets a condition from the Database status
func (d *Database) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range d.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// IsReady returns true if the Database is ready
func (d *Database) IsReady() bool {
	if condition := d.GetCondition("Ready"); condition != nil {
		return condition.Status == metav1.ConditionTrue
	}
	return false
}
