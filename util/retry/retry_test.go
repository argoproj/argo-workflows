package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsResourceQuotaConflictErr(t *testing.T) {
	assert.False(t, IsResourceQuotaConflictErr(apierr.NewConflict(schema.GroupResource{}, "", nil)))
	assert.True(t, IsResourceQuotaConflictErr(apierr.NewConflict(schema.GroupResource{Group: "v1", Resource: "resourcequotas"}, "", nil)))
}
