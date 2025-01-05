package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObjectMeta(t *testing.T) {
	ctx := context.Background()
	ctx = InjectObjectMeta(ctx, &meta.ObjectMeta{Name: "foo", Namespace: "bar"})
	assert.Equal(t, "foo", ObjectName(ctx))
	assert.Equal(t, "bar", ObjectNamespace(ctx))
}
