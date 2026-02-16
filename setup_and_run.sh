#!/bin/bash
set -e

# Create a persistent fake GOPATH
FAKE_GOPATH="$HOME/.gemini/fake_gopath"
mkdir -p "$FAKE_GOPATH"
echo "Using fake GOPATH at $FAKE_GOPATH"

# Define the project path within the fake GOPATH
PROJECT_PATH="$FAKE_GOPATH/src/github.com/argoproj/argo-workflows"
mkdir -p "$(dirname "$PROJECT_PATH")"

# Symlink the current directory to the project path if it doesn't exist or is broken
if [ ! -L "$PROJECT_PATH" ]; then
    ln -s "$(pwd)" "$PROJECT_PATH"
fi

# Set environment variables
export GOPATH="$FAKE_GOPATH"
export PATH="$GOPATH/bin:$PATH"
export GO111MODULE=on

# Install tools required by the Makefile (only if missing)
echo "Installing dependencies..."
if [ ! -f "$GOPATH/bin/go-to-protobuf" ]; then go install k8s.io/code-generator/cmd/go-to-protobuf@v0.33.1; fi
if [ ! -f "$GOPATH/bin/protoc-gen-gogo" ]; then go install github.com/gogo/protobuf/protoc-gen-gogo@v1.3.2; fi
if [ ! -f "$GOPATH/bin/protoc-gen-gogofast" ]; then go install github.com/gogo/protobuf/protoc-gen-gogofast@v1.3.2; fi
if [ ! -f "$GOPATH/bin/protoc-gen-grpc-gateway" ]; then go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0; fi
if [ ! -f "$GOPATH/bin/protoc-gen-swagger" ]; then go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.16.0; fi
if [ ! -f "$GOPATH/bin/openapi-gen" ]; then go install k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20220124234850-424119656bbf; fi
if [ ! -f "$GOPATH/bin/swagger" ]; then go install github.com/go-swagger/go-swagger/cmd/swagger@v0.33.1; fi
if [ ! -f "$GOPATH/bin/goimports" ]; then go install golang.org/x/tools/cmd/goimports@v0.35.0; fi
if [ ! -f "$GOPATH/bin/mockery" ]; then go install github.com/vektra/mockery/v3@v3.5.1; fi
if [ ! -f "$GOPATH/bin/controller-gen" ]; then go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0; fi

# Clone gogo/protobuf to GOPATH src for legacy imports (needed by protoc)
echo "Setting up gogo/protobuf source..."
mkdir -p "$GOPATH/src/github.com/gogo"
if [ ! -d "$GOPATH/src/github.com/gogo/protobuf" ]; then
    git clone --depth 1 https://github.com/gogo/protobuf.git -b v1.3.2 "$GOPATH/src/github.com/gogo/protobuf"
fi

# Ensure vendor directory exists
echo "Ensuring vendor directory exists..."
if [ ! -d "vendor" ]; then
    go mod vendor
fi

# Copy vendor directories to GOPATH src to resolve imports
echo "Copying vendor directories..."
rm -rf "$GOPATH/src/k8s.io" "$GOPATH/src/google.golang.org"
mkdir -p "$GOPATH/src/k8s.io"
# Use cp -L to follow symlinks if any
cp -rL "vendor/k8s.io/apimachinery" "$GOPATH/src/k8s.io/"
cp -rL "vendor/k8s.io/api" "$GOPATH/src/k8s.io/"
cp -rL "vendor/k8s.io/utils" "$GOPATH/src/k8s.io/" || true # utils might be needed
cp -rL "vendor/k8s.io/kube-openapi" "$GOPATH/src/k8s.io/" || true
mkdir -p "$GOPATH/src/google.golang.org"
cp -rL "vendor/google.golang.org/genproto" "$GOPATH/src/google.golang.org/"

# Make files writable
chmod -R u+w "$GOPATH/src"

echo "Verifying generated.proto existence..."
ls -l "$GOPATH/src/k8s.io/apimachinery/pkg/runtime/generated.proto" || echo "Still missing!"

# Run the codegen target using the original Makefile
echo "Running codegen..."
cd "$PROJECT_PATH"
# Ensure we are in the correct directory before running make
echo "Working directory: $(pwd)"
make -f Makefile codegen USE_NIX=false

echo "Done."
