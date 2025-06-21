# Build configuration
BINARY_NAME=agent
VERSION=$(shell date +'%Y.%m.%d')
GRPC_ADDR=127.0.0.1:50051
API_URL=http://127.0.0.1:8080
BUILD_DIR=.

# Go build flags
LDFLAGS=-X winterflow-agent/internal/application/version.version=${VERSION} -X winterflow-agent/internal/application/config.grpcServerAddress=${GRPC_ADDR} -X winterflow-agent/internal/application/config.apiBaseURL=${API_URL}
BUILD_FLAGS=-v

.PHONY: all clean grpc build run install-tools ansible-version

all: grpc build

# Create version file
ansible-version:
	@echo "Creating version file..."
	@mkdir -p ansible
	@echo "${VERSION}" > ansible/version.txt

# Build for local development
build: ansible-version
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@echo "go build ${BUILD_FLAGS} -ldflags=\"${LDFLAGS}\" -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/agent/main.go"
	@go build ${BUILD_FLAGS} -ldflags="${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/agent/main.go
	@chmod +x ${BUILD_DIR}/${BINARY_NAME}

# Run the agent
run: build
	@echo "Starting ${BINARY_NAME}..."
	@${BUILD_DIR}/${BINARY_NAME}

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -f ${BUILD_DIR}/${BINARY_NAME}
	@rm -f internal/infra/winterflow/grpc/pb/*.pb.go
	@rm -f ansible/version.txt

grpc:
	@echo "Generating gRPC code..."
	@PATH="$$PATH:$$(go env GOPATH)/bin" protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/infra/winterflow/grpc/pb/server.proto

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

generate-certs:
	openssl genrsa -out agent.key 2048
	openssl req -new -key agent.key -out agent.csr
	# openssl req -x509 -new -nodes -key .certs/agent.key -sha256 -days 36500 -out .certs/agent.crt
