.PHONY: build test clean install lint

# Binary name
BINARY_NAME=ekssm
CMD_PATH=./cmd/$(BINARY_NAME)
BUILD_DIR=bin

# Versioning
# Get the latest Git tag, or 'v0.0.0' if no tags exist
# Remove --always so it fails and falls back to echo if no tag found
GIT_TAG=$(shell git describe --tags --abbrev=0 --match='v*' 2>/dev/null || echo "v0.0.0")
# Get the short commit hash
GIT_COMMIT=$(shell git rev-parse --short HEAD)
# Build timestamp
BUILD_TIMESTAMP=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
# Combine for version string (e.g., v1.0.0+a1b2c3d)
VERSION_STRING=$(GIT_TAG)+$(GIT_COMMIT)

# Go parameters
GOCMD=go
GOPATH=$(shell go env GOPATH)
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint
# Values for Go build flags
LD_FLAGS="-X main.version=$(VERSION_STRING)"
GC_FLAGS=all=-trimpath=$(PWD)
# Use vendor directory
GO_FLAGS=-mod=vendor

# Default target
all: test build

# Build the binary
build: 
	@echo "Building $(BINARY_NAME) version $(VERSION_STRING)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(GO_FLAGS) -gcflags=$(GC_FLAGS) -ldflags=$(LD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	$(GOTEST) -v $(shell go list ./... | grep -v /tmp/)

# Clean up binaries
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installation complete."

# Run linter
lint:
	$(GOLINT) run

# Format code
fmt:
	$(GOFMT) ./...

# Update dependencies
deps:
	$(GOMOD) tidy

# Tag a new release
release:
	@echo "Tagging release $(VERSION)..."
	@git tag v$(VERSION)
	@git push origin v$(VERSION)