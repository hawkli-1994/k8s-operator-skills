# K8s Operator Development Skill

A comprehensive Claude skill for building Kubernetes operators using Go, controller-runtime, and Kubebuilder.

## Overview

This skill provides everything needed to help developers build production-ready Kubernetes operators. It includes detailed patterns, examples, and best practices for operator development.

## What's Included

### Core Documentation
- **skill.md** - Complete skill guide with workflows, patterns, and best practices
- **CLAUDE.md** - Repository overview for Claude Code
- **QUICKREF.md** - Quick reference for common tasks and markers

### Code Patterns (patterns/)
- **crd.go** - Custom Resource Definition patterns with validation
- **reconciler.go** - Complete reconciler implementation with finalizers, status updates
- **webhook.go** - Validation and defaulting webhook patterns
- **test.go** - Unit and integration test patterns with fake client and envtest

### Examples (examples/)
- **README.md** - Comprehensive example documentation
- **simple-operator/** - Complete working example of a simple operator

### Documentation
- **docs/** - Extracted book content (when available)

## Key Features

### Vibe Coding Support
- Iterative, conversational development approach
- Show progress frequently
- Ask clarifying questions when needed
- Adapt based on feedback

### Comprehensive Coverage
- CRD design and validation
- Reconciliation loop patterns
- Status and condition management
- Finalizers and cleanup
- Webhooks (validation & defaulting)
- Testing strategies
- RBAC configuration
- Owner references
- Watch patterns

### Production-Ready Patterns
- Error handling and retries
- Conflict resolution
- Resource cleanup
- External API integration
- Metrics and logging
- Multi-version API support

## How to Use

### For Users

1. Load this skill in Claude Code
2. Ask questions about building operators
3. Request help with specific patterns
4. Get explanations and examples

Example prompts:
- "Help me create an operator for managing databases"
- "Show me how to add a webhook for validation"
- "How do I implement finalizers for cleanup?"
- "Write tests for my reconciler"

### For Claude

When working on operator development:

1. **Understand Requirements**
   - What resource to manage?
   - What operations needed?
   - External systems involved?

2. **Follow the Workflow**
   - Scaffold with kubebuilder
   - Define CRD
   - Implement reconciler
   - Add webhooks (if needed)
   - Write tests
   - Deploy and verify

3. **Use the Patterns**
   - Reference code patterns for common tasks
   - Follow established conventions
   - Include proper markers and validation

4. **Communicate**
   - Show your work
   - Explain your approach
   - Ask for clarification
   - Iterate based on feedback

## Project Structure

```
k8s-operator-skills/
├── skill.md              # Main skill documentation
├── CLAUDE.md             # Repository overview
├── QUICKREF.md           # Quick reference guide
├── patterns/             # Code patterns
│   ├── crd.go           # CRD patterns
│   ├── reconciler.go    # Reconciler patterns
│   ├── webhook.go       # Webhook patterns
│   └── test.go          # Testing patterns
├── examples/             # Example implementations
│   ├── README.md        # Example docs
│   └── simple-operator/ # Simple example
└── docs/                 # Book content (if available)
```

## Common Workflows

### Creating a New Operator
```bash
# 1. Initialize project
kubebuilder init --domain my.domain --repo my.domain/myproject

# 2. Create API
kubebuilder create api --group myapp --version v1 --kind MyResource

# 3. Define CRD (edit api/v1/myresource_types.go)

# 4. Implement reconciler (edit controllers/myresource_controller.go)

# 5. Generate manifests
make manifests

# 6. Run tests
make test

# 7. Install CRDs
make install

# 8. Run locally
make run
```

### Adding Webhooks
```bash
# 1. Create webhook
kubebuilder create webhook --group myapp --version v1 --kind MyResource --defaulting --programmatic-validation

# 2. Implement webhook handlers

# 3. Generate manifests
make manifests

# 4. Run tests
make test
```

### Testing
```bash
# Unit tests
make test

# Integration tests
make test-integration

# End-to-end tests
kubectl apply -f config/samples/
kubectl get myresources
```

## Key Patterns Reference

### CRD with Status
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
    // Fetch
    obj := &MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if !obj.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, obj)
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

## Best Practices

1. **Always use context** - Pass `context.Context` to all API calls
2. **Handle errors properly** - Distinguish transient vs permanent errors
3. **Use finalizers** - For external resource cleanup
4. **Set owner references** - For automatic garbage collection
5. **Update status** - Keep users informed with conditions
6. **Write tests** - Unit tests with fake client, integration with envtest
7. **Add metrics** - Use controller-runtime metrics
8. **Log appropriately** - Use structured logging
9. **Follow conventions** - Kubernetes API conventions
10. **Document CRDs** - Use kubebuilder markers

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Extending Kubernetes](https://kubernetes.io/docs/concepts/extend-kubernetes/)

## Contributing

When extending this skill:
1. Follow existing code style and patterns
2. Add comprehensive comments
3. Include usage examples
4. Update documentation
5. Test changes thoroughly

## License

This skill is provided as-is for educational and development purposes.

## Summary

This K8s Operator Development Skill provides:

✅ Complete skill documentation (skill.md)
✅ Repository overview (CLAUDE.md)
✅ Quick reference guide (QUICKREF.md)
✅ CRD definition patterns (patterns/crd.go)
✅ Reconciler implementation patterns (patterns/reconciler.go)
✅ Webhook patterns (patterns/webhook.go)
✅ Testing patterns (patterns/test.go)
✅ Example documentation (examples/README.md)
✅ Simple working example (examples/simple-operator/)
✅ Comprehensive best practices and workflows

The skill is ready to use for building production-ready Kubernetes operators with Go and controller-runtime!
