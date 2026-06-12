package file_test

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/rand"
)

// TestCompressContentString ensures compressing then decompressing a content string works as expected
func TestCompressContentString(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, gzipImpl := range []string{file.GZIP, file.PGZIP} {
		t.Setenv(file.GZipImplEnvVarKey, gzipImpl)
		content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"name\":\"pod-limits-rrdm8[0]." +
			"run-pod(0:0)\",\"displayName\":\"run-pod(0:0)\",\"type\":\"Pod\",\"templateName\":\"run-pod\",\"phase\":" +
			"\"Succeeded\",\"boundaryID\":\"pod-limits-rrdm8\",\"startedAt\":\"2019-03-07T19:14:50Z\",\"finishedAt\":" +
			"\"2019-03-07T19:14:55Z\"}}"

		compString := file.CompressEncodeString(ctx, content)

		resultString, _ := file.DecodeDecompressString(ctx, compString)

		assert.Equal(t, content, resultString)
	}
}

// TestZstdCompression ensures the writer switches to zstd via the environment
// variable while the reader detects the algorithm from the content alone.
func TestZstdCompression(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"phase\":\"Succeeded\"}}"

	var compressed string
	t.Run("CompressWithZstd", func(t *testing.T) {
		t.Setenv(file.CompressionAlgorithmEnvVarKey, file.ZStdAlgorithm)
		compressed = file.CompressEncodeString(ctx, content)
		buf, err := base64.StdEncoding.DecodeString(compressed)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x28, 0xb5, 0x2f, 0xfd}, buf[:4], "expected zstd magic bytes")
	})

	t.Run("DecompressDetectsZstdWithoutEnvVar", func(t *testing.T) {
		resultString, err := file.DecodeDecompressString(ctx, compressed)
		require.NoError(t, err)
		assert.Equal(t, content, resultString)
	})

	t.Run("UnknownAlgorithmFallsBackToGzip", func(t *testing.T) {
		t.Setenv(file.CompressionAlgorithmEnvVarKey, "lz4")
		comp := file.CompressEncodeString(ctx, content)
		buf, err := base64.StdEncoding.DecodeString(comp)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x1f, 0x8b}, buf[:2], "expected gzip magic bytes")
		resultString, err := file.DecodeDecompressString(ctx, comp)
		require.NoError(t, err)
		assert.Equal(t, content, resultString)
	})
}

// TestBrotliCompression ensures brotli writes round-trip and are detected by
// the reader by elimination (no gzip/zstd magic bytes), with the quality knob
// falling back to the default when invalid.
func TestBrotliCompression(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"phase\":\"Succeeded\"}}"

	var compressed string
	t.Run("CompressWithBrotli", func(t *testing.T) {
		t.Setenv(file.CompressionAlgorithmEnvVarKey, file.BrotliAlgorithm)
		compressed = file.CompressEncodeString(ctx, content)
		buf, err := base64.StdEncoding.DecodeString(compressed)
		require.NoError(t, err)
		assert.NotEqual(t, []byte{0x1f, 0x8b}, buf[:2], "must not look like gzip")
		assert.NotEqual(t, []byte{0x28, 0xb5, 0x2f, 0xfd}, buf[:4], "must not look like zstd")
	})

	t.Run("DecompressDetectsBrotliWithoutEnvVar", func(t *testing.T) {
		resultString, err := file.DecodeDecompressString(ctx, compressed)
		require.NoError(t, err)
		assert.Equal(t, content, resultString)
	})

	t.Run("InvalidLevelFallsBackToDefault", func(t *testing.T) {
		t.Setenv(file.CompressionAlgorithmEnvVarKey, file.BrotliAlgorithm)
		t.Setenv(file.CompressionLevelEnvVarKey, "banana")
		comp := file.CompressEncodeString(ctx, content)
		resultString, err := file.DecodeDecompressString(ctx, comp)
		require.NoError(t, err)
		assert.Equal(t, content, resultString)
	})

	t.Run("ExplicitLevel", func(t *testing.T) {
		t.Setenv(file.CompressionAlgorithmEnvVarKey, file.BrotliAlgorithm)
		t.Setenv(file.CompressionLevelEnvVarKey, "11")
		comp := file.CompressEncodeString(ctx, content)
		resultString, err := file.DecodeDecompressString(ctx, comp)
		require.NoError(t, err)
		assert.Equal(t, content, resultString)
	})
}

// TestCompressionLevel ensures WORKFLOW_COMPRESSION_LEVEL applies to each
// algorithm with algorithm-specific ranges.
func TestCompressionLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"phase\":\"Succeeded\"}}"

	for algo, level := range map[string]string{
		file.GZipAlgorithm:   "9",
		file.ZStdAlgorithm:   "4",
		file.BrotliAlgorithm: "5",
	} {
		t.Run(algo, func(t *testing.T) {
			t.Setenv(file.CompressionAlgorithmEnvVarKey, algo)
			t.Setenv(file.CompressionLevelEnvVarKey, level)
			comp := file.CompressEncodeString(ctx, content)
			resultString, err := file.DecodeDecompressString(ctx, comp)
			require.NoError(t, err)
			assert.Equal(t, content, resultString)
		})
	}
}

// TestGetGzipReader checks whether we can obtain the Gzip reader based on environment variable.
func TestGetGzipReader(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, gzipImpl := range []string{file.GZIP, file.PGZIP} {
		t.Setenv(file.GZipImplEnvVarKey, gzipImpl)
		rawContent := "this is the content"
		content := file.CompressEncodeString(ctx, rawContent)
		buf, err := base64.StdEncoding.DecodeString(content)
		require.NoError(t, err)
		bufReader := bytes.NewReader(buf)
		reader, err := file.GetGzipReader(bufReader)
		require.NoError(t, err)
		res, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, rawContent, string(res))
	}
}

func TestExistsInTar(t *testing.T) {
	type fakeFile struct {
		name, body string
		isDir      bool
	}

	newTarReader := func(t *testing.T, files []fakeFile) *tar.Reader {
		var buf bytes.Buffer
		writer := tar.NewWriter(&buf)
		for _, f := range files {
			mode := os.FileMode(0o600)
			if f.isDir {
				mode |= os.ModeDir
			}
			hdr := tar.Header{Name: f.name, Mode: int64(mode), Size: int64(len(f.body))}
			err := writer.WriteHeader(&hdr)
			require.NoError(t, err)
			_, err = writer.Write([]byte(f.body))
			require.NoError(t, err)
		}
		err := writer.Close()
		require.NoError(t, err)

		return tar.NewReader(&buf)
	}

	type TestCase struct {
		sourcePath string
		expected   bool
		files      []fakeFile
	}

	tests := []TestCase{
		{
			sourcePath: "/root.txt", expected: true,
			files: []fakeFile{{name: "root.txt", body: "file in the root"}},
		},
		{
			sourcePath: "/tmp/file/in/subfolder.txt", expected: true,
			files: []fakeFile{{name: "subfolder.txt", body: "a file in a subfolder"}},
		},
		{
			sourcePath: "/root", expected: true,
			files: []fakeFile{
				{name: "root/", isDir: true},
				{name: "root/a.txt", body: "a"},
				{name: "root/b.txt", body: "b"},
			},
		},
		{
			sourcePath: "/tmp/subfolder", expected: true,
			files: []fakeFile{
				{name: "subfolder/", isDir: true},
				{name: "subfolder/a.txt", body: "a"},
				{name: "subfolder/b.txt", body: "b"},
			},
		},
		{
			// should an empty tar return true??
			sourcePath: "/tmp/empty", expected: true,
			files: []fakeFile{
				{name: "empty/", isDir: true},
			},
		},
		{
			sourcePath: "/tmp/folder/that", expected: false,
			files: []fakeFile{
				{name: "this/", isDir: true},
				{name: "this/a.txt", body: "a"},
				{name: "this/b.txt", body: "b"},
			},
		},
		{
			sourcePath: "/empty.txt", expected: true,
			files: []fakeFile{
				{name: "empty.txt", body: ""},
			},
		},
		{
			sourcePath: "/tmp/empty.txt", expected: true,
			files: []fakeFile{
				{name: "empty.txt", body: ""},
			},
		},
	}
	for _, tc := range tests {
		t.Run("source path "+tc.sourcePath, func(t *testing.T) {
			t.Parallel()
			tarReader := newTarReader(t, tc.files)
			actual := file.ExistsInTar(tc.sourcePath, tarReader)
			assert.Equalf(t, tc.expected, actual, "sourcePath %s not found", tc.sourcePath)
		})
	}
}

// TestIsDirectory tests if a path is a directory. Errors if directory doesn't exist
func TestIsDirectory(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not determine test directory")
	}
	testDir := filepath.Dir(filename)

	isDir, err := file.IsDirectory(testDir)
	require.NoError(t, err)
	assert.True(t, isDir)

	isDir, err = file.IsDirectory(filename)
	require.NoError(t, err)
	assert.False(t, isDir)

	isDir, err = file.IsDirectory("doesnt-exist")
	require.Error(t, err)
	assert.False(t, isDir)
}

func TestExists(t *testing.T) {
	assert.True(t, file.Exists("/"))
	path, err := rand.String(10)
	require.NoError(t, err)
	randFilePath := fmt.Sprintf("/%s", path)
	assert.False(t, file.Exists(randFilePath))
}
