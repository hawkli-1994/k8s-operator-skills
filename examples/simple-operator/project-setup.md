# Cocktail Operator - Setup Guide

This is a complete, runnable Kubernetes operator example that manages Cocktail custom resources.

## Prerequisites

- Go 1.21+
- Docker (for building the container image)
- kubectl configured with cluster access
- A Kubernetes cluster (v1.29+ recommended)

### Optional (for full make commands)

- [controller-gen](https://github.com/kubernetes-sigs/controller-tools) v0.14.0+
  ```bash
  go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
  export PATH=$PATH:$(go env GOPATH)/bin
  ```

- [kustomize](https://kustomize.io/) v5.0+
  ```bash
  go install sigs.k8s.io/kustomize/kustomize/v5@latest
  export PATH=$PATH:$(go env GOPATH)/bin
  ```

- [envtest](https://book.kubebuilder.io/reference/envtest.html) (for running tests)
  ```bash
  go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
  setup-envtest use -p environment
  ```

## Quick Start

### 1. Install Dependencies

```bash
cd examples/simple-operator
go mod download
```

### 2. Install CRDs

```bash
kubectl apply -f config/crd/bases/bar.my.domain_cocktails.yaml
```

Verify installation:
```bash
kubectl get crd cocktails.bar.my.domain
```

### 3. Run Locally (Development)

```bash
go run ./main.go
```

The controller will start watching for Cocktail resources.

### 4. Create Sample Resources

In another terminal:

```bash
# Create a mojito cocktail
kubectl apply -f config/samples/bar_v1_cocktail.yaml

# Create a margarita cocktail
kubectl apply -f config/samples/bar_v1_cocktail_margarita.yaml
```

### 5. Watch the Operator

```bash
# List cocktails
kubectl get cocktails

# Watch status updates
kubectl get cocktails -w

# Get detailed information
kubectl describe cocktail cocktail-mojito
```

Expected output should show:
```
NAME              PHASE   READY   AGE
cocktail-mojito   Ready   2       30s
cocktail-margarita Ready   1       15s
```

## Build and Deploy

### Build Container Image

```bash
docker build -t cocktail-operator:latest .
```

### Deploy to Cluster

```bash
# Update image in deployment
cd config/manager
kustomize edit set image controller=cocktail-operator:latest
cd ../..

# Deploy using kustomize
kustomize build config/default | kubectl apply -f -
```

### Verify Deployment

```bash
# Check controller pod
kubectl get pods -n cocktail-system

# View logs
kubectl logs -n cocktail-system -l control-plane=controller-manager -f
```

## Development Workflow

### Running Tests

```bash
# Unit tests (requires envtest)
make test

# Integration tests
make test-integration
```

### Code Generation

If you modify the API types in `api/v1/cocktail_types.go`:

```bash
# Regenerate CRD manifests
make manifests

# Regenerate code
make generate
```

### Linting

```bash
# Format code
make fmt

# Run go vet
make vet

# Run golangci-lint (if installed)
make lint
```

## Cleaning Up

```bash
# Delete sample resources
kubectl delete -f config/samples/

# Delete CRDs
kubectl delete -f config/crd/bases/bar.my.domain_cocktails.yaml

# Delete controller deployment
kustomize build config/default | kubectl delete -f -
```

## What This Operator Does

The Cocktail operator demonstrates:

1. **Custom Resource Definition**: Cocktail CRD with validation
2. **Reconciliation Loop**: Watches and reconciles Cocktail resources
3. **Status Management**: Updates phase and conditions
4. **Finalizers**: Proper cleanup on deletion
5. **Validation**: Enum constraints, min/max values
6. **Printer Columns**: Custom kubectl output columns

### Key Features

- **Spec Fields**:
  - `size`: Number of servings (1-10)
  - `recipe`: Type of cocktail (Mojito, Margarita, OldFashioned, Cosmopolitan)
  - `garnish`: Whether to add garnish
  - `instructions`: Custom preparation instructions

- **Status Fields**:
  - `phase`: Current preparation state
  - `servingsReady`: Number of servings ready
  - `lastPrepared`: Timestamp of last preparation
  - `conditions`: Detailed condition information

## Learning Path

1. **Start here**: Read `api/v1/cocktail_types.go` to understand the resource structure
2. **Controller logic**: Study `controllers/cocktail_controller.go` for the reconciliation pattern
3. **Patterns**: Refer to `/patterns/` directory for reusable patterns
4. **Advanced**: Try adding new fields or business logic to practice

## Troubleshooting

### Controller not running
- Check RBAC permissions: `kubectl get clusterrole manager-role -o yaml`
- View logs: `kubectl logs -n cocktail-system -l control-plane=controller-manager`

### Status not updating
- Verify status subresource: `kubectl get crd cocktails.bar.my.domain -o yaml`
- Check RBAC includes status permissions

### Resource stuck in "Preparing"
- Check controller logs for errors
- Verify recipe is one of the allowed values

## Next Steps

- Explore the `/patterns/` directory for reusable operator patterns
- Check `/examples/database-operator/` for a more complex example
- Read the testing guide in `/docs/testing-guide.md`
- Learn about CI/CD in `/docs/ci-cd.md`
