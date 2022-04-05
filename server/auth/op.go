package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
)

var table = map[string]string{
	"/service.TestService/GetTests": "get tests",
}

func getOperationID(ctx context.Context) (string, error) {
	s := grpc.ServerTransportStreamFromContext(ctx)
	if s == nil {
		return "", fmt.Errorf("unable to get transport stream from context")
	}
	m := s.Method()
	op, ok := table[m]
	if !ok {
		return "", fmt.Errorf("failed to find operation ID: unknown method %q", m)
	}
	return op, nil
}

func splitOp(method string) (string, string) {
	parts := strings.Split(method, " ")
	if len(parts) != 2 {
		panic(fmt.Errorf("expected 2 parts in %q", method))
	}
	return parts[0], parts[1]
}
