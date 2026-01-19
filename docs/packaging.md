# Packaging and Distribution Guide for Kubernetes Operators

This guide covers different methods for packaging and distributing Kubernetes operators.

## Overview

There are several ways to package and distribute operators:

| Method | Complexity | Use Case |
|--------|-----------|----------|
| **Helm** | Low | Simple deployments, customization |
| **OLM** | Medium | OpenShift clusters, OperatorHub |
| **Kustomize** | Low | GitOps workflows |
| **Static Manifests** | Low | Simple deployments, version control |

---

## 1. Helm Charts

Helm is the most popular package manager for Kubernetes. It's ideal for operators that need configuration flexibility.

### Quick Start

Install Helm:
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Chart Structure

```
helm/my-operator/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration values
├── values.schema.json      # JSON schema for values (optional)
└── templates/
    ├── deployment.yaml     # Controller deployment
    ├── serviceaccount.yaml # Service account
    ├── rbac/
    │   ├── role.yaml
    │   ├── rolebinding.yaml
    │   └── serviceaccount.yaml
    ├── crds/
    │   └── crd.yaml       # CRD definitions
    └── templates/          # Additional templates
```

### Example Chart.yaml

```yaml
apiVersion: v2
name: my-operator
description: A Helm chart for my Kubernetes operator
type: application
version: 1.0.0        # Chart version
appVersion: "1.0.0"   # Operator version
keywords:
  - kubernetes-operator
  - database
home: https://github.com/your-org/my-operator
sources:
  - https://github.com/your-org/my-operator
maintainers:
  - name: Your Name
    email: you@example.com
icon: https://example.com/icon.png
annotations:
  artifacthub.io/category: database
  artifacthub.io/license: Apache-2.0
```

### Example values.yaml

```yaml
# Controller configuration
controller:
  image:
    repository: ghcr.io/your-org/my-operator
    tag: "1.0.0"
    pullPolicy: IfNotPresent

  replicas: 1

  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

  # Node selector
  nodeSelector: {}

  # Tolerations
  tolerations: []

  # Affinity
  affinity: {}

# Service configuration
service:
  type: ClusterIP
  annotations: {}

# RBAC configuration
rbac:
  create: true

# CRD installation
crds:
  install: true
  keep: true  # Keep CRDs when uninstalling

# Logging configuration
log:
  level: info
  format: json

# Metrics configuration
metrics:
  enabled: true
  serviceMonitor:
    enabled: false
```

### Example Template

```yaml
# templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "my-operator.fullname" . }}
  labels:
    {{- include "my-operator.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.controller.replicas }}
  selector:
    matchLabels:
      {{- include "my-operator.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "my-operator.selectorLabels" . | nindent 8 }}
    spec:
      serviceAccountName: {{ include "my-operator.serviceAccountName" . }}
      containers:
      - name: manager
        image: "{{ .Values.controller.image.repository }}:{{ .Values.controller.image.tag }}"
        imagePullPolicy: {{ .Values.controller.image.pullPolicy }}
        args:
        - --leader-elect
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        resources:
          {{- toYaml .Values.controller.resources | nindent 10 }}
```

### Packaging and Publishing

```bash
# Package chart
helm package helm/my-operator/

# Test chart locally
helm install my-operator ./my-operator --dry-run --debug

# Install chart
helm install my-operator ./my-operator --namespace my-operator --create-namespace

# Upgrade chart
helm upgrade my-operator ./my-operator

# Uninstall chart
helm uninstall my-operator --namespace my-operator
```

### Publishing to Helm Repository

```bash
# Create index
helm repo index .

# Publish to GitHub Pages (example)
mkdir gh-pages
cp my-operator-*.tgz gh-pages/
cd gh-pages
helm repo index .
git add .
git commit -m "Add chart version"
git push
```

---

## 2. Operator Lifecycle Manager (OLM)

OLM is the default operator management system for OpenShift and is available for Kubernetes via OperatorHub.

### Bundle Structure

```
bundle/
├── manifests/
│   ├── my-operator.clusterserviceversion.yaml
│   ├── my.domain_databases.yaml
│   └── my-operator-controller-metrics-service.yaml
├── metadata/
│   └── annotations.yaml
└── bundle.Dockerfile
```

### ClusterServiceVersion (CSV)

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: my-operator.v1.0.0
  annotations:
    alm-examples: |-
      [{"apiVersion":"my.domain/v1","kind":"Database","metadata":{"name":"example"}}]
    capabilities: Basic Install
    categories: "Database"
    certified: "false"
    description: "Database Operator"
    repository: "https://github.com/your-org/my-operator"
    support: "Your Name"
    olm.skipRange: '>=1.0.0 <1.0.1'
spec:
  displayName: My Operator
  description: |
    A Kubernetes operator for managing databases

  icon:
  - base64data: PHN2Zy4uLjwvc3ZnPg==
    mediatype: image/svg+xml

  keywords:
  - database
  - postgres

  links:
  - name: Source Code
    url: https://github.com/your-org/my-operator

  maintainers:
  - email: you@example.com
    name: Your Name

  version: 1.0.0
  maturity: stable

  installs:
  - spec:
      deployments:
      - name: my-operator
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: my-operator
          template:
            metadata:
              labels:
                app: my-operator
            spec:
              serviceAccountName: my-operator
              containers:
              - name: manager
                image: ghcr.io/your-org/my-operator:1.0.0
    strategy: deployment

  customresourcedefinitions:
    owned:
    - description: Database is the Schema for the databases API
      displayName: Database
      kind: Database
      name: databases.my.domain
      version: v1

  permissions:
  - rules:
    - apiGroups:
      - my.domain
      resources:
      - databases
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    serviceAccountName: my-operator
```

### Building and Pushing Bundle

```bash
# Install opm
go install github.com/operator-framework/operator-registry/cmd/opm@latest

# Build bundle image
docker build -f bundle.Dockerfile -t ghcr.io/your-org/my-operator-bundle:v1.0.0 .

# Push bundle
docker push ghcr.io/your-org/my-operator-bundle:v1.0.0

# Create index
opm index add \
  --bundles ghcr.io/your-org/my-operator-bundle:v1.0.0 \
  --tag ghcr.io/your-org/my-operator-index:latest \
  -c docker

# Push index
docker push ghcr.io/your-org/my-operator-index:latest
```

### Submitting to OperatorHub

See [OperatorHub documentation](https://operatorhub.io/preview) for detailed steps.

---

## 3. Kustomize

Kustomize is ideal for GitOps workflows and simple patching strategies.

### Base Structure

```
config/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── serviceaccount.yaml
│   └── rbac.yaml
└── overlays/
    ├── production/
    │   ├── kustomization.yaml
    │   └── patches/
    └── development/
        ├── kustomization.yaml
        └── patches/
```

### Base kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- deployment.yaml
- serviceaccount.yaml
- rbac.yaml
- crd/

# Common labels applied to all resources
commonLabels:
  app.kubernetes.io/name: my-operator
  app.kubernetes.io/managed-by: kustomize

# Images to replace
images:
- name: controller
  newName: ghcr.io/your-org/my-operator
  newTag: v1.0.0
```

### Overlay Example

```yaml
# overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: my-operator-prod

resources:
- ../../base

patchesStrategicMerge:
- patches/deployment-replicas.yaml
```

```yaml
# overlays/production/patches/deployment-replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-operator
spec:
  replicas: 3
```

### Applying with Kustomize

```bash
# Apply base
kubectl apply -k config/base/

# Apply overlay
kubectl apply -k config/overlays/production/
```

---

## 4. Static Manifests

For simple deployments, static YAML manifests work well.

### Structure

```
manifests/
├── namespace.yaml
├── crds.yaml
├── serviceaccount.yaml
├── rbac.yaml
└── deployment.yaml
```

### Applying Manifests

```bash
kubectl apply -f manifests/
```

---

## Release Strategy

### Semantic Versioning

- **v1.0.0** - First stable release
- **v1.1.0** - New features, backward compatible
- **v1.0.1** - Bug fix
- **v2.0.0** - Breaking changes

### Version Compatibility Matrix

| Operator Version | API Version | Kubernetes |
|-----------------|-------------|------------|
| v1.x | my.domain/v1 | 1.27+ |
| v2.x | my.domain/v1, my.domain/v2 | 1.29+ |

### Upgrade Path

```
v1.0.0 → v1.1.0 → v1.2.0 → v2.0.0
  ✓         ✓         ⚠️ (breaking)
```

---

## Multi-Architecture Builds

Support multiple CPU architectures:

```bash
# Build for amd64 and arm64
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/your-org/my-operator:v1.0.0 \
  --push \
  .
```

---

## Distribution Checklist

### Before Release:

- [ ] All tests pass
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version numbers updated
- [ ] Security scan passed
- [ ] Multi-arch images built
- [ ] Helm chart tested
- [ ] OLM bundle validated

### After Release:

- [ ] Tag created in git
- [ ] GitHub release created
- [ ] Helm chart published
- [ ] OLM bundle published
- [ ] Documentation published
- [ ] Release notes sent to users

---

## Troubleshooting

### Helm Chart Issues

**Chart install fails:**
```bash
# Debug with --dry-run
helm install my-operator ./my-operator --dry-run --debug

# Check templates
helm template my-operator ./my-operator
```

### OLM Bundle Issues

**CSV validation fails:**
```bash
# Validate CSV
operator-csv my-operator.clusterserviceversion.yaml
```

### Manifest Issues

**Missing CRDs:**
```bash
# Check CRDs
kubectl get crd | grep my.domain

# Apply CRDs first
kubectl apply -f config/crd/bases/
```

---

## Resources

- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
- [OLM Documentation](https://olm.operatorframework.io/)
- [Kustomize Documentation](https://kustomize.io/)
- [OperatorHub](https://operatorhub.io/)
