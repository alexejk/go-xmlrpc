# We don't need make's built-in rules.
MAKEFLAGS += --no-builtin-rules

GO_FLAGS= CGO_ENABLED=0
GO_LDFLAGS= -ldflags=""
GO_BUILD_CMD=$(GO_FLAGS) go build $(GO_LDFLAGS)

BINARY_NAME=$(APP_NAME)
BUILD_DIR=build

.PHONY: all
all: clean generate-all lint test build-all package-all

#--------------------------------
# Validation steps
#--------------------------------

.PHONY: lint
lint:
	@echo "Linting code..."
	@go vet ./...

.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

#--------------------------------
# Code generation steps
#--------------------------------

.PHONY: code-gen
code-gen:
	@echo "Generating code..."
	@go generate ./...

.PHONY: generate-all
generate-all: code-gen

#--------------------------------
# Build steps
#--------------------------------

.PHONY: pre-build
pre-build:
	@mkdir -p $(BUILD_DIR)

.PHONY: build-linux
build-linux: pre-build
	@echo "Building Linux binary..."
	GOOS=linux GOARCH=amd64 $(GO_BUILD_CMD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64

.PHONY: build-osx
build-osx: pre-build
	@echo "Building OSX binary..."
	GOOS=darwin GOARCH=amd64 $(GO_BUILD_CMD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64

.PHONY: build build-all
build build-all: build-linux build-osx

#--------------------------------
# Package steps
#--------------------------------

.PHONY: package-linux
package-linux:
	@echo "Packaging Linux binary..."
	tar -C $(BUILD_DIR) -zcf $(BUILD_DIR)/$(BINARY_NAME)-$(APP_VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64

.PHONY: package-osx
package-osx:
	@echo "Packaging OSX binary..."
	tar -C $(BUILD_DIR) -zcf $(BUILD_DIR)/$(BINARY_NAME)-$(APP_VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64

.PHONY: package-all
package-all: package-linux package-osx

#--------------------------------
# Docker steps
#--------------------------------

.PHONY: docker
docker:
# Build a new image (delete old one)
	docker build --force-rm --build-arg GOPROXY -t $(BINARY_NAME) .

.PHONY: build-in-docker
build-in-docker: docker
# Force-stop any containers with this name
	docker rm -f $(BINARY_NAME) || true
# Create a new container with newly built image (but don't run it)
	docker create --name $(BINARY_NAME) $(BINARY_NAME)
# Copy over the binary to disk (from container)
	docker cp '$(BINARY_NAME):/opt/' $(BUILD_DIR)
# House-keeping: removing container
	docker rm -f $(BINARY_NAME)

#--------------------------------
# Cleanup steps
#--------------------------------

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -Rf $(BUILD_DIR)
