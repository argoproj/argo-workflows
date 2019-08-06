package archive

import (
	"bufio"
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
		os.MkdirAll(dir, 0700)
	}
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	filePath := filepath.Join(dir, prefix+hex.EncodeToString(randBytes)+suffix)
	return os.Create(filePath)
}

func TestTarDirectory(t *testing.T) {
	f, err := tempFile(os.TempDir()+"/argo-test", "dir-", ".tgz")
	assert.Nil(t, err)
	log.Infof("Taring to %s", f.Name())
	w := bufio.NewWriter(f)

	err = TarGzToWriter("../../test/e2e", w)
	assert.Nil(t, err)

	err = f.Close()
	assert.Nil(t, err)
}

func TestTarFile(t *testing.T) {
	data, err := tempFile(os.TempDir()+"/argo-test", "file-", "")
	assert.Nil(t, err)
	_, err = data.WriteString("hello world")
	assert.Nil(t, err)
	data.Close()

	dataTarPath := data.Name() + ".tgz"
	f, err := os.Create(dataTarPath)
	assert.Nil(t, err)
	log.Infof("Taring to %s", f.Name())
	w := bufio.NewWriter(f)

	err = TarGzToWriter(data.Name(), w)
	assert.Nil(t, err)
	err = os.Remove(data.Name())
	assert.Nil(t, err)

	err = f.Close()
	assert.Nil(t, err)
}
