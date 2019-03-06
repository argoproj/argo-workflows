package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"strings"
)

//IsFileOrDirExistInGZip return true if file or directory exists in GZip file
func IsFileOrDirExistInGZip(sourcePath string, gzipFilePath string) bool {

	fi, err := os.Open(gzipFilePath)

	if os.IsNotExist(err) {
		return false
	}
	defer fi.Close()

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return false
	}
	defer fz.Close()
	tr := tar.NewReader(fz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		if hdr.FileInfo().IsDir() && strings.Contains(strings.Trim(hdr.Name, "/"), strings.Trim(sourcePath, "/")) {
			return true
		}
		if strings.Contains(sourcePath, hdr.Name) && hdr.Size > 0 {
			return true
		}
	}
	return false
}
