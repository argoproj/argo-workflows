package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"io/ioutil"
	"strings"

	"github.com/klauspost/pgzip"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/env"
)

var gzipImpl = env.GetString(GZipImplEnvVarKey, GZIP)

const (
	GZipImplEnvVarKey = "GZIP_IMPLEMENTATION"
	GZIP              = "GZip"
	PGZIP             = "PGZip"
)

type TarReader interface {
	Next() (*tar.Header, error)
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

type GZipWriter interface {
	Write(p []byte) (int, error)
	Close() error
}

type GZipReader interface {
	Read(p []byte) (int, error)
	Close() error
}

// CompressContent will compress the byte array using zip writer
func CompressContent(content []byte) []byte {
	var buf bytes.Buffer
	var gzipWriter GZipWriter
	switch gzipImpl {
	case PGZIP:
		gzipWriter = pgzip.NewWriter(&buf)
	case GZIP:
		gzipWriter = gzip.NewWriter(&buf)
	default:
		log.Warnf("%s implementation for compression is not supported. Fallback to Gzip.", gzipImpl)
		gzipWriter = gzip.NewWriter(&buf)
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

	var gzipReader GZipReader
	switch gzipImpl {
	case PGZIP:
		gzipReader, _ = pgzip.NewReader(buf)
	case GZIP:
		gzipReader, _ = gzip.NewReader(buf)
	default:
		log.Warnf("%s implementation for decompression is not supported. Fallback to Gzip.", gzipImpl)
		gzipReader, _ = gzip.NewReader(buf)
	}
	defer close(gzipReader)
	return ioutil.ReadAll(gzipReader)
}
