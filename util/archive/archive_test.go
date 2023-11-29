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

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
			assert.NoError(t, err)

			log.Infof("Taring to %s", f.Name())

			err = TarGzToWriter(tt.src, tt.level, bufio.NewWriter(f))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			err = f.Close()
			assert.NoError(t, err)

			err = os.Remove(f.Name())
			assert.NoError(t, err)
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
			assert.NoError(t, err)
			_, err = data.WriteString("hello world")
			assert.NoError(t, err)
			err = data.Close()
			assert.NoError(t, err)

			dataTarPath := data.Name() + ".tgz"
			f, err := os.Create(dataTarPath)
			assert.NoError(t, err)

			log.Infof("Taring to %s", f.Name())

			err = TarGzToWriter(data.Name(), tt.level, bufio.NewWriter(f))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			err = os.Remove(data.Name())
			assert.NoError(t, err)
			err = f.Close()
			assert.NoError(t, err)
			err = os.Remove(f.Name())
			assert.NoError(t, err)
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
			assert.NoError(t, err)

			log.Infof("Zipping to %s", f.Name())

			err = ZipToWriter(tt.src, zip.NewWriter(f))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			err = f.Close()
			assert.NoError(t, err)

			err = os.Remove(f.Name())
			assert.NoError(t, err)
		})
	}
}

func TestZipFile(t *testing.T) {
	t.Run("test_zip_file", func(t *testing.T) {
		data, err := tempFile(os.TempDir()+"/argo-test", "file-random-", "")
		assert.NoError(t, err)
		_, err = data.WriteString("hello world")
		assert.NoError(t, err)
		err = data.Close()
		assert.NoError(t, err)

		dataZipPath := data.Name() + ".zip"
		f, err := os.Create(dataZipPath)
		assert.NoError(t, err)

		err = ZipToWriter(data.Name(), zip.NewWriter(f))
		assert.NoError(t, err)

		err = os.Remove(data.Name())
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)
		err = os.Remove(f.Name())
		assert.NoError(t, err)
	})
}
