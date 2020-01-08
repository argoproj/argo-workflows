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
		err := os.MkdirAll(dir, 0700)
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
	f, err := tempFile(os.TempDir()+"/argo-test", "dir-", ".tgz")
	assert.NoError(t, err)
	log.Infof("Taring to %s", f.Name())
	w := bufio.NewWriter(f)

	err = TarGzToWriter("../../test/e2e", w)
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)
}

func TestTarFile(t *testing.T) {
	data, err := tempFile(os.TempDir()+"/argo-test", "file-", "")
	assert.NoError(t, err)
	_, err = data.WriteString("hello world")
	assert.NoError(t, err)
	data.Close()

	dataTarPath := data.Name() + ".tgz"
	f, err := os.Create(dataTarPath)
	assert.NoError(t, err)
	log.Infof("Taring to %s", f.Name())
	w := bufio.NewWriter(f)

	err = TarGzToWriter(data.Name(), w)
	assert.NoError(t, err)
	err = os.Remove(data.Name())
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)
}
