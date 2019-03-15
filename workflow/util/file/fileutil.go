package file

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"io/ioutil"
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
	defer close(fi)

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

//Close the file
func close(f io.Closer) {
	err := f.Close()
	if err != nil {
		log.Warn("Failed to close the file/writer/reader. ", err)
	}
}

//EncodeContent will encode using base64
func EncodeContent(content []byte) string {
	encoder := base64.StdEncoding
	return encoder.EncodeToString(content)

}

//DecodeContent will decode using base64
func DecodeContent(content string) ([]byte, error) {
	encoder := base64.StdEncoding
	return encoder.DecodeString(content)
}

//CompressEncodeString will return the compressed string with base64 encoded
func CompressEncodeString(content string) string {
	return EncodeContent(CompressContent([]byte(content)))
}

//DecodeDecompressString will return  decode and decompress the
func DecodeDecompressString(content string) (string, error) {

	buf, err := DecodeContent(content)
	if err != nil {
		return "", err
	}
	dBuf, err := DecompressContent(buf)
	if err != nil {
		return "", err
	}
	return string(dBuf), nil
}

//CompressContent will compress the byte array using zip writer
func CompressContent(content []byte) []byte {
	var buf bytes.Buffer
	zipWriter := gzip.NewWriter(&buf)

	_, err := zipWriter.Write(content)
	if err != nil {
		log.Warn("Error in compressing. v%", err)
	}
	close(zipWriter)
	return buf.Bytes()
}

//D
func DecompressContent(content []byte) ([]byte, error) {

	buf := bytes.NewReader(content)
	gZipReader, _ := gzip.NewReader(buf)
	defer close(gZipReader)
	return ioutil.ReadAll(gZipReader)
}
