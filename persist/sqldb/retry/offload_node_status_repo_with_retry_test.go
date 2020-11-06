package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/argoproj/argo/persist/sqldb"
	sqldbmocks "github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var transientErr = apierr.NewTooManyRequests("", 0)
var permanentErr = errors.New("")

func Test_offloadNodeStatusRepoWithRetry(t *testing.T) {
	t.Run("PermanentError", func(t *testing.T) {
		delegate := &sqldbmocks.OffloadNodeStatusRepo{}
		o := WithRetry(delegate)
		delegate.On("Save", mock.Anything, mock.Anything, mock.Anything).
			Return("", transientErr).
			Return("", permanentErr)
		_, err := o.Save("my-uid", "my-ns", wfv1.Nodes{})
		assert.Equal(t, permanentErr, err)
	})
	delegate := &sqldbmocks.OffloadNodeStatusRepo{}
	o := WithRetry(delegate)
	t.Run("Save", func(t *testing.T) {
		delegate.On("Save", "my-uid", "my-ns", mock.Anything).
			Return("", transientErr).
			Return("my-version", nil)
		version, err := o.Save("my-uid", "my-ns", wfv1.Nodes{})
		if assert.NoError(t, err) {
			assert.Equal(t, "my-version", version)
		}
	})
	t.Run("Get", func(t *testing.T) {
		delegate.On("Get", "my-uid", "my-version").
			Return(nil, transientErr).
			Return(wfv1.Nodes{}, nil)
		nodes, err := o.Get("my-uid", "my-version")
		if assert.NoError(t, err) {
			assert.NotNil(t, nodes)
		}
	})
	t.Run("List", func(t *testing.T) {
		delegate.On("List", "my-ns").
			Return(nil, transientErr).
			Return(make(map[sqldb.UUIDVersion]wfv1.Nodes), nil)
		list, err := o.List("my-ns")
		if assert.NoError(t, err) {
			assert.NotNil(t, list)
		}
	})
	t.Run("ListOldOffloads", func(t *testing.T) {
		delegate.On("ListOldOffloads", "my-ns").
			Return(nil, transientErr).
			Return(make([]sqldb.UUIDVersion, 0), nil)
		list, err := o.ListOldOffloads("my-ns")
		if assert.NoError(t, err) {
			assert.NotNil(t, list)
		}
	})
	t.Run("Delete", func(t *testing.T) {
		delegate.On("Delete", "my-uid", "my-version").
			Return(transientErr).
			Return(nil)
		err := o.Delete("my-uid", "my-version")
		assert.NoError(t, err)
	})
	t.Run("IsEnabled", func(t *testing.T) {
		delegate.On("IsEnabled").
			Return(true)
		assert.True(t, o.IsEnabled())
	})
}

func Test_done(t *testing.T) {
	assert.True(t, done(nil))
	assert.False(t, done(transientErr))
	assert.True(t, done(permanentErr))
}
