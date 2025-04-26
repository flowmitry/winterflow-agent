# Build configuration
BINARY_NAME=winterflow-agent
VERSION=$(shell date +'%Y.%m.%d')
BUILD_DIR=build

# Go build flags
LDFLAGS=-X winterflow-agent/internal/agent.version=${VERSION}
BUILD_FLAGS=-v -ldflags="${LDFLAGS}"

.PHONY: all clean grpc build run install-tools

all: grpc build

# Build for local development
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build ${BUILD_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/main.go

# Run the agent
run: build
	@echo "Starting ${BINARY_NAME}..."
	@${BUILD_DIR}/${BINARY_NAME}

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -rf ${BUILD_DIR}
	@rm -f internal/grpc/pb/*.pb.go

grpc:
	@echo "Generating gRPC code..."
	@PATH="$$PATH:$$(go env GOPATH)/bin" protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/winterflow/grpc/pb/server.proto

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0