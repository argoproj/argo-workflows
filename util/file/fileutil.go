package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
	"k8s.io/utils/env"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

var (
	gzipImpl    = env.GetString(GZipImplEnvVarKey, PGZIP)
	manifestExt = map[string]bool{
		".yaml": true,
		".yml":  true,
		".json": true,
	}
)

const (
	GZipImplEnvVarKey = "GZIP_IMPLEMENTATION"
	GZIP              = "GZip"
	PGZIP             = "PGZip"
)

type TarReader interface {
	Next() (*tar.Header, error)
}

// GetGzipReader gets the GzipReader based on `GZipImplEnvVarKey` environment variable.
func GetGzipReader(reader io.Reader) (io.ReadCloser, error) {
	var err error
	var gzipReader io.ReadCloser
	switch gzipImpl {
	case GZIP:
		gzipReader, err = gzip.NewReader(reader)
	default:
		gzipReader, err = pgzip.NewReader(reader)
	}
	if err != nil {
		return nil, err
	}
	return gzipReader, nil
}

// ExistsInTar return true if file or directory exists in tar
func ExistsInTar(sourcePath string, tarReader TarReader) bool {
	sourcePath = strings.Trim(sourcePath, "/")
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		if hdr.FileInfo().IsDir() && strings.Contains(sourcePath, strings.Trim(hdr.Name, "/")) {
			return true
		}
		if strings.Contains(sourcePath, hdr.Name) {
			return true
		}
	}
	return false
}

// Close the file
func closeFile(ctx context.Context, f io.Closer) {
	err := f.Close()
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Failed to close the file/writer/reader")
	}
}

// CompressEncodeString will return the compressed string with base64 encoded
func CompressEncodeString(ctx context.Context, content string) string {
	return base64.StdEncoding.EncodeToString(CompressContent(ctx, []byte(content)))
}

// DecodeDecompressString will return  decode and decompress the
func DecodeDecompressString(ctx context.Context, content string) (string, error) {
	buf, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", err
	}
	dBuf, err := DecompressContent(ctx, buf)
	if err != nil {
		return "", err
	}
	return string(dBuf), nil
}

// CompressContent will compress the byte array using zip writer
func CompressContent(ctx context.Context, content []byte) []byte {
	var buf bytes.Buffer
	var gzipWriter io.WriteCloser
	switch gzipImpl {
	case GZIP:
		gzipWriter = gzip.NewWriter(&buf)
	default:
		gzipWriter = pgzip.NewWriter(&buf)
	}

	_, err := gzipWriter.Write(content)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Error in compressing")
	}
	closeFile(ctx, gzipWriter)
	return buf.Bytes()
}

// DecompressContent will return the uncompressed content
func DecompressContent(ctx context.Context, content []byte) ([]byte, error) {
	buf := bytes.NewReader(content)
	gzipReader, err := GetGzipReader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	defer closeFile(ctx, gzipReader)
	return io.ReadAll(gzipReader)
}

// WalkManifests is based on filepath.Walk but will only walk through Kubernetes manifests
func WalkManifests(ctx context.Context, root string, fn func(path string, data []byte) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		var r io.Reader
		switch {
		case path == "-":
			path = "stdin"
			r = os.Stdin
		case err != nil:
			return err
		case strings.HasPrefix(path, "/dev/") || manifestExt[filepath.Ext(path)]:
			f, err := os.Open(filepath.Clean(path))
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					logging.RequireLoggerFromContext(ctx).WithError(err).WithField("path", path).WithFatal().Error(ctx, "Error closing file")
				}
			}()
			r = f
		case info.IsDir():
			return nil // skip
		default:
			logging.RequireLoggerFromContext(ctx).WithField("path", path).Debug(ctx, "ignoring file with unknown extension")
			return nil
		}

		bytes, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		return fn(path, bytes)
	})
}

// IsDirectory returns whether or not the given file is a directory
func IsDirectory(path string) (bool, error) {
	fileOrDir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = fileOrDir.Close() }()
	stat, err := fileOrDir.Stat()
	if err != nil {
		return false, err
	}
	return stat.IsDir(), nil
}

// Exists returns whether or not a path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
