# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# .PHONY: phony targets
.PHONY: all

all: test

# @note: This would require running `kubebuilder` init first, or downloading
# the kubebuilder binary manually. For this example project, we'll use
# `go run` instead for local development.
#
# CONTROLLER_GEN_CFG=$(shell pwd)/bin/controller-gen
# .PHONY: controller-gen
# controller-gen: ## Download controller-gen locally if necessary.
# 	$(call go-install-tool,$(CONTROLLER_GEN_CFG),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0)

# @note: Running make requires controller-gen to be installed or in PATH
# To install: go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
# Then add to PATH or update the CONTROLLER_GEN variable below
CONTROLLER_GEN ?= controller-gen

# @note: Running make requires kustomize to be installed or in PATH
# To install: go install sigs.k8s.io/kustomize/kustomize/v5@latest
# Then add to PATH or update the KUSTOMIZE variable below
KUSTOMIZE ?= kustomize

# @note: Running tests requires envtest to be set up
# To install envtest: go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
# Then run: setup-envtest use -p environment
ENVTEST ?= go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
ENVTEST_K8S_VERSION = 1.29.0

.PHONY: manifests
manifests: ## Generate manifests e.g. CRD
	$(CONTROLLER_GEN) \
		crd:maxDescLen=0 \
		rbac:roleName=manager-role \
		webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: ## Generate code
	$(CONTROLLER_GEN) \
		object:headerFile="hack/boilerplate.go.txt" \
		paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet manifests generate ## Run tests
	$(ENVTEST) use $(ENVTEST_K8S_VERSION)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-integration
test-integration: fmt vet manifests generate ## Run integration tests
	$(ENVTEST) use $(ENVTEST_K8S_VERSION)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -v -integration

.PHONY: run
run: fmt vet ## Run controller from your host
	go run ./main.go

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

.PHONY: docker-build
docker-build: ## Build docker image with the controller.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the controller.
	docker push ${IMG}

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run --timeout 5m

.PHONY: clean
clean: ## Clean up generated files
	rm -rf bin/
	rm -f cover.out
