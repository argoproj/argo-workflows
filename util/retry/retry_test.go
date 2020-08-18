package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsResourceQuotaConflictErr(t *testing.T) {
	assert.False(t, isResourceQuotaConflictErr(apierr.NewConflict(schema.GroupResource{}, "", nil)))
	assert.True(t, isResourceQuotaConflictErr(apierr.NewConflict(schema.GroupResource{Group: "v1", Resource: "resourcequotas"}, "", nil)))
}

func Test_isExceededQuotaErr(t *testing.T) {
	assert.False(t, isExceededQuotaErr(apierr.NewForbidden(schema.GroupResource{}, "", nil)))
	assert.True(t, isExceededQuotaErr(apierr.NewForbidden(schema.GroupResource{Group: "v1", Resource: "pods"}, "", errors.New("exceeded quota"))))
}