package packr

import (
	"bytes"
	"fmt"
	"os"
	"time"
)

var virtualFileModTime = time.Now()
var _ File = virtualFile{}

type virtualFile struct {
	*bytes.Reader
	Name string
	info fileInfo
}

func (f virtualFile) FileInfo() (os.FileInfo, error) {
	return f.info, nil
}

func (f virtualFile) Close() error {
	return nil
}

func (f virtualFile) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("not implemented")
}

func (f virtualFile) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{f.info}, nil
}

func (f virtualFile) Stat() (os.FileInfo, error) {
	return f.info, nil
}

func newVirtualFile(name string, b []byte) File {
	return virtualFile{
		Reader: bytes.NewReader(b),
		Name:   name,
		info: fileInfo{
			Path:     name,
			Contents: b,
			size:     int64(len(b)),
			modTime:  virtualFileModTime,
		},
	}
}

func newVirtualDir(name string) File {
	var b []byte
	v := newVirtualFile(name, b).(virtualFile)
	v.info.isDir = true
	return v
}
