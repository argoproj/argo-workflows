package packr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

var testBox = NewBox("./fixtures")
var virtualBox = NewBox("./virtual")

func init() {
	PackBytes(virtualBox.Path, "a", []byte("a"))
	PackBytes(virtualBox.Path, "b", []byte("b"))
	PackBytes(virtualBox.Path, "c", []byte("c"))
	PackBytes(virtualBox.Path, "d/a", []byte("d/a"))
}

func Test_PackBytes(t *testing.T) {
	r := require.New(t)
	PackBytes(testBox.Path, "foo", []byte("bar"))
	s := testBox.String("foo")
	r.Equal("bar", s)
}

func Test_PackJSONBytes(t *testing.T) {
	r := require.New(t)
	b, err := json.Marshal([]byte("json bytes"))
	r.NoError(err)
	err = PackJSONBytes(testBox.Path, "the bytes", string(b))
	r.NoError(err)
	s, err := testBox.MustBytes("the bytes")
	r.NoError(err)
	r.Equal([]byte("json bytes"), s)
}

func Test_PackBytesGzip(t *testing.T) {
	r := require.New(t)
	err := PackBytesGzip(testBox.Path, "gzip", []byte("gzip foobar"))
	r.NoError(err)
	s := testBox.String("gzip")
	r.Equal("gzip foobar", s)
}
