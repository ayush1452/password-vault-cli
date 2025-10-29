# Password Vault CLI - Build System
# Provides cross-platform build, test, and release automation

# Build configuration
BINARY_NAME := vault
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -s -w"

# Go configuration
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOMOD := $(GO) mod
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOCLEAN := $(GO) clean
GOGET := $(GO) get
GOFMT := gofmt
GOLINT := golangci-lint

# Directories
BUILD_DIR := build
DIST_DIR := dist
DOCS_DIR := docs
TESTS_DIR := tests

# Platform targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: all build clean test lint fmt help install uninstall dev-setup
.PHONY: test-unit test-integration test-security test-fuzz test-acceptance
.PHONY: build-all release docker coverage security-scan
.PHONY: docs docs-serve benchmark profile

# Default target
all: clean fmt lint test build

# Help target
help: ## Show this help message
	@echo "$(BLUE)Password Vault CLI - Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development setup
dev-setup: ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	$(GOMOD) download
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/securecodewarrior/sast-scan
	@echo "$(GREEN)Development environment ready!$(NC)"

# Build targets
build: ## Build binary for current platform
	@echo "$(BLUE)Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vault
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-all: ## Build binaries for all platforms
	@echo "$(BLUE)Building for all platforms...$(NC)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} $(MAKE) build-platform PLATFORM=$$platform; \
	done
	@echo "$(GREEN)All builds complete!$(NC)"

build-platform:
	@echo "Building for $(PLATFORM)..."
	@GOOS=$(shell echo $(PLATFORM) | cut -d'/' -f1) \
	 GOARCH=$(shell echo $(PLATFORM) | cut -d'/' -f2) \
	 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-$(shell echo $(PLATFORM) | tr '/' '-') ./cmd/vault

# Install/Uninstall
install: build ## Install binary to system PATH
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)Installation complete!$(NC)"

uninstall: ## Remove binary from system PATH
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)...$(NC)"
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)Uninstallation complete!$(NC)"

# Code quality
fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GOFMT) -s -w .
	@goimports -w .
	@echo "$(GREEN)Code formatting complete!$(NC)"

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@$(GOLINT) run ./...
	@echo "$(GREEN)Linting complete!$(NC)"

fix: ## Auto-fix linting issues
	@echo "$(BLUE)Auto-fixing linting issues...$(NC)"
	@$(GOLINT) run --fix ./...
	@echo "$(GREEN)Auto-fix complete!$(NC)"

# Testing
test: ## Run all tests
	@echo "$(BLUE)Running all tests...$(NC)"
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)All tests passed!$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	@$(GOTEST) -v -race -short ./internal/...
	@echo "$(GREEN)Unit tests passed!$(NC)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@$(GOTEST) -v -race -tags=integration ./tests/integration/...
	@echo "$(GREEN)Integration tests passed!$(NC)"

test-security: ## Run security tests
	@echo "$(BLUE)Running security tests...$(NC)"
	@$(GOTEST) -v -race -timeout=30m ./tests/security_tests.go
	@echo "$(GREEN)Security tests passed!$(NC)"

test-fuzz: ## Run fuzz tests
	@echo "$(BLUE)Running fuzz tests...$(NC)"
	@$(GOTEST) -v -race -timeout=10m ./tests/fuzz_tests.go
	@echo "$(GREEN)Fuzz tests passed!$(NC)"

test-acceptance: ## Run acceptance tests
	@echo "$(BLUE)Running acceptance tests...$(NC)"
	@$(GOTEST) -v -race -timeout=15m ./tests/acceptance_tests.go
	@echo "$(GREEN)Acceptance tests passed!$(NC)"

test-comprehensive: ## Run comprehensive test suite
	@echo "$(BLUE)Running comprehensive test suite...$(NC)"
	@./tests/run_all_tests.sh
	@echo "$(GREEN)Comprehensive testing complete!$(NC)"

# Coverage
coverage: ## Generate test coverage report
	@echo "$(BLUE)Generating coverage report...$(NC)"
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@$(GO) tool cover -func=coverage.out | tail -1
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

# Benchmarking
benchmark: ## Run performance benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./internal/vault/
	@echo "$(GREEN)Benchmarks complete!$(NC)"

profile: ## Generate performance profiles
	@echo "$(BLUE)Generating performance profiles...$(NC)"
	@$(GO) tool pprof -http=:8080 cpu.prof &
	@echo "$(GREEN)Profile server running at http://localhost:8080$(NC)"

# Security
security-scan: ## Run security vulnerability scan
	@echo "$(BLUE)Running security scan...$(NC)"
	@gosec -fmt json -out security-report.json ./...
	@echo "$(GREEN)Security scan complete: security-report.json$(NC)"

audit: ## Audit dependencies for vulnerabilities
	@echo "$(BLUE)Auditing dependencies...$(NC)"
	@$(GO) list -json -deps ./... | nancy sleuth
	@echo "$(GREEN)Dependency audit complete!$(NC)"

# Documentation
docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	@godoc -http=:6060 &
	@echo "$(GREEN)Documentation server running at http://localhost:6060$(NC)"

docs-serve: ## Serve documentation locally
	@echo "$(BLUE)Serving documentation...$(NC)"
	@cd $(DOCS_DIR) && python3 -m http.server 8000
	@echo "$(GREEN)Documentation server running at http://localhost:8000$(NC)"

# Release
release: clean test-comprehensive build-all ## Create release artifacts
	@echo "$(BLUE)Creating release artifacts...$(NC)"
	@mkdir -p $(DIST_DIR)/release
	@for platform in $(PLATFORMS); do \
		platform_name=$$(echo $$platform | tr '/' '-'); \
		binary_name=$(BINARY_NAME)-$$platform_name; \
		if [ "$${platform%/*}" = "windows" ]; then \
			binary_name=$$binary_name.exe; \
		fi; \
		cp $(DIST_DIR)/$$binary_name $(DIST_DIR)/release/; \
		tar -czf $(DIST_DIR)/release/$(BINARY_NAME)-$(VERSION)-$$platform_name.tar.gz \
			-C $(DIST_DIR)/release $$binary_name \
			-C ../../../ README.md LICENSE SECURITY.md; \
	done
	@echo "$(GREEN)Release artifacts created in $(DIST_DIR)/release/$(NC)"

# Docker
docker: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "$(GREEN)Docker image built: $(BINARY_NAME):$(VERSION)$(NC)"

# Cleanup
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@rm -f cpu.prof mem.prof
	@rm -f security-report.json
	@$(GOCLEAN)
	@echo "$(GREEN)Cleanup complete!$(NC)"

# Development helpers
dev-build: ## Quick development build
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vault

dev-test: ## Quick development test
	@$(GOTEST) -short ./...

dev-run: dev-build ## Build and run for development
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

# CI/CD helpers
ci-setup: ## Setup CI environment
	@echo "$(BLUE)Setting up CI environment...$(NC)"
	@$(GOMOD) download
	@echo "$(GREEN)CI environment ready!$(NC)"

ci-test: ## Run tests in CI environment
	@echo "$(BLUE)Running CI tests...$(NC)"
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out
	@echo "$(GREEN)CI tests complete!$(NC)"

ci-build: ## Build in CI environment
	@echo "$(BLUE)Building in CI...$(NC)"
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/vault
	@echo "$(GREEN)CI build complete!$(NC)"

# Version information
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell $(GO) version)"
	@echo "Platform: $(GOOS)/$(GOARCH)"

# Pre-commit hooks
pre-commit: fmt lint test ## Run pre-commit checks
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"

# Local development server
dev-server: build ## Run development server
	@echo "$(BLUE)Starting development server...$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --config dev-config.yaml

# Database operations
db-migrate: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME) migrate
	@echo "$(GREEN)Migrations complete!$(NC)"

# Performance testing
perf-test: build ## Run performance tests
	@echo "$(BLUE)Running performance tests...$(NC)"
	@./scripts/performance-test.sh
	@echo "$(GREEN)Performance tests complete!$(NC)"

# Integration with external tools
validate-config: ## Validate configuration files
	@echo "$(BLUE)Validating configuration...$(NC)"
	@yamllint config/*.yaml
	@echo "$(GREEN)Configuration validation complete!$(NC)"

# Show build information
info: ## Show build environment information
	@echo "$(BLUE)Build Environment Information:$(NC)"
	@echo "Go Version: $(shell $(GO) version)"
	@echo "Go Path: $(shell $(GO) env GOPATH)"
	@echo "Go Root: $(shell $(GO) env GOROOT)"
	@echo "Platform: $(GOOS)/$(GOARCH)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Demo targets
demo-cli: ## Run CLI demo
	@echo "$(BLUE)Running CLI demo...$(NC)"
	@$(GO) run ./examples/cli/
	@echo "$(GREEN)CLI demo complete!$(NC)"

demo-crypto: ## Run cryptography demo
	@echo "$(BLUE)Running cryptography demo...$(NC)"
	@$(GO) run ./examples/crypto/
	@echo "$(GREEN)Cryptography demo complete!$(NC)"

demo-storage: ## Run storage layer demo
	@echo "$(BLUE)Running storage layer demo...$(NC)"
	@$(GO) run ./examples/storage/
	@echo "$(GREEN)Storage demo complete!$(NC)"

demo-all: demo-crypto demo-storage ## Run all demos
	@echo "$(GREEN)All demos complete!$(NC)"