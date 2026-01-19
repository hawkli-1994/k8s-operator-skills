# K8s Operator Development Skill

This directory contains a comprehensive skill for helping users build Kubernetes operators using Go, controller-runtime, and Kubebuilder.

## Structure

```
k8s-operator-skills/
├── skill.md              # Main skill documentation
├── patterns/             # Code patterns and templates
│   ├── crd.go           # CRD definition patterns
│   ├── reconciler.go    # Reconciler implementation patterns
│   ├── webhook.go       # Webhook patterns
│   └── test.go          # Testing patterns
├── examples/             # Example implementations
│   ├── README.md        # Example documentation
│   └── simple-operator/ # Simple example operator
└── docs/                 # Extracted book content (if available)
```

## Quick Start

When a user asks for help building a Kubernetes operator:

1. **Understand Requirements**
   - What resource do they want to manage?
   - What operations should it support?
   - What external systems does it need to interact with?

2. **Scaffold the Project**
   ```bash
   kubebuilder init --domain my.domain --repo my.domain/myproject
   kubebuilder create api --group myapp --version v1 --kind MyResource
   ```

3. **Define the CRD**
   - Use `patterns/crd.go` as a reference
   - Add proper validation markers
   - Include status subresource
   - Add printcolumn markers for kubectl output

4. **Implement the Reconciler**
   - Use `patterns/reconciler.go` as a reference
   - Follow the reconciliation loop pattern
   - Handle finalizers properly
   - Update status correctly

5. **Add Webhooks (if needed)**
   - Use `patterns/webhook.go` as a reference
   - Implement validation and/or defaulting
   - Add webhook markers

6. **Write Tests**
   - Use `patterns/test.go` as a reference
   - Write unit tests with fake client
   - Write integration tests with envtest

## Common Workflows

### Creating a New Operator

1. Ask clarifying questions about the resource
2. Use kubebuilder to scaffold the project
3. Define the CRD structure
4. Implement the reconciler
5. Add tests
6. Deploy and verify

### Adding Features to Existing Operator

1. Read existing code to understand structure
2. Follow established patterns
3. Add new fields to CRD
4. Update reconciler logic
5. Add/update tests
6. Run `make manifests` and `make generate`

### Debugging Operator Issues

1. Check RBAC markers
2. Verify CRD is installed
3. Check controller logs
4. Verify resource is in watched namespace
5. Check for conflicts

## Key Patterns

### CRD Definition

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.phase`

type MyResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MyResourceSpec   `json:"spec,omitempty"`
    Status MyResourceStatus `json:"status,omitempty"`
}
```

### Reconciliation Loop

```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the resource
    // 2. Check deletion timestamp
    // 3. Add finalizer if needed
    // 4. Reconcile the resource
    // 5. Update status
    // 6. Return result
}
```

### Status Updates

```go
func (r *MyReconciler) updateStatus(ctx context.Context, obj *MyResource) {
    obj.SetCondition("Ready", metav1.ConditionTrue, "Ready", "Resource is ready")
    r.Status().Update(ctx, obj)
}
```

## Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run controller locally
make run

# Install CRDs
make install

# Deploy to cluster
make deploy
```

## Best Practices

1. Always use `context.Context` for all API calls
2. Handle `errors.IsConflict()` for conflicts
3. Use `r.Status().Update()` for status, not `r.Update()`
4. Add finalizers before deletion timestamp appears
5. Set owner references for child resources
6. Use structured logging
7. Add conditions for complex status
8. Write comprehensive tests
9. Use appropriate RBAC markers
10. Follow Kubernetes API conventions

## Commands Reference

| Command | Purpose |
|---------|---------|
| `kubebuilder init` | Initialize project |
| `kubebuilder create api` | Create new API |
| `make manifests` | Generate CRD manifests |
| `make generate` | Generate code |
| `make install` | Install CRDs |
| `make run` | Run locally |
| `make deploy` | Deploy to cluster |
| `make test` | Run tests |

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)

## Contributing

When adding new patterns or examples:
1. Follow existing code style
2. Add comments explaining the pattern
3. Include usage examples
4. Update this README
