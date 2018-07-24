package packr

import (
	"io"
	"os"
)

type File interface {
	io.ReadCloser
	io.Writer
	FileInfo() (os.FileInfo, error)
	Readdir(count int) ([]os.FileInfo, error)
	Seek(offset int64, whence int) (int64, error)
	Stat() (os.FileInfo, error)
}
