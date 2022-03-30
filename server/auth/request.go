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
	fullMethod := grpc.ServerTransportStreamFromContext(ctx).Method()
	parts := strings.Split(fullMethod, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("full method %q invalid", fullMethod)
	}
	return parts[2], nil
}

func parseMethod(method string) (string, string) {
	h := regexp.MustCompile(`[A-Z][a-z]*`).FindString(method)
	return strings.ToLower(h), strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(method, "V2"), h))
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
