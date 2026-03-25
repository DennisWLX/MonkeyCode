.PHONY: build run test docker-build docker-run docker-stop docker-buildx

# Image settings
IMAGE_NAME=ghcr.io/chaitin/monkeycode/runner
IMAGE_TAG=latest

# Build for native platform
build:
	@echo "Building runner for native platform..."
	go build -ldflags "-s -w" -o bin/runner ./cmd/runner

# Run in development
run:
	@echo "Running runner in development mode..."
	go run ./cmd/runner

# Run tests
test:
	@echo "Running tests..."
	go test ./... -v

# Build Docker image (native)
docker-build:
	@echo "Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Build Docker image for multiple architectures
docker-buildx:
	@echo "Setting up Docker buildx..."
	docker run --rm --privileged multiarch/qemu-user-static:register --reset
	@echo "Building multi-architecture Docker image..."
	docker buildx create --name mybuilder || true
	docker buildx use mybuilder
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-t $(IMAGE_NAME):$(IMAGE_TAG)-amd64 \
		-t $(IMAGE_NAME):$(IMAGE_TAG)-arm64 \
		--push .

# Run container locally
docker-run:
	@echo "Running runner container..."
	docker run -d \
		--name monkeycode_runner \
		-p 8080:8080 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ./data:/app/data \
		-e TOKEN=your-token \
		-e GRPC_URL=localhost:50051 \
		$(IMAGE_NAME):$(IMAGE_TAG)

# Run container with specific configuration
docker-run-full:
	@echo "Running runner container with full configuration..."
	docker run -d \
		--name monkeycode_runner \
		--restart unless-stopped \
		-p 8080:8080 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${HOST_STORAGE_DIR:-./data}:/app/data \
		-e TOKEN=$$TOKEN \
		-e GRPC_URL=$$GRPC_URL \
		-e GRPC_HOST=$$GRPC_HOST \
		-e GRPC_PORT=$$GRPC_PORT \
		--health-cmd="curl -f http://localhost:8080/health || exit 1" \
		--health-interval=30s \
		--health-timeout=10s \
		--health-retries=3 \
		$(IMAGE_NAME):$(IMAGE_TAG)

# Stop and remove container
docker-stop:
	@echo "Stopping and removing container..."
	docker stop monkeycode_runner || true
	docker rm monkeycode_runner || true

# View logs
docker-logs:
	docker logs -f monkeycode_runner

# Shell into container
docker-shell:
	docker exec -it monkeycode_runner /bin/sh

# Push image to registry
docker-push:
	@echo "Pushing image to registry..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):$(IMAGE_TAG)-amd64 || true
	docker push $(IMAGE_NAME):$(IMAGE_TAG)-arm64 || true

# Clean build artifacts
clean:
	rm -rf bin/
	docker system prune -f

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build runner binary for native platform"
	@echo "  make run            - Run runner in development mode"
	@echo "  make test           - Run tests"
	@echo "  make docker-build   - Build Docker image for native platform"
	@echo "  make docker-buildx  - Build and push multi-architecture images"
	@echo "  make docker-run    - Run container locally"
	@echo "  make docker-stop   - Stop and remove container"
	@echo "  make docker-logs   - View container logs"
	@echo "  make docker-shell  - Shell into container"
	@echo "  make docker-push   - Push image to registry"
	@echo "  make clean         - Clean build artifacts"
