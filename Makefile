IMG ?= ghcr.io/mfagotto/ip-rewrite-operator:latest
INIT_IMG ?= ghcr.io/mfagotto/ip-rewrite-init:latest

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
	kubectl apply -k config/default

.PHONY: undeploy
undeploy:
	kubectl delete -k config/default --ignore-not-found
