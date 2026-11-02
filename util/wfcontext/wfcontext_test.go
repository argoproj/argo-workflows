package wfcontext

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestObjectMeta(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx = InjectObjectMeta(ctx, &meta.ObjectMeta{
		Name:              "foo",
		Namespace:         "bar",
		CreationTimestamp: meta.NewTime(ts),
	})
	assert.Equal(t, "foo", ObjectName(ctx))
	assert.Equal(t, "bar", ObjectNamespace(ctx))
	assert.Equal(t, ts, ObjectCreationTimestamp(ctx))
}

func TestCreationTimestamp_Missing(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert.True(t, ObjectCreationTimestamp(ctx).IsZero())
}
