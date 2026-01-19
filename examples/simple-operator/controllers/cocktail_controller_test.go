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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	barv1 "your.domain/project/api/v1"
)

func TestCocktailReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, barv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name           string
		initialCocktail *barv1.Cocktail
		expectError    bool
		verifyStatus   func(*testing.T, *barv1.Cocktail)
	}{
		{
			name: "successful reconciliation",
			initialCocktail: &barv1.Cocktail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cocktail",
					Namespace: "default",
				},
				Spec: barv1.CocktailSpec{
					Size:     2,
					Recipe:   "Mojito",
					Garnish:  true,
				},
			},
			expectError: false,
			verifyStatus: func(t *testing.T, cocktail *barv1.Cocktail) {
				assert.Equal(t, "Ready", cocktail.Status.Phase)
				assert.Equal(t, int32(2), cocktail.Status.ServingsReady)
				assert.NotNil(t, cocktail.Status.LastPrepared)
			},
		},
		{
			name: "cocktail not found",
			initialCocktail: &barv1.Cocktail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent",
					Namespace: "default",
				},
			},
			expectError: false, // Should not error, just return
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.initialCocktail).
				Build()

			// Create reconciler
			reconciler := &CocktailReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// Run reconciliation
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.initialCocktail.Name,
					Namespace: tt.initialCocktail.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Should requeue after 5 minutes
				assert.Equal(t, ctrl.Result{RequeueAfter: time.Minute * 5}, result)
			}

			// Verify status if cocktail exists
			if tt.name != "cocktail not found" {
				cocktail := &barv1.Cocktail{}
				err = fakeClient.Get(context.Background(), req.NamespacedName, cocktail)
				require.NoError(t, err)
				if tt.verifyStatus != nil {
					tt.verifyStatus(t, cocktail)
				}
			}
		})
	}
}

func TestCocktailReconciler_ReconcileWithFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, barv1.AddToScheme(scheme))

	now := metav1.Now()
	cocktail := &barv1.Cocktail{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cocktail",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{cocktailFinalizer},
		},
		Spec: barv1.CocktailSpec{
			Size:   1,
			Recipe: "Margarita",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cocktail).
		Build()

	reconciler := &CocktailReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-cocktail",
			Namespace: "default",
		},
	}

	// Run reconciliation
	result, err := reconciler.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify finalizer was removed
	updatedCocktail := &barv1.Cocktail{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, updatedCocktail)
	require.NoError(t, err)
	assert.Empty(t, updatedCocktail.Finalizers, "Finalizer should be removed after cleanup")
}

func TestGetPreparationTime(t *testing.T) {
	reconciler := &CocktailReconciler{}

	tests := []struct {
		recipe           string
		expectedDuration time.Duration
	}{
		{"Mojito", 2 * time.Minute},
		{"Margarita", 3 * time.Minute},
		{"OldFashioned", 4 * time.Minute},
		{"Cosmopolitan", 2 * time.Minute},
		{"Unknown", 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.recipe, func(t *testing.T) {
			result := reconciler.getPreparationTime(tt.recipe)
			assert.Equal(t, tt.expectedDuration, result)
		})
	}
}
