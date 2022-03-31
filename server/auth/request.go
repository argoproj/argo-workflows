package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/grpc"

	servertypes "github.com/argoproj/argo-workflows/v3/server/types"
)

func getMethod(ctx context.Context) (string, error) {
	s := grpc.ServerTransportStreamFromContext(ctx)
	if s == nil {
		return "", fmt.Errorf("unable to get transport stream from context")
	}
	m := s.Method()
	parts := strings.Split(m, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("method %q invalid", m)
	}
	return parts[2], nil
}

func parseMethod(method string) (string, string) {
	h := regexp.MustCompile(`[A-Z][a-z]*`).FindString(method)
	return strings.ToLower(h), strings.ToLower(strings.TrimPrefix(method, h))
}

func getNamespace(req interface{}) string {
	if req == nil {
		return ""
	}
	v, ok := req.(servertypes.NamespacedRequest)
	if !ok {
		return ""
	}
	return v.GetNamespace()
}
