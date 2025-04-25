.PHONY: build test clean install lint

# Binary name
BINARY_NAME=ekssm
BUILD_DIR=bin

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
MAIN_PATH=./cmd/ekssm

# Default target
all: test build

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run tests
test:
	$(GOTEST) -v $(shell go list ./... | grep -v /tmp/)

# Clean up binaries
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Install the binary to GOPATH/bin
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

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
	git tag v$(VERSION)
	git push origin v$(VERSION)

# Run the application locally
run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)