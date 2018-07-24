package packr

import "os"

var _ File = physicalFile{}

type physicalFile struct {
	*os.File
}

func (p physicalFile) FileInfo() (os.FileInfo, error) {
	return os.Stat(p.Name())
}
