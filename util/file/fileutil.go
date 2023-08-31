package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/env"
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
func close(f io.Closer) {
	err := f.Close()
	if err != nil {
		log.Warnf("Failed to close the file/writer/reader. %v", err)
	}
}

// CompressEncodeString will return the compressed string with base64 encoded
func CompressEncodeString(content string) string {
	return base64.StdEncoding.EncodeToString(CompressContent([]byte(content)))
}

// DecodeDecompressString will return  decode and decompress the
func DecodeDecompressString(content string) (string, error) {
	buf, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", err
	}
	dBuf, err := DecompressContent(buf)
	if err != nil {
		return "", err
	}
	return string(dBuf), nil
}

// CompressContent will compress the byte array using zip writer
func CompressContent(content []byte) []byte {
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
		log.Warnf("Error in compressing: %v", err)
	}
	close(gzipWriter)
	return buf.Bytes()
}

// DecompressContent will return the uncompressed content
func DecompressContent(content []byte) ([]byte, error) {
	buf := bytes.NewReader(content)
	gzipReader, err := GetGzipReader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	defer close(gzipReader)
	return io.ReadAll(gzipReader)
}

// WalkManifests is based on filepath.Walk but will only walk through Kubernetes manifests
func WalkManifests(root string, fn func(path string, data []byte) error) error {
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
					log.Fatalf("Error closing file[%s]: %v", path, err)
				}
			}()
			r = f
		case info.IsDir():
			return nil // skip
		default:
			logrus.Debugf("ignoring file with unknown extension: %s", path)
			return nil
		}

		bytes, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		return fn(path, bytes)
	})
}
