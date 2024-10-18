// Package context contains common functions for storing and retrieving information from
// standard go context
package context

import (
	"context"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type objectIdentifier string

const (
	name      objectIdentifier = `object_name`
	namespace objectIdentifier = `object_namespace`
)

func InjectObjectMeta(ctx context.Context, meta *meta.ObjectMeta) context.Context {
	ctx = context.WithValue(ctx, name, meta.Name)
	return context.WithValue(ctx, namespace, meta.Namespace)
}

func ObjectName(ctx context.Context) string {
	if n, ok := ctx.Value(name).(string); ok {
		return n
	}
	return ""
}

func ObjectNamespace(ctx context.Context) string {
	if n, ok := ctx.Value(namespace).(string); ok {
		return n
	}
	return ""
}
