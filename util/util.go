package util

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Closer interface {
	Close() error
}

// Close is a convenience function to close a object that has a Close() method, ignoring any errors
// Used to satisfy errcheck lint
func Close(c Closer) {
	_ = c.Close()
}

// CopyFile copies the contents from src to dst.
func CopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer Close(in)
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
