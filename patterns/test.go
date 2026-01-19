package patterns

// Testing Pattern
//
// This file shows patterns for testing Kubernetes operators
// Includes both unit tests and integration tests

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// UNIT TEST WITH FAKE CLIENT
// ===========================

func TestMyResourceReconciler(t *testing.T) {
	g := NewWithT(t)

	// Create a scheme with all the types we need
	scheme := runtime.NewScheme()
	err := MyGroupV1AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())

	err = appsv1.AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())

	// Create a fake client with initial objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&MyResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
				Namespace: "default",
			},
			Spec: MyResourceSpec{
				Replicas: 3,
				Image:    "nginx:latest",
			},
		}).
		Build()

	// Create the reconciler
	reconciler := &MyResourceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Create a reconcile request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Call Reconcile
	result, err := reconciler.Reconcile(context.Background(), req)

	// Assert results
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Minute * 5))

	// Verify the resource was updated
	instance := &MyResource{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, instance)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(instance.Status.ReadyReplicas).To(Equal(int32(0))) // Assuming no pods ready yet
}

func TestMyResourceReconciler_Deletion(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	err := MyGroupV1AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())

	now := metav1.Now()
	instance := &MyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-resource",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{"myresource.my.domain/finalizer"},
		},
		Spec: MyResourceSpec{
			Replicas: 3,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(instance).
		Build()

	reconciler := &MyResourceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())

	// Verify finalizer was removed
	err = fakeClient.Get(context.Background(), req.NamespacedName, instance)
	if !errors.IsNotFound(err) {
		g.Expect(instance.Finalizers).NotTo(ContainElement("myresource.my.domain/finalizer"))
	}
}

func TestMyResourceReconciler_Conflict(t *testing.T) {
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	err := MyGroupV1AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())

	instance := &MyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-resource",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: MyResourceSpec{
			Replicas: 3,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(instance).
		Build()

	reconciler := &MyResourceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Simulate conflict by modifying the resource in another goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		conflictedInstance := &MyResource{}
		fakeClient.Get(context.Background(), req.NamespacedName, conflictedInstance)
		conflictedInstance.Spec.Replicas = 5
		fakeClient.Update(context.Background(), conflictedInstance)
	}()

	// This should handle the conflict and retry
	result, err := reconciler.Reconcile(context.Background(), req)

	// Your reconciler should handle conflicts gracefully
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeTrue()) // Should requeue on conflict
}

// INTEGRATION TEST WITH ENVTEST
// ==============================

var (
	testEnv   *envtest.Environment
	testCtx   context.Context
	cancel    context.CancelFunc
	testClient client.Client
)

var _ = BeforeSuite(func() {
	testCtx, cancel = context.WithCancel(context.Background())

	// Create the test environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "config", "webhook")},
		},
	}

	// Start the test environment
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Create the manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Metrics:            metricsserver.Options{BindAddress: "0"},
		LeaderElection:     false,
		LeaderElectionID:   "test-operator",
	})
	Expect(err).NotTo(HaveOccurred())

	// Register the reconciler
	err = (&MyResourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// Start the manager
	go func() {
		err = mgr.Start(testCtx)
		Expect(err).NotTo(HaveOccurred())
	}()

	testClient = mgr.GetClient()
})

var _ = AfterSuite(func() {
	cancel()
	testEnv.Stop()
})

var _ = Describe("MyResource Integration Tests", func() {
	Context("When creating a MyResource", func() {
		It("Should reconcile and create deployments", func() {
			By("Creating a new MyResource")
			instance := &MyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test",
					Namespace: "default",
				},
				Spec: MyResourceSpec{
					Replicas: 2,
					Image:    "nginx:latest",
				},
			}

			Expect(testClient.Create(testCtx, instance)).To(Succeed())

			// Wait for reconciliation
			Eventually(func() bool {
				err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "integration-test",
					Namespace: "default",
				}, instance)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Check status was updated
			Eventually(func() *metav1.Condition {
				if err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "integration-test",
					Namespace: "default",
				}, instance); err != nil {
					return nil
				}
				return instance.GetCondition("Ready")
			}, timeout, interval).ShouldNot(BeNil())

			// Verify deployment was created
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "integration-test",
					Namespace: "default",
				}, deployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(deployment.Spec.Replicas).To(Equal(int32(2)))
		})
	})

	Context("When deleting a MyResource", func() {
		It("Should clean up external resources", func() {
			By("Creating a MyResource")
			instance := &MyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-test",
					Namespace: "default",
				},
				Spec: MyResourceSpec{
					Replicas: 1,
					Image:    "nginx:latest",
				},
			}

			Expect(testClient.Create(testCtx, instance)).To(Succeed())

			// Wait for deployment to be created
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "cleanup-test",
					Namespace: "default",
				}, deployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Deleting the MyResource")
			Expect(testClient.Delete(testCtx, instance)).To(Succeed())

			// Verify finalizer was cleaned up
			Eventually(func() bool {
				err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "cleanup-test",
					Namespace: "default",
				}, instance)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			// Verify deployment was deleted (by garbage collector)
			Eventually(func() bool {
				err := testClient.Get(testCtx, types.NamespacedName{
					Name:      "cleanup-test",
					Namespace: "default",
				}, deployment)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})
})

// WEBHOOK TESTS
// =============

func TestMyResourceValidator(t *testing.T) {
	tests := []struct {
		name      string
		instance  *MyResource
		wantErr   bool
		errString string
	}{
		{
			name: "valid resource",
			instance: &MyResource{
				Spec: MyResourceSpec{
					Replicas: 3,
					Image:    "nginx:latest",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid replicas - negative",
			instance: &MyResource{
				Spec: MyResourceSpec{
					Replicas: -1,
					Image:    "nginx:latest",
				},
			},
			wantErr:   true,
			errString: "replicas must be between 0 and 100",
		},
		{
			name: "missing image",
			instance: &MyResource{
				Spec: MyResourceSpec{
					Replicas: 3,
					Image:    "",
				},
			},
			wantErr:   true,
			errString: "image must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			scheme := runtime.NewScheme()
			err := MyGroupV1AddToScheme(scheme)
			g.Expect(err).NotTo(HaveOccurred())

			decoder := admission.NewDecoder(scheme)
			validator := &MyResourceValidator{
				Decoder: decoder,
			}

			// Create admission request
			raw, err := json.Marshal(tt.instance)
			g.Expect(err).NotTo(HaveOccurred())

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{Raw: raw},
				},
			}

			// Handle the request
			response := validator.Handle(context.Background(), req)

			if tt.wantErr {
				g.Expect(response.Allowed).To(BeFalse())
				g.Expect(response.Result.Message).To(ContainSubstring(tt.errString))
			} else {
				g.Expect(response.Allowed).To(BeTrue())
			}
		})
	}
}
