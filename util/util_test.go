package util_test

import (
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo/util"
	"github.com/stretchr/testify/assert"
)

func TestCopyFile(t *testing.T) {
	srcf, err := ioutil.TempFile("", "test-copy-file")
	defer util.Close(srcf)
	assert.NoError(t, err)
	expected := []byte("foobar")
	ioutil.WriteFile(srcf.Name(), expected, 0)
	dstf, err := ioutil.TempFile("", "test-copy-file")
	defer util.Close(dstf)
	assert.NoError(t, err)

	err = util.CopyFile(dstf.Name(), srcf.Name())
	assert.NoError(t, err)

	actual, err := ioutil.ReadFile(dstf.Name())
	assert.Equal(t, expected, actual)
}
