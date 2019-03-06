package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

//IsFileOrDirExistInGZip return true if file or directory exists in GZip file
func IsFileOrDirExistInGZip(sourcePath string, gzipFilePath string) bool {

	fi, err := os.Open(gzipFilePath)

	if os.IsNotExist(err) {
		return false
	}
	defer closeFile(fi)

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return false
	}
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

func closeFile(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Warn("Failed to close the file. v%", err)
	}
}
