// +build tools

// This package contains code generation utilities
// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "bou.ke/staticfiles"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/gogo/protobuf/gogoproto"
	_ "github.com/gogo/protobuf/protoc-gen-gogo"
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "github.com/mattn/goreman"
	_ "golang.org/x/tools/cmd/goimports"
)
