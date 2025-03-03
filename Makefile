# Build configuration
BINARY_NAME=winterflow-agent
VERSION=$(shell date +'%Y%m%d.%H%M%S')
BUILD_DIR=build

# Go build flags
LDFLAGS=-X main.version=${VERSION}
BUILD_FLAGS=-v -ldflags="${LDFLAGS}"

.PHONY: all clean build

all: build

# Build for local development
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/main.go

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -rf ${BUILD_DIR}

# Show help
help:
	@echo "Available targets:"
	@echo "  build (default) - Build the binary"
	@echo "  clean          - Remove build artifacts"
	@echo "  help           - Show this help message" 