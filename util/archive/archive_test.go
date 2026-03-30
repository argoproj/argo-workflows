package archive

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func tempFile(dir, prefix, suffix string) (*os.File, error) {
	if dir == "" {
		dir = os.TempDir()
	} else {
		err := os.MkdirAll(dir, 0o700)
		if err != nil {
			return nil, err
		}
	}
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(dir, prefix+hex.EncodeToString(randBytes)+suffix)
	return os.Create(filePath)
}

func TestTarDirectory(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		level   int
		wantErr bool
	}{
		{
			"dir_missing",
			"./fake/dir",
			gzip.NoCompression,
			true,
		},
		{
			"level_default",
			"../../test/e2e",
			gzip.DefaultCompression,
			false,
		},
		{
			"level_none",
			"../../test/e2e",
			gzip.NoCompression,
			false,
		},
		{
			"level_out_of_range",
			"../../test/e2e",
			-5,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tempFile(os.TempDir()+"/argo-test", "dir-"+tt.name+"-", ".tgz")
			require.NoError(t, err)

			ctx := logging.TestContext(t.Context())

			err = TarGzToWriter(ctx, tt.src, tt.level, bufio.NewWriter(f))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			err = f.Close()
			require.NoError(t, err)

			err = os.Remove(f.Name())
			require.NoError(t, err)
		})
	}
}

func TestTarFile(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		wantErr bool
	}{
		{
			"level_default",
			gzip.DefaultCompression,
			false,
		},
		{
			"level_none",
			gzip.NoCompression,
			false,
		},
		{
			"level_out_of_range",
			-5,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tempFile(os.TempDir()+"/argo-test", "file-"+tt.name+"-", "")
			require.NoError(t, err)
			_, err = data.WriteString("hello world")
			require.NoError(t, err)
			err = data.Close()
			require.NoError(t, err)

			dataTarPath := data.Name() + ".tgz"
			f, err := os.Create(dataTarPath)
			require.NoError(t, err)

			ctx := logging.TestContext(t.Context())

			err = TarGzToWriter(ctx, data.Name(), tt.level, bufio.NewWriter(f))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			err = os.Remove(data.Name())
			require.NoError(t, err)
			err = f.Close()
			require.NoError(t, err)
			err = os.Remove(f.Name())
			require.NoError(t, err)
		})
	}
}

func TestZipDirectory(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			"dir_missing",
			"./fake/dir",
			true,
		},
		{
			"dir_common",
			"../../test/e2e",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tempFile(os.TempDir()+"/argo-test", "dir-"+tt.name+"-", ".tgz")
			require.NoError(t, err)

			ctx := logging.TestContext(t.Context())

			err = ZipToWriter(ctx, tt.src, zip.NewWriter(f))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			err = f.Close()
			require.NoError(t, err)

			err = os.Remove(f.Name())
			require.NoError(t, err)
		})
	}
}

func TestZipFile(t *testing.T) {
	t.Run("test_zip_file", func(t *testing.T) {
		data, err := tempFile(os.TempDir()+"/argo-test", "file-random-", "")
		require.NoError(t, err)
		_, err = data.WriteString("hello world")
		require.NoError(t, err)
		err = data.Close()
		require.NoError(t, err)

		dataZipPath := data.Name() + ".zip"
		f, err := os.Create(dataZipPath)
		require.NoError(t, err)

		ctx := logging.TestContext(t.Context())

		err = ZipToWriter(ctx, data.Name(), zip.NewWriter(f))
		require.NoError(t, err)

		err = os.Remove(data.Name())
		require.NoError(t, err)
		err = f.Close()
		require.NoError(t, err)
		err = os.Remove(f.Name())
		require.NoError(t, err)
	})
}
