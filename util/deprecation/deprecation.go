// Package deprecation records uses of deprecated features so that users can be made aware of
// things that may be removed in a future version and move away from them.
package deprecation

// This is a deliberate singleton devised to be functional when initialised with an
// instance of metrics, and otherwise to remain quiet
//
// This avoids the problem of injecting the metrics package (or whatever recording method the deprecation
// recorder is using) temporarily into packages and then painfully removing the injection later when the
// package no longer has deprecated features (as they've been removed)

import (
	"context"

	wfctx "github.com/argoproj/argo-workflows/v3/util/context"
)

type metricsFunc func(context.Context, string, string)

var (
	metricsF metricsFunc
)

type Type int

const (
	Undefined Type = iota
)

func (t *Type) asString() string {
	switch *t {
	case Undefined:
		return `undefined`
	default:
		return `unknown`
	}
}

func Initialize(m metricsFunc) {
	metricsF = m
}

func Record(ctx context.Context, deprecation Type) {
	if metricsF != nil {
		metricsF(ctx, deprecation.asString(), wfctx.ObjectNamespace(ctx))
	}
}
