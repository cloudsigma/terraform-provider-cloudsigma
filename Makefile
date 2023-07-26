# Project variables
PROJECT_NAME := terraform-provider-cloudsigma
PKG_NAME = cloudsigma

# Build variables
.DEFAULT_GOAL = test
BUILD_DIR := build
DEV_GOARCH := $(shell go env GOARCH)
DEV_GOOS := $(shell go env GOOS)
EXE =
ifeq ($(DEV_GOOS),windows)
EXE = .exe
endif


## tools: Install required tooling.
.PHONY: tools
tools:
	@echo "==> Installing required tooling..."
	@cd tools && go install github.com/golangci/golangci-lint/cmd/golangci-lint

## clean: Delete the build directory.
.PHONY: clean
clean:
	@echo "==> Removing '$(BUILD_DIR)' directory..."
	@rm -rf $(BUILD_DIR)

## lint: Lint code with golangci-lint.
.PHONY: lint
lint:
	@echo "==> Linting code with 'golangci-lint'..."
	@golangci-lint run ./...

## test: Run all unit tests.
.PHONY: test
test:
	@echo "==> Running unit tests..."
	@mkdir -p $(BUILD_DIR)
	@go test -count=1 -v -cover -coverprofile=$(BUILD_DIR)/coverage.out -parallel=4 ./...

## testacc: Run all acceptance tests.
.PHONY: testacc
testacc: lint
	@echo "==> Running acceptance tests..."
	@mkdir -p $(BUILD_DIR)
	TF_ACC=1 go test -count=1 -v -cover -coverprofile=$(BUILD_DIR)/coverage-with-acceptance.out -timeout 120m ./...

## build: Build binary for default local system's operating system and architecture.
.PHONY: build
build:
	@echo "==> Building binary..."
	@echo "    running go build for GOOS=$(DEV_GOOS) GOARCH=$(DEV_GOARCH)"
	@go build -o $(BUILD_DIR)/$(PROJECT_NAME)$(EXE) main.go


help: Makefile
	@echo "Usage: make <command>"
	@echo ""
	@echo "Commands:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
