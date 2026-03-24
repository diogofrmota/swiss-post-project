BINARY        ?= manager
IMAGE_REGISTRY ?= ghcr.io
IMAGE_REPO    ?= $(IMAGE_REGISTRY)/diogofrmota/custom-operator
TAG           ?= $(shell git rev-parse --short HEAD)
PLATFORM      ?= linux/arm64

.PHONY: all build build-arm64 test lint fmt vet generate manifests docker-build docker-push run install-crds clean help

all: build

## build: Compile the operator binary for the current platform
build:
	go build -o bin/$(BINARY) ./cmd/main/main.go

## build-arm64: Cross-compile for Raspberry Pi (arm64)
build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	  go build -a -ldflags="-s -w" -o bin/$(BINARY)-arm64 ./cmd/main/main.go

## test: Run all unit tests with race detector
test:
	go test ./... -v -race -coverprofile=cover.out
	go tool cover -func=cover.out

## lint: Run golangci-lint
lint:
	golangci-lint run --config .golangci.yaml ./...

## fmt: Format Go source files
fmt:
	gofmt -s -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## generate: Regenerate DeepCopy methods (requires controller-gen)
generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

## manifests: Regenerate CRD manifests (requires controller-gen)
manifests:
	controller-gen crd paths="./..." output:crd:artifacts:config=../infrastructure/overlays/production

## docker-build: Build the Docker image for arm64
docker-build:
	docker buildx build \
	  --platform $(PLATFORM) \
	  --tag $(IMAGE_REPO):$(TAG) \
	  --tag $(IMAGE_REPO):latest \
	  --file Dockerfile \
	  .

## docker-push: Build and push the Docker image to the registry
docker-push:
	docker buildx build \
	  --platform $(PLATFORM) \
	  --tag $(IMAGE_REPO):$(TAG) \
	  --tag $(IMAGE_REPO):latest \
	  --file Dockerfile \
	  --push \
	  .

## run: Run the operator locally against the current kubeconfig context
run: fmt vet
	go run ./cmd/main/main.go \
	  --metrics-bind-address=:8080 \
	  --health-probe-bind-address=:8081

## install-crds: Apply the CRD to the current cluster
install-crds:
	kubectl apply -f ../infrastructure/overlays/production/crd.yaml

## clean: Remove build artifacts
clean:
	rm -rf bin/ cover.out

## help: Print this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'


