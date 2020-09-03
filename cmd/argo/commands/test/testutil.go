package test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
)

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
