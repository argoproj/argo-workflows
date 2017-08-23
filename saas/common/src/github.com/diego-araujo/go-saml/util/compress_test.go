package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompressString(t *testing.T) {
	expected := "This is the test string"
	compressed := CompressString(expected)
	decompressed := DecompressString(compressed)
	assert.Equal(t, expected, decompressed)
	assert.True(t, len(compressed) > len(decompressed))
}

func TestCompress(t *testing.T) {
	expected := []byte("This is the test string")
	compressed := Compress(expected)
	decompressed := Decompress(compressed)
	assert.Equal(t, expected, decompressed)
	assert.True(t, len(compressed) > len(decompressed))
}
