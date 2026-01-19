# Kubernetes Operator Development - Quick Reference

A quick reference guide for common Kubernetes operator development tasks.

## Table of Contents
- [CRD Markers](#crd-markers)
- [RBAC Markers](#rbac-markers)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)
- [Testing](#testing)

## CRD Markers

### Basic Object Markers
```go
//+kubebuilder:object:root=true
```

### Status Subresource
```go
//+kubebuilder:subresource:status
```

### Additional Printer Columns
```go
//+kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:printcolumn:name="REPLICAS",type=integer,JSONPath=`.spec.replicas`
```

### Resource ShortName
```go
//+kubebuilder:resource:shortName=mr
```

### Validation Markers
```go
//+kubebuilder:validation:Minimum=0
//+kubebuilder:validation:Maximum=100
//+kubebuilder:validation:MinLength=1
//+kubebuilder:validation:MaxLength=255
//+kubebuilder:validation:Pattern="^[a-z0-9](-[a-z0-9])*$"
//+kubebuilder:validation:Enum=Option1;Option2;Option3
//+kubebuilder:validation:Optional
//+kubebuilder:default="default-value"
```

## RBAC Markers

### Resource Permissions
```go
//+kubebuilder:rbac:groups=myapp.my.domain,resources=myresources,verbs=get;list;watch;create;update;patch;delete
```

### Status Permissions
```go
//+kubebuilder:rbac:groups=myapp.my.domain,resources=myresources/status,verbs=get;update;patch
```

### Finalizer Permissions
```go
//+kubebuilder:rbac:groups=myapp.my.domain,resources=myresources/finalizers,verbs=update
```

### Cluster-scoped Resources
```go
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
```

## Common Patterns

### Reconciliation Loop
```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch
    obj := &MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if !obj.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, obj)
    }

    // Add finalizer
    if !controllerutil.ContainsFinalizer(obj, finalizer) {
        controllerutil.AddFinalizer(obj, finalizer)
        return ctrl.Result{}, r.Update(ctx, obj)
    }

    // Reconcile
    if err := r.reconcile(ctx, obj); err != nil {
        return ctrl.Result{}, err
    }

    // Update status
    r.Status().Update(ctx, obj)

    return ctrl.Result{}, nil
}
```

### Create or Update Pattern
```go
op, err := controllerutil.CreateOrPatch(ctx, r.Client, deployment, func() error {
    deployment.Spec = desiredSpec
    return nil
})
```

### Owner Reference
```go
controllerutil.SetControllerReference(owner, child, r.Scheme)
```

### List with Options
```go
list := &MyResourceList{}
opts := []client.ListOption{
    client.InNamespace(namespace),
    client.MatchingLabels{"app": "myapp"},
}
r.List(ctx, list, opts...)
```

### Patch for Status Updates
```go
patch := client.MergeFrom(obj.DeepCopy())
obj.Status.Phase = "Ready"
r.Status().Patch(ctx, obj, patch)
```

### Retry on Conflict
```go
err := r.Update(ctx, obj)
if errors.IsConflict(err) {
    return ctrl.Result{Requeue: true}, nil
}
```

### Conditional Requeue
```go
if !isReady {
    return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}
return ctrl.Result{}, nil
```

### Watch Related Resources
```go
Watches(
    &source.Kind{Type: &corev1.ConfigMap{}},
    handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
)
```

### Get Single Item
```go
obj := &corev1.ConfigMap{}
err := r.Get(ctx, types.NamespacedName{
    Name:      "my-config",
    Namespace: "default",
}, obj)
```

### Create Resource
```go
obj := &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}
controllerutil.SetControllerReference(owner, obj, r.Scheme)
r.Create(ctx, obj)
```

### Update Resource
```go
obj.Spec.Replicas = 5
r.Update(ctx, obj)
```

### Delete Resource
```go
r.Delete(ctx, obj)
```

### List All in Namespace
```go
list := &corev1.PodList{}
r.List(ctx, list, client.InNamespace("default"))
```

## Conditions Pattern

### Set Condition
```go
obj.Status.Conditions = []metav1.Condition{{
    Type:               "Ready",
    Status:             metav1.ConditionTrue,
    LastTransitionTime: metav1.Now(),
    Reason:             "Ready",
    Message:            "Resource is ready",
}}
```

### Get Condition
```go
func getCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
    for i := range conditions {
        if conditions[i].Type == conditionType {
            return &conditions[i]
        }
    }
    return nil
}
```

## Webhook Markers

### Validation Webhook
```go
//+kubebuilder:webhook:path=/validate-myapp-my-domain-v1-myresource,mutating=false,failurePolicy=fail,sideEffects=None,groups=myapp.my.domain,resources=myresources,verbs=create;update,versions=v1,name=vmyresource.kb.io,admissionReviewVersions=v1
```

### Defaulting Webhook
```go
//+kubebuilder:webhook:path=/mutate-myapp-my-domain-v1-myresource,mutating=true,failurePolicy=fail,sideEffects=None,groups=myapp.my.domain,resources=myresources,verbs=create;update,versions=v1,name=myresource.kb.io,admissionReviewVersions=v1
```

## Testing Patterns

### Fake Client Setup
```go
scheme := runtime.NewScheme()
mygroupv1.AddToScheme(scheme)
fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
```

### Create Test Object
```go
obj := &mygroupv1.MyResource{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "test",
        Namespace: "default",
    },
}
fakeClient.Create(ctx, obj)
```

### Assert Results
```go
Eventually(func() bool {
    err := fakeClient.Get(ctx, key, obj)
    return err == nil
}, timeout).Should(BeTrue())
```

## Logging

### Structured Logging
```go
log := log.FromContext(ctx)
log.Info("Message", "key", "value")
log.Error(err, "Error message")
```

## Metrics

### Controller Runtime Metrics
Available at `:8080/metrics` by default:
- `controller_runtime_reconcile_total`
- `controller_runtime_reconcile_errors_total`
- `workqueue_depth`

## Troubleshooting

### Reconciler Not Running
1. Check RBAC markers
2. Verify CRD is installed: `kubectl get crd`
3. Check controller logs: `kubectl logs -n <namespace> <pod>`
4. Verify resource is in watched namespace

### Status Not Updating
1. Check for `//+kubebuilder:subresource:status` marker
2. Use `r.Status().Update()` not `r.Update()`
3. Verify RBAC includes `.../status` permissions

### Finalizer Not Running
1. Ensure finalizer is added before DeletionTimestamp
2. Return error from reconcileDelete if cleanup fails
3. Check logs for errors during cleanup

### Conflict Errors
1. Always handle `errors.IsConflict(err)`
2. Return `ctrl.Result{Requeue: true}` for conflicts
3. Use `patch := client.MergeFrom()` for updates

### Child Resources Not Deleted
1. Set owner reference: `controllerutil.SetControllerReference()`
2. Verify owner reference is correct
3. Check if owner has DeletionTimestamp

### Tests Timing Out
1. Use `Eventually` with appropriate timeout
2. Ensure controller is started in tests
3. Check if fake client has correct scheme
4. Verify objects are being created correctly

## Common Commands

### Development
```bash
# Initialize project
kubebuilder init --domain my.domain --repo my.domain/myproject

# Create API
kubebuilder create api --group myapp --version v1 --kind MyResource

# Generate manifests
make manifests

# Generate code
make generate

# Run tests
make test

# Run locally
make run

# Install CRDs
make install

# Deploy to cluster
make deploy

# Uninstall
make uninstall
```

### Debugging
```bash
# Get CRD
kubectl get crd myresources.myapp.my.domain -o yaml

# Get resource
kubectl get myresource -n <namespace>

# Describe resource
kubectl describe myresource <name> -n <namespace>

# Get resource as YAML
kubectl get myresource <name> -n <namespace> -o yaml

# View controller logs
kubectl logs -n <namespace> <controller-pod>

# Watch events
kubectl get events -n <namespace> --watch
```

## Best Practices

1. **Always use context** - Pass context to all API calls
2. **Handle errors** - Return errors for permanent failures, requeue for transient
3. **Use finalizers** - For external resource cleanup
4. **Set owner refs** - For automatic garbage collection
5. **Update status** - Keep users informed
6. **Add conditions** - For complex status
7. **Write tests** - Both unit and integration
8. **Log appropriately** - Use structured logging
9. **Add metrics** - For observability
10. **Document CRDs** - Use kubebuilder markers

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Extending Kubernetes](https://kubernetes.io/docs/concepts/extend-kubernetes/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
