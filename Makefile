IMG ?= quay.io/mfagotto/ip-rewrite-operator:0.2.0
INIT_IMG ?= quay.io/mfagotto/ip-rewrite-operator:init-0.2.0
BUNDLE_IMG ?= quay.io/mfagotto/ip-rewrite-operator:bundle-0.2.0
OPERATOR_SDK ?= ./bin/operator-sdk
OPERATOR_SDK_VERSION ?= v1.41.1

.PHONY: all
all: test build

.PHONY: build
build:
	go build -o bin/manager ./cmd/

.PHONY: test
test:
	go test ./... -count=1

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: docker-build
docker-build:
	podman build -t $(IMG) -f Dockerfile .

.PHONY: docker-build-init
docker-build-init:
	podman build -t $(INIT_IMG) -f images/init/Containerfile images/init/

.PHONY: deploy
deploy:
	kubectl apply -k config/

.PHONY: undeploy
undeploy:
	kubectl delete -k config/ --ignore-not-found

.PHONY: operator-sdk
operator-sdk:
	@if [ ! -x "$(OPERATOR_SDK)" ]; then \
	  mkdir -p bin; \
	  curl -sL "https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_linux_amd64" -o "$(OPERATOR_SDK)"; \
	  chmod +x "$(OPERATOR_SDK)"; \
	fi
	@$(OPERATOR_SDK) version

.PHONY: bundle
bundle:
	@echo "OLM bundle is maintained under bundle/ (CSV + CRD + webhook Service/MWC + SCC)"
	@ls -la bundle/manifests bundle/metadata

.PHONY: bundle-validate
bundle-validate: operator-sdk
	./hack/bundle-validate.sh

.PHONY: bundle-build
bundle-build:
	podman build -t $(BUNDLE_IMG) -f bundle.Dockerfile .

.PHONY: cluster-validate
cluster-validate:
	./hack/cluster-validate.sh
