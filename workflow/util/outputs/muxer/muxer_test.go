package muxer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestMux(t *testing.T) {
	t.Run("NoOutputs", func(t *testing.T) {
		mux, err := Mux("foo", nil)
		assert.NoError(t, err)
		assert.Equal(t, "foo", mux)
		demux, outputs, err := Demux(mux)
		assert.NoError(t, err)
		assert.Equal(t, "foo", demux)
		assert.Nil(t, outputs)
	})
	t.Run("Outputs", func(t *testing.T) {
		mux, err := Mux("foo", &wfv1.Outputs{})
		assert.NoError(t, err)
		assert.Equal(t, "foo|{}", mux)
		demux, outputs, err := Demux(mux)
		assert.NoError(t, err)
		assert.Equal(t, "foo", demux)
		assert.NotNil(t, outputs)
	})
}
