package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestObjectMeta(t *testing.T) {
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = InjectObjectMeta(ctx, &meta.ObjectMeta{Name: "foo", Namespace: "bar"})
	assert.Equal(t, "foo", ObjectName(ctx))
	assert.Equal(t, "bar", ObjectNamespace(ctx))
}
