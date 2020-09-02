package commands

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ansiColorCode(t *testing.T) {
	// check we get a nice range of colours
	assert.Equal(t, FgYellow, ansiColorCode("foo"))
	assert.Equal(t, FgGreen, ansiColorCode("bar"))
	assert.Equal(t, FgYellow, ansiColorCode("baz"))
	assert.Equal(t, FgRed, ansiColorCode("qux"))
}

func CaptureOutput(f func()) string {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	var buf bytes.Buffer
	log.SetOutput(&buf)
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	os.Stderr = rescueStderr
	return string(out) + buf.String()
}