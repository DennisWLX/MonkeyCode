.PHONY: all build clean install test

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=installer
BINARY_NAME_WIN=installer.exe

# Build flags
LDFLAGS=-ldflags "-s -w"

# Directories
BUILD_DIR=build
DIST_DIR=dist

all: clean test build

build:
	@echo "Building installer..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux/$(BINARY_NAME) .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux/$(BINARY_NAME)-arm64 .

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)/darwin
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin/$(BINARY_NAME) .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin/$(BINARY_NAME)-arm64 .

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/windows/$(BINARY_NAME_WIN) .

# Build all platforms
build-all: build-linux build-darwin build-windows
	@echo "Build complete!"
	@ls -la $(BUILD_DIR)/*/

# Cross-compile for all platforms
cross-compile:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/installer_linux_amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/installer_linux_arm64 .
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/installer_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/installer_darwin_arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/installer_windows_amd64.exe .

	@echo "Cross-compile complete!"
	@ls -lh $(BUILD_DIR)

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

install: build
	@echo "Installing installer..."
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/installer

# Development
dev:
	$(GOCMD) run .

fmt:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

lint: vet fmt
	@echo "Linting complete!"

# Docker build (for release)
docker-build:
	@echo "Building Docker image for installer..."
	docker build -t monkeycode-installer:latest .

# Upload to release (requires GitHub CLI)
release: cross-compile
	@echo "Creating release..."
	gh release create $(VERSION) \
		--title "Installer $(VERSION)" \
		--notes "MonkeyCode Runner Installer $(VERSION)" \
		$(BUILD_DIR)/*

# Help
help:
	@echo "Available targets:"
	@echo "  make build         - Build for current platform"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make cross-compile - Cross-compile for all platforms"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install       - Install to /usr/local/bin"
	@echo "  make dev           - Run in development mode"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run linter (fmt + vet)"
