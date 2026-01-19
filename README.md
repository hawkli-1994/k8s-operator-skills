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
- **advanced-reconciler.go** - Production patterns: leader election, watches, retries, conflict resolution
- **webhook.go** - Validation and defaulting webhook patterns
- **test.go** - Unit and integration test patterns with fake client and envtest

### Examples (examples/)
- **simple-operator/** - Complete runnable kubebuilder project
  - Full project structure (go.mod, Makefile, main.go)
  - Config directory with CRD, RBAC, samples
  - Dockerfile and setup documentation
- **database-operator/** - Real-world example managing Deployments, Services, ConfigMaps, Secrets, PVCs
  - Multi-resource orchestration
  - Owner references and status aggregation
  - ConfigMap watching

### Templates (templates/)
- **.github/workflows/** - CI/CD workflows (lint, test, build, release)
- **Dockerfile.multiarch** - Multi-architecture container build
- **.golangci.yml** - Golangci-lint configuration

### Documentation (docs/)
- **ci-cd.md** - Complete CI/CD setup guide
- **packaging.md** - Helm, OLM, Kustomize distribution guide
- **testing-guide.md** - Unit, integration, E2E testing strategies

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

## Installation

### Option 1: One-Click Install (Recommended)

**Quick install with curl:**
```bash
curl -fsSL https://raw.githubusercontent.com/hawkli-1994/k8s-operator-skills/main/install.sh | bash
```

**Or manually clone to Claude skills directory:**
```bash
git clone https://github.com/hawkli-1994/k8s-operator-skills.git ~/.claude/skills/k8s-operator
```

The skill will be automatically available when you start working on Kubernetes operator projects!

### Option 2: Browse Repository (For Learning)

If you just want to browse the code and documentation without installing as a skill:

```bash
# Clone to any location
git clone https://github.com/hawkli-1994/k8s-operator-skills.git
cd k8s-operator-skills
```

**Note:** This will NOT install it as a Claude skill. The skill will only be available in the cloned directory, not globally across all conversations.

### Quick Start

Once installed, the skill is automatically available. Try these prompts:

```
# Get started
"Help me create a Kubernetes operator for managing databases"

# Learn advanced patterns
"Show me the advanced reconciler patterns with leader election"

# Use examples
"Explain the simple-operator example"
"How do I deploy the database operator?"

# Get help with specific tasks
"How do I set up CI/CD for my operator?"
"Write tests for my reconciler"
"Add a webhook for validation"
```

## How to Use

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
├── README.md             # This file
├── patterns/             # Code patterns
│   ├── crd.go                    # CRD patterns
│   ├── reconciler.go             # Reconciler patterns
│   ├── advanced-reconciler.go    # Advanced production patterns
│   ├── webhook.go                # Webhook patterns
│   └── test.go                   # Testing patterns
├── examples/             # Example implementations
│   ├── README.md                  # Example docs
│   ├── simple-operator/           # Complete runnable example
│   │   ├── go.mod
│   │   ├── Makefile
│   │   ├── main.go
│   │   ├── Dockerfile
│   │   ├── api/v1/
│   │   ├── controllers/
│   │   ├── config/
│   │   └── project-setup.md
│   └── database-operator/         # Real-world multi-resource example
│       ├── api/v1/
│       ├── controllers/
│       └── config/
├── templates/            # Reusable templates
│   ├── .github/workflows/
│   │   ├── ci.yml
│   │   ├── release.yml
│   │   └── kind-config.yaml
│   ├── Dockerfile.multiarch
│   └── .golangci.yml
└── docs/                 # Detailed guides
    ├── ci-cd.md
    ├── packaging.md
    └── testing-guide.md
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

### Core Content
✅ Complete skill documentation (skill.md)
✅ Repository overview (CLAUDE.md)
✅ Quick reference guide (QUICKREF.md)
✅ Comprehensive README with installation guide

### Code Patterns
✅ CRD definition patterns (patterns/crd.go)
✅ Reconciler implementation patterns (patterns/reconciler.go)
✅ **Advanced production patterns** (patterns/advanced-reconciler.go)
  - Leader election, complex watches, selective reconciliation
  - Retry with backoff, patch strategies, conflict resolution
✅ Webhook patterns (patterns/webhook.go)
✅ Testing patterns (patterns/test.go)

### Working Examples
✅ **Simple operator** - Complete runnable kubebuilder project
  - Full scaffolding (go.mod, Makefile, main.go)
  - Config structure (CRD, RBAC, manager, samples)
  - Dockerfile and setup guide
✅ **Database operator** - Real-world multi-resource orchestration
  - Manages Deployments, Services, ConfigMaps, Secrets, PVCs
  - Owner references and status aggregation

### CI/CD Templates
✅ GitHub Actions workflows (CI, Release)
✅ Kind cluster configuration for testing
✅ Multi-architecture Dockerfile
✅ Golangci-lint configuration

### Documentation
✅ **CI/CD guide** (docs/ci-cd.md)
  - Complete workflow setup and customization
✅ **Packaging guide** (docs/packaging.md)
  - Helm, OLM, Kustomize distribution strategies
✅ **Testing guide** (docs/testing-guide.md)
  - Unit, integration, E2E testing strategies

The skill is ready to use for building production-ready Kubernetes operators with Go and controller-runtime!

**Total: 25+ files, 4500+ lines of code and documentation**
