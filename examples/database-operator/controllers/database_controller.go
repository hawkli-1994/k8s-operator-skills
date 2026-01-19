package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	databasev1 "your.domain/project/api/v1"
)

const databaseFinalizer = "database.my.domain/finalizer"

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.domain,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.domain,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=databases/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Database instance
	database := &databasev1.Database{}
	err := r.Get(ctx, req.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !database.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, database)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(database, databaseFinalizer) {
		controllerutil.AddFinalizer(database, databaseFinalizer)
		if err := r.Update(ctx, database); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the database
	logger.Info("Reconciling Database", "name", database.Name, "replicas", database.Spec.Replicas)

	// Update status phase
	database.Status.Phase = "Reconciling"
	if updateErr := r.Status().Update(ctx, database); updateErr != nil {
		logger.Error(updateErr, "failed to update status")
	}

	// Reconcile child resources
	if err := r.reconcilePVC(ctx, database); err != nil {
		return r.setErrorStatus(ctx, database, "PVCCreateFailed", err)
	}

	if err := r.reconcileSecret(ctx, database); err != nil {
		return r.setErrorStatus(ctx, database, "SecretCreateFailed", err)
	}

	if err := r.reconcileConfigMap(ctx, database); err != nil {
		return r.setErrorStatus(ctx, database, "ConfigMapCreateFailed", err)
	}

	if err := r.reconcileDeployment(ctx, database); err != nil {
		return r.setErrorStatus(ctx, database, "DeploymentCreateFailed", err)
	}

	if err := r.reconcileService(ctx, database); err != nil {
		return r.setErrorStatus(ctx, database, "ServiceCreateFailed", err)
	}

	// Update status
	if err := r.updateStatus(ctx, database); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue for status check
	requeueAfter := time.Minute * 1
	if !database.IsReady() {
		requeueAfter = time.Second * 10
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDelete handles deletion of Database resources
func (r *DatabaseReconciler) reconcileDelete(ctx context.Context, database *databasev1.Database) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(database, databaseFinalizer) {
		logger.Info("Deleting Database", "name", database.Name)

		// Cleanup is handled automatically by garbage collection
		// due to owner references

		controllerutil.RemoveFinalizer(database, databaseFinalizer)
		if err := r.Update(ctx, database); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// reconcilePVC creates or updates the persistent volume claim
func (r *DatabaseReconciler) reconcilePVC(ctx context.Context, database *databasev1.Database) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, pvc, func() error {
		pvc.Spec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", database.Spec.Storage)),
				},
			},
			StorageClassName: &database.Spec.StorageClass,
		}
		return controllerutil.SetControllerReference(database, pvc, r.Scheme)
	})

	return err
}

// reconcileSecret creates or updates the database password secret
func (r *DatabaseReconciler) reconcileSecret(ctx context.Context, database *databasev1.Database) error {
	secretName := database.Spec.PasswordSecretName
	if secretName == "" {
		secretName = database.Name + "-password"
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: database.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, secret, func() error {
		if secret.Data == nil {
			// Generate secure random password
			password, err := generateRandomPassword(24)
			if err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}
			secret.Data = map[string][]byte{
				"password": []byte(password),
				"username": []byte(database.Spec.UserName),
			}
		}
		return controllerutil.SetControllerReference(database, secret, r.Scheme)
	})

	return err
}

// generateRandomPassword generates a secure random password
func generateRandomPassword(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// reconcileConfigMap creates or updates the configuration
func (r *DatabaseReconciler) reconcileConfigMap(ctx context.Context, database *databasev1.Database) error {
	if database.Spec.ConfigMapName == "" {
		return nil // ConfigMap is optional
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Spec.ConfigMapName,
			Namespace: database.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, cm, func() error {
		cm.Data = map[string]string{
			"POSTGRES_DB":       database.Spec.DatabaseName,
			"POSTGRES_USER":     database.Spec.UserName,
			"POSTGRES_PASSWORD": fmt.Sprintf("file:///etc/secrets/%s-password/password", database.Name),
		}
		return controllerutil.SetControllerReference(database, cm, r.Scheme)
	})

	return err
}

// reconcileDeployment creates or updates the deployment
func (r *DatabaseReconciler) reconcileDeployment(ctx context.Context, database *databasev1.Database) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, deployment, func() error {
		deployment.Spec.Replicas = &database.Spec.Replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": database.Name},
		}
		deployment.Spec.Template.ObjectMeta.Labels = map[string]string{"app": database.Name}

		// Set up container
		container := corev1.Container{
			Name:  "database",
			Image: database.Spec.Image,
			Env: []corev1.EnvVar{
				{
					Name: "POSTGRES_DB",
					Value: database.Spec.DatabaseName,
				},
				{
					Name: "POSTGRES_USER",
					Value: database.Spec.UserName,
				},
				{
					Name: "POSTGRES_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: database.Spec.PasswordSecretName,
							},
							Key: "password",
						},
					},
				},
			},
			Ports: []corev1.ContainerPort{
				{ContainerPort: 5432},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/var/lib/postgresql/data",
				},
			},
		}

		// Add ConfigMap volume if specified
		if database.Spec.ConfigMapName != "" {
			container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: database.Spec.ConfigMapName,
					},
				},
			})
		}

		deployment.Spec.Template.Spec.Containers = []corev1.Container{container}

		// Add PVC volume
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: database.Name,
					},
				},
			},
		}

		return controllerutil.SetControllerReference(database, deployment, r.Scheme)
	})

	return err
}

// reconcileService creates or updates the service
func (r *DatabaseReconciler) reconcileService(ctx context.Context, database *databasev1.Database) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: database.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, service, func() error {
		service.Spec.Type = database.Spec.ServiceType
		if service.Spec.Type == "" {
			service.Spec.Type = corev1.ServiceTypeClusterIP
		}

		service.Spec.Selector = map[string]string{"app": database.Name}
		service.Spec.Ports = []corev1.ServicePort{
			{
				Port:       5432,
				TargetPort: intstr.FromInt(5432),
				Protocol:   corev1.ProtocolTCP,
			},
		}

		return controllerutil.SetControllerReference(database, service, r.Scheme)
	})

	return err
}

// updateStatus updates the database status
func (r *DatabaseReconciler) updateStatus(ctx context.Context, database *databasev1.Database) error {
	// Get deployment status
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	// Update status
	database.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	database.Status.DeploymentName = deployment.Name
	database.Status.ServiceName = database.Name
	database.Status.ObservedGeneration = database.Generation

	// Update conditions
	if deployment.Status.ReadyReplicas == database.Spec.Replicas {
		database.Status.Phase = "Ready"
		database.SetCondition("Ready", metav1.ConditionTrue, "Ready", "Database is ready")
	} else {
		database.Status.Phase = "Progressing"
		database.SetCondition("Ready", metav1.ConditionFalse, "Progressing",
			fmt.Sprintf("Waiting for replicas: %d/%d", deployment.Status.ReadyReplicas, database.Spec.Replicas))
	}

	return r.Status().Update(ctx, database)
}

// setErrorStatus sets error status and returns error
func (r *DatabaseReconciler) setErrorStatus(ctx context.Context, database *databasev1.Database, reason string, err error) (ctrl.Result, error) {
	database.Status.Phase = "Failed"
	database.SetCondition("Ready", metav1.ConditionFalse, reason, err.Error())
	_ = r.Status().Update(ctx, database)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.Database{}).
		// Watch owned deployment
		Owns(&appsv1.Deployment{}).
		// Watch owned service
		Owns(&corev1.Service{}).
		// Watch owned configmap (if specified)
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findDatabasesForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		// Configure controller options
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 2,
		}).
		Complete(r)
}

// findDatabasesForConfigMap finds Databases that reference a ConfigMap
func (r *DatabaseReconciler) findDatabasesForConfigMap(ctx context.Context, o client.Object) []reconcile.Request {
	configMap := o.(*corev1.ConfigMap)
	logger := log.FromContext(ctx)

	var list databasev1.DatabaseList
	if err := r.List(ctx, &list, client.InNamespace(configMap.Namespace)); err != nil {
		logger.Error(err, "failed to list Databases")
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
