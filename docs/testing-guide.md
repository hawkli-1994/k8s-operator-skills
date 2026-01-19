# Testing Guide for Kubernetes Operators

This guide covers testing strategies and best practices for Kubernetes operators.

## Table of Contents

1. [Testing Overview](#testing-overview)
2. [Unit Testing](#unit-testing)
3. [Integration Testing](#integration-testing)
4. [End-to-End Testing](#end-to-end-testing)
5. [Testing Scenarios](#testing-scenarios)
6. [Test Utilities](#test-utilities)
7. [Best Practices](#best-practices)

---

## Testing Overview

### Testing Pyramid

```
           /\
          /  \
         / E2E \        - Few, slow, expensive
        /------\
       /Integration\    - Some, medium speed
      /------------\
     /   Unit Tests  \  - Many, fast, cheap
    /----------------\
```

### Tooling

| Tool | Purpose |
|------|---------|
| **Go testing** | Unit tests |
| **envtest** | Integration tests with real K8s API |
| **Ginkgo** | BDD-style testing |
| **gomega** | Assertions |
| **Kind/k3d** | E2E testing in real clusters |
| **envtest** | Lightweight K8s API for testing |

---

## 1. Unit Testing

Unit tests verify individual functions and methods in isolation using a fake client.

### Setting Up the Fake Client

```go
package controllers_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	myappv1 "your.domain/project/api/v1"
	"your.domain/project/controllers"
)

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, myappv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	// Create fake client with initial objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&myappv1.MyResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
				Namespace: "default",
			},
			Spec: myappv1.MyResourceSpec{
				Replicas: 3,
			},
		}).
		Build()

	// Create reconciler
	reconciler := &controllers.MyResourceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Run reconciliation
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)

	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Minute * 5}, result)

	// Verify object was updated
	resource := &myappv1.MyResource{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, resource)
	require.NoError(t, err)
	assert.Equal(t, "Ready", resource.Status.Phase)
}
```

### Testing Reconcile Logic

```go
func TestReconcileLogic(t *testing.T) {
	tests := []struct {
		name           string
		initialObject  *myappv1.MyResource
		expectedStatus myappv1.MyResourceStatus
		expectError    bool
	}{
		{
			name: "successful reconciliation",
			initialObject: &myappv1.MyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: myappv1.MyResourceSpec{
					Replicas: 2,
				},
			},
			expectedStatus: myappv1.MyResourceStatus{
				Phase:           "Ready",
				ReadyReplicas:   2,
				ObservedGeneration: 1,
			},
			expectError: false,
		},
		{
			name: "invalid spec",
			initialObject: &myappv1.MyResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: myappv1.MyResourceSpec{
					Replicas: 0, // Invalid
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, myappv1.AddToScheme(scheme))

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.initialObject).
				Build()

			reconciler := &controllers.MyResourceReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.initialObject.Name,
					Namespace: tt.initialObject.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{RequeueAfter: time.Minute * 5}, result)

				// Verify status
				resource := &myappv1.MyResource{}
				err = fakeClient.Get(context.Background(), req.NamespacedName, resource)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus.Phase, resource.Status.Phase)
			}
		})
	}
}
```

### Testing Finalizers

```go
func TestFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, myappv1.AddToScheme(scheme))

	now := metav1.Now()
	resource := &myappv1.MyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{"myresource.my.domain/finalizer"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(resource).
		Build()

	reconciler := &controllers.MyResourceReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// Verify finalizer was removed
	updated := &myappv1.MyResource{}
	err = fakeClient.Get(context.Background(), req.NamespacedName, updated)
	require.NoError(t, err)
	assert.Empty(t, updated.Finalizers)
}
```

---

## 2. Integration Testing

Integration tests use envtest to run against a real Kubernetes API server.

### Setting Up envtest

```go
package controllers_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	myappv1 "your.domain/project/api/v1"
	"your.domain/project/controllers"
)

// These tests use Ginkgo (BDD-style testing framework)
func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	cancelMgr context.CancelFunc
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = myappv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&controllers.MyResourceReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	ctx, cancelMgr = context.WithCancel(context.Background())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelMgr()
	Expect(testEnv.Stop()).To(Succeed())
})
```

### Integration Test Example

```go
var _ = Describe("MyResource Controller", func() {
	ctx := context.Background()

	It("should create a deployment", func() {
		By("creating a MyResource")
		resource := &myappv1.MyResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "my.domain/v1",
				Kind:       "MyResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
				Namespace: "default",
			},
			Spec: myappv1.MyResourceSpec{
				Replicas: 2,
			},
		}
		Expect(k8sClient.Create(ctx, resource)).To(Succeed())

		By("checking if deployment is created")
		Eventually(func() bool {
			deployment := &appsv1.Deployment{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			}, deployment)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("checking if resource status is updated")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			}, resource)
			if err != nil {
				return false
			}
			return resource.Status.Phase == "Ready"
		}, timeout, interval).Should(BeTrue())
	})
})
```

### Webhook Testing

```go
func TestValidatingWebhook(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, myappv1.AddToScheme(scheme))

	webhook := &&myappv1.MyResourceWebhook{}

	tests := []struct {
		name      string
		resource  *myappv1.MyResource
		expectErr bool
	}{
		{
			name: "valid resource",
			resource: &myappv1.MyResource{
				Spec: myappv1.MyResourceSpec{
					Replicas: 3,
				},
			},
			expectErr: false,
		},
		{
			name: "invalid replicas",
			resource: &myappv1.MyResource{
				Spec: myappv1.MyResourceSpec{
					Replicas: 0,
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := webhook.ValidateCreate(context.Background(), tt.resource)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

---

## 3. End-to-End Testing

E2E tests run against a real Kubernetes cluster using Kind or k3d.

### Setting Up Kind for E2E Tests

```bash
# Install Kind
go install sigs.k8s.io/kind@latest

# Create cluster
kind create cluster --name test

# Load operator image
kind load docker-image --name test my-operator:latest

# Install operator
kubectl apply -f config/crd/bases/
kubectl apply -f config/rbac/
kubectl apply -f config/manager/
```

### E2E Test Example

```go
// +build e2e

package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	myappv1 "your.domain/project/api/v1"
)

func TestE2E(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	k8sClient, err := client.New(cfg, client.Options{})
	require.NoError(t, err)

	// Create resource
	resource := &myappv1.MyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-test",
			Namespace: "default",
		},
		Spec: myappv1.MyResourceSpec{
			Replicas: 3,
		},
	}
	err = k8sClient.Create(ctx, resource)
	require.NoError(t, err)

	// Wait for readiness
	assert.Eventually(t, func() bool {
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      "e2e-test",
			Namespace: "default",
		}, resource)
		if err != nil {
			return false
		}
		return resource.Status.Phase == "Ready"
	}, 5*time.Minute, 10*time.Second)

	// Verify deployment exists
	deployment := &appsv1.Deployment{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      "e2e-test",
		Namespace: "default",
	}, deployment)
	require.NoError(t, err)
	assert.Equal(t, int32(3), *deployment.Spec.Replicas)

	// Cleanup
	k8sClient.Delete(ctx, resource)
}
```

---

## 4. Testing Scenarios

### Creation Flow

```go
func TestCreationFlow(t *testing.T) {
	// 1. Create resource
	// 2. Verify finalizer added
	// 3. Verify child resources created
	// 4. Verify status updated
}
```

### Update Flow

```go
func TestUpdateFlow(t *testing.T) {
	// 1. Create resource
	// 2. Update spec
	// 3. Verify child resources updated
	// 4. Verify status updated
}
```

### Deletion Flow

```go
func TestDeletionFlow(t *testing.T) {
	// 1. Create resource
	// 2. Delete resource
	// 3. Verify finalizer runs
	// 4. Verify child resources deleted
	// 5. Verify finalizer removed
}
```

### Error Scenarios

```go
func TestErrorRecovery(t *testing.T) {
	// 1. Test invalid spec
	// 2. Test external service unavailable
	// 3. Test insufficient permissions
}
```

---

## 5. Test Utilities

### Custom Assertions

```go
// HaveCondition checks if a resource has a specific condition
func HaveCondition(conditionType, status string) types.GomegaMatcher {
	return WithTransform(func(resource *MyResource) (metav1.Condition, error) {
		for _, c := range resource.Status.Conditions {
			if c.Type == conditionType {
				return c, nil
			}
		}
		return metav1.Condition{}, fmt.Errorf("condition not found")
	}, MatchFields(IgnoreExtras, Fields{
		"Type":   Equal(conditionType),
		"Status": Equal(metav1.ConditionStatus(status)),
	}))
}
```

### Test Resource Builders

```go
func NewTestResource(name, namespace string, replicas int32) *myappv1.MyResource {
	return &myappv1.MyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: myappv1.MyResourceSpec{
			Replicas: replicas,
		},
	}
}

func WithFinalizer(resource *myappv1.MyResource) *myappv1.MyResource {
	resource.Finalizers = append(resource.Finalizers, "myresource.my.domain/finalizer")
	return resource
}

func MarkForDeletion(resource *myappv1.MyResource) *myappv1.MyResource {
	now := metav1.Now()
	resource.DeletionTimestamp = &now
	return resource
}
```

---

## 6. Best Practices

### DO's

1. **Test both success and failure paths**
2. **Use table-driven tests for multiple scenarios**
3. **Clean up test resources**
4. **Use Eventually for async operations**
5. **Mock external dependencies**
6. **Test error handling**
7. **Verify RBAC markers match code**
8. **Test finalizers thoroughly**

### DON'Ts

1. **Don't skip error handling in tests**
2. **Don't use real clusters for unit tests**
3. **Don't leak resources in tests**
4. **Don't ignore test timeouts**
5. **Don't test library code (Kubernetes itself)**
6. **Don't use sleep() - use Eventually**
7. **Don't create tests that are too specific**

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
go test ./... -short

# Run integration tests
go test ./... -integration

# Run with coverage
go test ./... -coverprofile=coverage.out

# Run with race detection
go test ./... -race

# Run specific test
go test -v ./controllers -run TestReconcile

# Run with verbose output
go test -v ./...
```

---

## Resources

- [Kubebuilder Testing](https://book.kubebuilder.io/reference/testing.html)
- [Controller Runtime Testing](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/testing)
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
