package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Guards the hand-maintained Size()/MarshalToSizedBuffer pair in
// generated.pb.go: Size must account for every field Marshal emits,
// otherwise the backward writer under-runs the buffer (index out of range
// panic, observed as argo-server crash-looping on workflow responses).
func TestTemplateSpecReferenceProtoSizeMatchesMarshal(t *testing.T) {
	cases := []*TemplateSpecReference{
		{UID: "0d1d21ee-09e0-4012-bdcb-905308b9b679", Version: "sha256:ccc417849534c14afa55f42c9300ba70800c750b05cb0ab3a81805415bdd4227", Hydrated: true},
		{UID: "u", Version: "v"},
		{UID: "uid-only-longer-string", Hydrated: false},
		{},
	}
	for _, in := range cases {
		size := in.Size()
		buf := make([]byte, size)
		written, err := in.MarshalToSizedBuffer(buf)
		require.NoError(t, err)
		data := buf[size-written:]
		var back TemplateSpecReference
		require.NoError(t, back.Unmarshal(data))
		assert.Equal(t, *in, back)
	}
}
