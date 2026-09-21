// Package wfcontext contains common functions for storing and retrieving information from
// standard go context
package wfcontext

import (
	"context"
	"time"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type objectName struct{}
type objectNamespace struct{}
type creationTimestamp struct{}

// InjectObjectMeta stores various parts of the object metadata in context
func InjectObjectMeta(ctx context.Context, meta *meta.ObjectMeta) context.Context {
	retctx := context.WithValue(ctx, creationTimestamp{}, meta.CreationTimestamp.Time)
	retctx = context.WithValue(retctx, objectNamespace{}, meta.Namespace)
	return context.WithValue(retctx, objectName{}, meta.Name)
}

// ObjectName fetches current object name from the context
func ObjectName(ctx context.Context) string {
	if n, ok := ctx.Value(objectName{}).(string); ok {
		return n
	}
	return ""
}

// ObjectNamespace fetches current object namespace from the context
func ObjectNamespace(ctx context.Context) string {
	if n, ok := ctx.Value(objectNamespace{}).(string); ok {
		return n
	}
	return ""
}

// ObjectCreationTimestamp fetches current object's creation time from the context
func ObjectCreationTimestamp(ctx context.Context) time.Time {
	if n, ok := ctx.Value(creationTimestamp{}).(time.Time); ok {
		return n
	}
	return time.Time{}
}

// UIDList fetches elements of the object metadata from the context as a string list
// for creating UIDs unique to the workflow
func UIDList(ctx context.Context) []string {
	return []string{
		ObjectName(ctx),
		ObjectNamespace(ctx),
		ObjectCreationTimestamp(ctx).String(),
	}
}
