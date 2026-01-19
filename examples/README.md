# K8s Operator Development Examples

This directory contains example implementations demonstrating various operator patterns.

## Example: Database Operator

A complete example of an operator that manages a database (e.g., PostgreSQL, MySQL).

### Features Demonstrated
- CRD with spec and status
- Reconciliation loop for managing deployments and services
- Secret management for credentials
- Backup and restore operations
- Status conditions
- Finalizers for cleanup

## Example: Cache Operator

An operator for managing cache clusters (e.g., Redis, Memcached).

### Features Demonstrated
- Scaling operations
- ConfigMap mounting
- Service creation
- Health checks
- Metrics integration

## Example: Application Operator

An operator for deploying and managing complex applications.

### Features Demonstrated
- Multi-resource orchestration
- Owner references
- Watch configuration
- Dependency management
- Rolling updates

## Running the Examples

Each example includes:
- `api/` - Custom Resource Definition
- `controllers/` - Controller implementation
- `webhooks/` - Validation and defaulting webhooks (if applicable)
- `config/` - Kubernetes manifests
- `main.go` - Operator entry point

To run an example:

```bash
# Install CRDs
make install

# Run the operator locally
make run

# Or deploy to the cluster
make deploy
```

## Testing Examples

```bash
# Run unit tests
make test

# Run integration tests
make test-integration
```

## Creating Your Own Operator

1. Copy the relevant example as a starting point
2. Modify the CRD to match your resource
3. Update the reconciler logic
4. Add validation/defaulting as needed
5. Write tests for your operator
6. Deploy and verify

## Common Patterns

### 1. Simple Deployment Manager

Create a deployment based on spec:
```go
deployment := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
        Name:      instance.Name,
        Namespace: instance.Namespace,
    },
    Spec: appsv1.DeploymentSpec{
        Replicas: &instance.Spec.Replicas,
        // ... more spec
    },
}

controllerutil.SetControllerReference(instance, deployment, r.Scheme)
r.Create(ctx, deployment)
```

### 2. Service Creation

Create a service to expose the deployment:
```go
service := &corev1.Service{
    ObjectMeta: metav1.ObjectMeta{
        Name:      instance.Name,
        Namespace: instance.Namespace,
    },
    Spec: corev1.ServiceSpec{
        Selector: map[string]string{"app": instance.Name},
        Ports: []corev1.ServicePort{
            {
                Port:       80,
                TargetPort: intstr.FromInt(8080),
            },
        },
    },
}

controllerutil.SetControllerReference(instance, service, r.Scheme)
r.Create(ctx, service)
```

### 3. ConfigMap Reference

Mount a ConfigMap referenced in spec:
```go
if instance.Spec.ConfigMapName != "" {
    configMap := &corev1.ConfigMap{}
    err := r.Get(ctx, types.NamespacedName{
        Name:      instance.Spec.ConfigMapName,
        Namespace: instance.Namespace,
    }, configMap)

    if err != nil {
        return err
    }

    // Add volume to pod spec
    podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
        Name: "config",
        VolumeSource: corev1.VolumeSource{
            ConfigMap: &corev1.ConfigMapVolumeSource{
                LocalObjectReference: corev1.LocalObjectReference{
                    Name: instance.Spec.ConfigMapName,
                },
            },
        },
    })
}
```

### 4. Secret Management

Generate and store credentials:
```go
secret := &corev1.Secret{
    ObjectMeta: metav1.ObjectMeta{
        Name:      instance.Name + "-credentials",
        Namespace: instance.Namespace,
    },
    Data: map[string][]byte{
        "username": []byte(instance.Spec.Username),
        "password": []byte(generatePassword()),
    },
}

controllerutil.SetControllerReference(instance, secret, r.Scheme)
r.Create(ctx, secret)
```

### 5. Status Update with Conditions

```go
func (r *MyReconciler) updateStatus(ctx context.Context, instance *MyResource) {
    ready := metav1.Condition{
        Type:               "Ready",
        Status:             metav1.ConditionTrue,
        Reason:             "Ready",
        Message:            "Resource is ready",
        LastTransitionTime: metav1.Now(),
    }

    instance.Status.Conditions = []metav1.Condition{ready}
    instance.Status.ReadyReplicas = deployment.Status.ReadyReplicas

    r.Status().Update(ctx, instance)
}
```

### 6. Finalizer for Cleanup

```go
const myFinalizer = "myresource.my.domain/finalizer"

func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    instance := &MyResource{}
    r.Get(ctx, req.NamespacedName, instance)

    if !instance.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, instance)
    }

    if !controllerutil.ContainsFinalizer(instance, myFinalizer) {
        controllerutil.AddFinalizer(instance, myFinalizer)
        return ctrl.Result{}, r.Update(ctx, instance)
    }

    // Normal reconciliation
    return ctrl.Result{}, nil
}

func (r *MyReconciler) reconcileDelete(ctx context.Context, instance *MyResource) (ctrl.Result, error) {
    if controllerutil.ContainsFinalizer(instance, myFinalizer) {
        // Clean up external resources
        r.cleanupExternalResources(ctx, instance)

        controllerutil.RemoveFinalizer(instance, myFinalizer)
        return ctrl.Result{}, r.Update(ctx, instance)
    }
    return ctrl.Result{}, nil
}
```

### 7. Watch External Resources

```go
func (r *MyReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&mygroupv1.MyResource{}).
        Watches(
            &source.Kind{Type: &corev1.ConfigMap{}},
            handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
        ).
        Complete(r)
}

func (r *MyReconciler) findObjectsForConfigMap(ctx context.Context, configMap client.Object) []reconcile.Request {
    var list mygroupv1.MyResourceList
    r.List(ctx, &list, client.InNamespace(configMap.GetNamespace()))

    var requests []reconcile.Request
    for _, item := range list.Items {
        if item.Spec.ConfigMapName == configMap.GetName() {
            requests = append(requests, reconcile.Request{
                NamespacedName: types.NamespacedName{
                    Name:      item.GetName(),
                    Namespace: item.GetNamespace(),
                },
            })
        }
    }
    return requests
}
```

### 8. Polling External API

```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch instance
    instance := &MyResource{}
    r.Get(ctx, req.NamespacedName, instance)

    // Check external API
    status, err := r.checkExternalAPI(ctx, instance)
    if err != nil {
        // Requeue after delay if API is unavailable
        return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
    }

    // Update status based on external API
    instance.Status.ExternalStatus = status
    r.Status().Update(ctx, instance)

    // Requeue for next poll
    return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
}
```

### 9. Patch Instead of Update

```go
patch := client.MergeFrom(instance.DeepCopy())
instance.Status.Phase = "Running"

r.Status().Patch(ctx, instance, patch)
```

### 10. Handle Resource Conflicts

```go
err := r.Update(ctx, instance)
if err != nil {
    if errors.IsConflict(err) {
        // Resource was modified, retry
        return ctrl.Result{Requeue: true}, nil
    }
    return ctrl.Result{}, err
}
```

## Best Practices

1. **Always use context** - Pass context to all Kubernetes API calls
2. **Handle errors properly** - Distinguish between transient and permanent errors
3. **Use finalizers** - For external resources that need cleanup
4. **Set owner references** - Let Kubernetes handle garbage collection
5. **Update status** - Keep users informed about resource state
6. **Add conditions** - Use conditions for complex status information
7. **Add metrics** - Use controller-runtime metrics for observability
8. **Write tests** - Unit tests for logic, integration tests for behavior
9. **Log appropriately** - Use structured logging
10. **Document your CRD** - Add kubebuilder documentation markers

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Extending Kubernetes](https://kubernetes.io/docs/concepts/extend-kubernetes/)
