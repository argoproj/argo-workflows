package osspecific

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CreateTemp writes what r reads to a new temporary file in dir, named after
// name, with mode perm whatever the umask, and returns its path.
func CreateTemp(dir, name string, r io.Reader, perm fs.FileMode) (string, error) {
	f, err := os.CreateTemp(dir, "."+name+"-*")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, r)
	if err == nil {
		err = f.Chmod(perm)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// WriteFileAtomic replaces dir/name, even a read-only one, with what r
// reads, with mode perm: it is written to a temporary file in dir and
// renamed into place, so a reader sees the old content or the new, never
// part of it.
func WriteFileAtomic(dir, name string, r io.Reader, perm fs.FileMode) error {
	tmp, err := CreateTemp(dir, name, r, perm)
	if err != nil {
		return err
	}
	if err := replaceFile(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
