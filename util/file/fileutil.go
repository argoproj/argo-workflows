package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"k8s.io/utils/env"

	"github.com/argoproj/argo-workflows/v4/util/logging"
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

	CompressionAlgorithmEnvVarKey = "WORKFLOW_COMPRESSION_ALGORITHM"
	GZipAlgorithm                 = "gzip"
	ZStdAlgorithm                 = "zstd"
	BrotliAlgorithm               = "brotli"

	// CompressionLevelEnvVarKey selects the compression level. Its range and
	// default are algorithm-specific: gzip 1-9 (default 6), zstd 1-4
	// (default 2), brotli 0-11 (default 6). Unset means each library's default.
	CompressionLevelEnvVarKey = "WORKFLOW_COMPRESSION_LEVEL"
)

var (
	// zstd.Encoder/Decoder are safe for concurrent EncodeAll/DecodeAll use.
	zstdDecoder, _ = zstd.NewReader(nil)

	zstdEncoders   = map[zstd.EncoderLevel]*zstd.Encoder{}
	zstdEncodersMu sync.Mutex

	zstdMagic = []byte{0x28, 0xb5, 0x2f, 0xfd}
	gzipMagic = []byte{0x1f, 0x8b}
)

// compressionLevel returns the level from CompressionLevelEnvVarKey if it is
// set and within [minLevel, maxLevel], and the algorithm's default otherwise.
func compressionLevel(ctx context.Context, defaultLevel, minLevel, maxLevel int) int {
	s := os.Getenv(CompressionLevelEnvVarKey)
	if s == "" {
		return defaultLevel
	}
	l, err := strconv.Atoi(s)
	if err != nil || l < minLevel || l > maxLevel {
		logging.RequireLoggerFromContext(ctx).WithField("level", s).
			Warn(ctx, "Invalid compression level, using the algorithm's default")
		return defaultLevel
	}
	return l
}

func zstdEncoderForLevel(level zstd.EncoderLevel) *zstd.Encoder {
	zstdEncodersMu.Lock()
	defer zstdEncodersMu.Unlock()
	enc, ok := zstdEncoders[level]
	if !ok {
		enc, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
		zstdEncoders[level] = enc
	}
	return enc
}

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
		if errors.Is(err, io.EOF) {
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

// CompressContent will compress the byte array using the algorithm selected
// by `CompressionAlgorithmEnvVarKey` (gzip by default). DecompressContent
// detects the algorithm from the content, so the variable only affects writes.
func CompressContent(ctx context.Context, content []byte) []byte {
	switch env.GetString(CompressionAlgorithmEnvVarKey, GZipAlgorithm) {
	case ZStdAlgorithm:
		level := zstd.EncoderLevel(compressionLevel(ctx, int(zstd.SpeedDefault), int(zstd.SpeedFastest), int(zstd.SpeedBestCompression)))
		return zstdEncoderForLevel(level).EncodeAll(content, nil)
	case BrotliAlgorithm:
		var buf bytes.Buffer
		brotliWriter := brotli.NewWriterLevel(&buf, compressionLevel(ctx, brotli.DefaultCompression, brotli.BestSpeed, brotli.BestCompression))
		_, err := brotliWriter.Write(content)
		if err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Error in compressing")
		}
		closeFile(ctx, brotliWriter)
		return buf.Bytes()
	}
	level := compressionLevel(ctx, gzip.DefaultCompression, gzip.BestSpeed, gzip.BestCompression)
	var buf bytes.Buffer
	var gzipWriter io.WriteCloser
	switch gzipImpl {
	case GZIP:
		gzipWriter, _ = gzip.NewWriterLevel(&buf, level)
	default:
		gzipWriter, _ = pgzip.NewWriterLevel(&buf, level)
	}

	_, err := gzipWriter.Write(content)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "Error in compressing")
	}
	closeFile(ctx, gzipWriter)
	return buf.Bytes()
}

// DecompressContent will return the uncompressed content, detecting the
// compression algorithm from the content. zstd and gzip are identified by
// their magic bytes; brotli streams have none, so anything else is treated
// as brotli (the only other algorithm we write).
func DecompressContent(ctx context.Context, content []byte) ([]byte, error) {
	if bytes.HasPrefix(content, zstdMagic) {
		return zstdDecoder.DecodeAll(content, nil)
	}
	if !bytes.HasPrefix(content, gzipMagic) {
		decompressed, err := io.ReadAll(brotli.NewReader(bytes.NewReader(content)))
		if err != nil {
			return nil, fmt.Errorf("failed to decompress: %w", err)
		}
		return decompressed, nil
	}
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
			f, openErr := os.Open(filepath.Clean(path))
			if openErr != nil {
				return openErr
			}
			defer func() {
				if closeErr := f.Close(); closeErr != nil {
					logging.RequireLoggerFromContext(ctx).WithError(closeErr).WithField("path", path).WithFatal().Error(ctx, "Error closing file")
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
