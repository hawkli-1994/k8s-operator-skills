package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	databasev1 "your.domain/project/api/v1"
)

func TestDatabaseReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, databasev1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name            string
		initialDatabase *databasev1.Database
		expectError     bool
		verifyStatus    func(*testing.T, *databasev1.Database)
	}{
		{
			name: "successful reconciliation",
			initialDatabase: &databasev1.Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-db",
					Namespace: "default",
				},
				Spec: databasev1.DatabaseSpec{
					Replicas:            1,
					Image:               "postgres:15",
					Storage:             1024,
					DatabaseName:        "appdb",
					UserName:            "appuser",
					PasswordSecretName:  "test-db-password",
					ServiceType:         "ClusterIP",
					StorageClass:        "standard",
				},
			},
			expectError: false,
			verifyStatus: func(t *testing.T, db *databasev1.Database) {
				assert.NotEmpty(t, db.Status.Phase)
				assert.NotEmpty(t, db.Status.DeploymentName)
				assert.NotEmpty(t, db.Status.ServiceName)
			},
		},
		{
			name: "database not found",
			initialDatabase: &databasev1.Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent",
					Namespace: "default",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.initialDatabase).
				Build()

			// Create reconciler
			reconciler := &DatabaseReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// Run reconciliation
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.initialDatabase.Name,
					Namespace: tt.initialDatabase.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, ctrl.Result{}, result, "Should return a result")
			}

			// Verify status if database exists
			if tt.name != "database not found" {
				database := &databasev1.Database{}
				err = fakeClient.Get(context.Background(), req.NamespacedName, database)
				require.NoError(t, err)
				if tt.verifyStatus != nil {
					tt.verifyStatus(t, database)
				}
			}
		})
	}
}

func TestDatabaseReconciler_ReconcileWithFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, databasev1.AddToScheme(scheme))

	now := metav1.Now()
	database := &databasev1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-db",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{databaseFinalizer},
		},
		Spec: databasev1.DatabaseSpec{
			Replicas:   1,
			Image:      "postgres:15",
			Storage:    1024,
			DatabaseName: "appdb",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(database).
		Build()

	reconciler := &DatabaseReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-db",
			Namespace: "default",
		},
	}

	// Run reconciliation
	result, err := reconciler.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify finalizer was removed
	updatedDB := &databasev1.Database{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, updatedDB)
	require.NoError(t, err)
	assert.Empty(t, updatedDB.Finalizers, "Finalizer should be removed after cleanup")
}

func TestGenerateRandomPassword(t *testing.T) {
	// Test that password generation works
	password, err := generateRandomPassword(24)
	assert.NoError(t, err)
	assert.Len(t, password, 32) // base64 encoding increases length

	// Test that generated passwords are unique
	password2, err := generateRandomPassword(24)
	assert.NoError(t, err)
	assert.NotEqual(t, password, password2, "Generated passwords should be unique")

	// Test different lengths
	shortPassword, err := generateRandomPassword(8)
	assert.NoError(t, err)
	assert.NotEqual(t, password, shortPassword)
}

func TestDatabase_SetCondition(t *testing.T) {
	db := &databasev1.Database{}

	// Set initial condition
	db.SetCondition("Ready", "True", "Ready", "Database is ready")
	assert.Len(t, db.Status.Conditions, 1)

	// Update condition
	db.SetCondition("Ready", "False", "Error", "Database failed")
	assert.Len(t, db.Status.Conditions, 1)

	// Add different condition
	db.SetCondition("Available", "True", "Available", "Database is available")
	assert.Len(t, db.Status.Conditions, 2)
}

func TestDatabase_IsReady(t *testing.T) {
	db := &databasev1.Database{}

	// Not ready initially
	assert.False(t, db.IsReady())

	// Set ready condition to true
	db.SetCondition("Ready", "True", "Ready", "Database is ready")
	assert.True(t, db.IsReady())

	// Set ready condition to false
	db.SetCondition("Ready", "False", "Error", "Database failed")
	assert.False(t, db.IsReady())
}
