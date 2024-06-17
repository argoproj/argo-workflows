package file_test

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/file"
)

// TestCompressContentString ensures compressing then decompressing a content string works as expected
func TestCompressContentString(t *testing.T) {
	for _, gzipImpl := range []string{file.GZIP, file.PGZIP} {
		t.Setenv(file.GZipImplEnvVarKey, gzipImpl)
		content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"name\":\"pod-limits-rrdm8[0]." +
			"run-pod(0:0)\",\"displayName\":\"run-pod(0:0)\",\"type\":\"Pod\",\"templateName\":\"run-pod\",\"phase\":" +
			"\"Succeeded\",\"boundaryID\":\"pod-limits-rrdm8\",\"startedAt\":\"2019-03-07T19:14:50Z\",\"finishedAt\":" +
			"\"2019-03-07T19:14:55Z\"}}"

		compString := file.CompressEncodeString(content)

		resultString, _ := file.DecodeDecompressString(compString)

		assert.Equal(t, content, resultString)
	}
}

// TestGetGzipReader checks whether we can obtain the Gzip reader based on environment variable.
func TestGetGzipReader(t *testing.T) {
	for _, gzipImpl := range []string{file.GZIP, file.PGZIP} {
		t.Setenv(file.GZipImplEnvVarKey, gzipImpl)
		rawContent := "this is the content"
		content := file.CompressEncodeString(rawContent)
		buf, err := base64.StdEncoding.DecodeString(content)
		assert.NoError(t, err)
		bufReader := bytes.NewReader(buf)
		reader, err := file.GetGzipReader(bufReader)
		assert.NoError(t, err)
		res, err := io.ReadAll(reader)
		assert.NoError(t, err)
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
			assert.NoError(t, err)
			_, err = writer.Write([]byte(f.body))
			assert.NoError(t, err)
		}
		err := writer.Close()
		assert.NoError(t, err)
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
		tc := tc
		t.Run("source path "+tc.sourcePath, func(t *testing.T) {
			t.Parallel()
			tarReader := newTarReader(t, tc.files)
			actual := file.ExistsInTar(tc.sourcePath, tarReader)
			assert.Equalf(t, tc.expected, actual, "sourcePath %s not found", tc.sourcePath)
		})
	}
}
