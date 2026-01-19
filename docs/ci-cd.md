# CI/CD Guide for Kubernetes Operators

This guide explains how to set up continuous integration and continuous deployment for Kubernetes operators using the provided templates.

## Quick Start

1. **Copy the templates to your project:**
   ```bash
   cp -r templates/.github .github/
   cp templates/Dockerfile.multiarch Dockerfile
   cp templates/.golangci.yml .golangci.yml
   cp templates/.github/workflows/kind-config.yaml .github/workflows/
   ```

2. **Update the workflow files:**
   - Replace `your.repository` references with your actual repository name
   - Configure your image registry (defaults to `ghcr.io`)
   - Adjust Go version if needed

3. **Commit and push:**
   ```bash
   git add .
   git commit -m "Add CI/CD workflows"
   git push
   ```

4. **Watch your workflows run at:**
   ```
   https://github.com/YOUR_ORG/YOUR_REPO/actions
   ```

---

## Workflow Files

### CI Workflow (`.github/workflows/ci.yml`)

The CI workflow runs on every push and pull request:

**Jobs:**

1. **Lint and Unit Tests**
   - `go fmt` - Check code formatting
   - `go vet` - Run Go vet
   - `golangci-lint` - Comprehensive linting
   - Unit tests with race detection
   - Coverage upload to Codecov

2. **Verify Manifests**
   - Generate CRD manifests
   - Generate code
   - Verify no uncommitted changes

3. **Build Images**
   - Multi-architecture builds (amd64, arm64)
   - Push to container registry
   - SBOM and provenance generation

4. **Integration Tests**
   - Spin up Kind cluster
   - Install CRDs
   - Run integration tests
   - Clean up cluster

5. **Security Scanning**
   - Trivy vulnerability scanner
   - Upload results to Security tab

6. **Release** (tagged commits only)
   - Create GitHub release
   - Generate release notes

### Release Workflow (`.github/workflows/release.yml`)

The release workflow runs on version tags (`v*`):

**Jobs:**

1. **Create Release**
   - Generate changelog
   - Create GitHub release
   - Mark as pre-release for rc/alpha/beta tags

2. **Build Release Images**
   - Multi-architecture builds
   - Tag with version and latest
   - Push to registry

3. **Generate Artifacts**
   - `manifests.yaml` - CRD manifests
   - `bundle.yaml` - Complete deployment bundle
   - Upload to release

4. **Package Helm Chart** (optional)
   - Package Helm chart
   - Upload to release

5. **Publish to OCI Registry** (optional)
   - Push Helm chart to OCI registry

---

## Required GitHub Secrets

Most workflows use built-in `GITHUB_TOKEN` for authentication. Additional secrets may be needed:

### For Container Registry Pushing

**GitHub Container Registry (ghcr.io):**
- Uses `GITHUB_TOKEN` automatically

**Docker Hub:**
```
DOCKER_USERNAME
DOCKER_PASSWORD
```

**Other Registries:**
```
REGISTRY_USERNAME
REGISTRY_PASSWORD
```

### For Codecov

```
CODECOV_TOKEN
```

Get this from: https://codecov.io/gh/YOUR_ORG/YOUR_REPO

---

## Customization

### Adjusting Go Version

Edit workflow files:
```yaml
env:
  GO_VERSION: '1.21'  # Change to your version
```

### Changing Image Registry

Edit workflow files:
```yaml
env:
  IMAGE_REGISTRY: ghcr.io  # Change to docker.io, gcr.io, etc.
```

### Modifying Linting Rules

Edit `.golangci.yml` to enable/disable linters or adjust settings.

### Adding Custom Tests

Add to your project:
```go
// +build integration

package controllers_test

func TestIntegration(t *testing.T) {
    // Your integration tests
}
```

Then update CI workflow to run:
```yaml
- name: Run integration tests
  run: go test -v -tags=integration ./...
```

---

## Local Development

### Running CI Checks Locally

Before pushing, run CI checks locally:

```bash
# Format check
go fmt ./...
if [ -n "$(gofmt -l .)" ]; then
  echo "Code is not formatted"
  exit 1
fi

# Vet
go vet ./...

# Lint
golangci-lint run

# Tests
go test -v -race ./...

# Verify manifests
make manifests
make generate
git diff --exit-code
```

### Testing Kind Cluster Locally

```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Create cluster
kind create cluster --image=kindest/node:v1.29.0

# Install CRDs
kubectl apply -f config/crd/bases/

# Run integration tests
KUBEBUILDER_ASSETS=$(setup-envtest use -p path 1.29.0) \
  go test -v -tags=integration ./...

# Delete cluster
kind delete cluster
```

---

## Release Process

### Creating a Release

1. **Update version in code:**
   ```go
   // main.go
   var Version = "v1.0.0"
   ```

2. **Tag and push:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. **Workflow automatically:**
   - Runs all CI checks
   - Builds and pushes images
   - Creates GitHub release
   - Generates artifacts

### Semantic Versioning

Follow semver for versioning:

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality
- **PATCH**: Backwards-compatible bug fixes

Examples:
- `v1.0.0` - First stable release
- `v1.1.0` - New feature
- `v1.1.1` - Bug fix
- `v2.0.0` - Breaking changes

### Pre-releases

Use suffixes for pre-releases:
- `v1.0.0-rc.1` - Release candidate
- `v1.0.0-alpha.1` - Alpha release
- `v1.0.0-beta.1` - Beta release

These are automatically marked as pre-releases in GitHub.

---

## Monitoring CI/CD

### GitHub Actions

View workflow runs at:
```
https://github.com/YOUR_ORG/YOUR_REPO/actions
```

### Required Checks for Merge Protection

Configure branch protection to require:
- CI workflow to pass
- Security scan to pass
- Integration tests to pass

Settings → Branches → Add Rule → Require status checks to pass

### Badge

Add CI badge to README.md:
```markdown
![CI](https://github.com/YOUR_ORG/YOUR_REPO/actions/workflows/ci.yml/badge.svg)
```

---

## Troubleshooting

### Workflow Fails on "go mod tidy"

Run locally:
```bash
go mod tidy
git add go.mod go.sum
```

### Image Push Fails

Check permissions:
1. Repository settings → Actions → General
2. Enable "Read and write permissions"
3. Save and re-run workflow

### Integration Tests Timeout

Increase timeout in CI workflow:
```yaml
- name: Run integration tests
  run: go test -v -timeout=20m ./...
```

### Kind Cluster Creation Fails

Check Kind config:
- Ensure Kind version matches node image version
- Verify resource limits in GitHub Actions runner

---

## Best Practices

1. **Always run CI checks locally before pushing**
   ```bash
   make test && make fmt && make vet
   ```

2. **Keep dependencies updated**
   ```bash
   go get -u ./...
   go mod tidy
   ```

3. **Use meaningful commit messages**
   ```
   feat: add new feature
   fix: resolve bug in reconciler
   docs: update README
   ```

4. **Tag releases from main branch**
   ```bash
   git checkout main
   git pull
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

5. **Review security scan results**
   - Check Security tab in GitHub
   - Address critical/high vulnerabilities

6. **Monitor build times**
   - Long-running builds waste resources
   - Cache dependencies to speed up builds

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/)
- [Semantic Versioning](https://semver.org/)
- [Docker Buildx](https://docs.docker.com/buildx/working-with-buildx/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Golangci-lint](https://golangci-lint.run/)
