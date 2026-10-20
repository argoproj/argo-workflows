package archive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

// zipEntries archives sourcePath and returns the entries it produced, keyed by
// the entry's base name.
func zipEntries(t *testing.T, sourcePath string) map[string]*zip.File {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	require.NoError(t, ZipToWriter(logging.TestContext(t.Context()), sourcePath, zw))
	require.NoError(t, zw.Close())

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	out := make(map[string]*zip.File, len(reader.File))
	for _, f := range reader.File {
		out[filepath.Base(f.Name)] = f
	}
	return out
}

// The tar strategy builds its headers with tar.FileInfoHeader, so a tarred
// artifact keeps its mode and timestamps. zip.Writer.Create builds a header
// from the name alone, so the zip strategy used to hand back every file as
// 0666 with a zero timestamp — an archived executable arrived without its exec
// bit, which is the user-visible half of this.
func TestZipPreservesModeAndModTime(t *testing.T) {
	modTime := time.Date(2020, 3, 4, 5, 6, 7, 0, time.UTC)

	newSource := func(t *testing.T) (dir string, script string) {
		t.Helper()
		dir = t.TempDir()
		script = filepath.Join(dir, "run.sh")
		require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755))
		require.NoError(t, os.Chtimes(script, modTime, modTime))
		return dir, script
	}

	assertEntry := func(t *testing.T, entry *zip.File, source string) {
		t.Helper()
		require.NotNil(t, entry)

		// Compare against the mode the file actually has rather than the 0755
		// it was written with. Windows has no executable bit — os.Chmod there
		// only toggles the read-only attribute — so the source is 0666 and this
		// compares 0666 to 0666, while the modification time below still has
		// teeth on every platform.
		info, err := os.Stat(source)
		require.NoError(t, err)
		if runtime.GOOS != "windows" {
			require.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "fixture is not executable, so the assertion below would prove nothing")
		}
		require.Equal(t, info.Mode().Perm(), entry.Mode().Perm(), "the archive must carry the file's mode")

		require.True(t, entry.Modified.Equal(modTime), "modification time = %v, want %v", entry.Modified, modTime)
		// zip.FileInfoHeader leaves Method at Store; losing compression while
		// fixing the mode would be a silent regression.
		require.Equal(t, zip.Deflate, entry.Method, "entries must stay compressed")
	}

	t.Run("single file", func(t *testing.T) {
		_, script := newSource(t)
		assertEntry(t, zipEntries(t, script)["run.sh"], script)
	})

	t.Run("directory", func(t *testing.T) {
		dir, script := newSource(t)
		assertEntry(t, zipEntries(t, dir)["run.sh"], script)
	})
}

// Characterization test, not a statement that this is right: zipDir writes a
// symlink as an entry with no mode and no content, so it extracts as an empty
// regular file. tar records the target in Linkname instead. Changing that would
// make argo emit archives whose symlinks an external extractor restores, which
// is a separate decision from carrying file modes — see TestUnzipMaliciousSymlink
// in workflow/executor for how argo's own extractor treats them today.
func TestZipLeavesSymlinksAsEmptyEntriesToday(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink on Windows needs a privilege the runner does not have")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "target.txt"), []byte("payload"), 0o644))
	require.NoError(t, os.Symlink("target.txt", filepath.Join(dir, "link.txt")))

	entry := zipEntries(t, dir)["link.txt"]
	require.NotNil(t, entry)
	require.Zero(t, entry.Mode()&os.ModeSymlink, "today the entry is not marked a symlink")

	rc, err := entry.Open()
	require.NoError(t, err)
	defer rc.Close()
	content, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Empty(t, content, "today the entry carries no link target")
}
