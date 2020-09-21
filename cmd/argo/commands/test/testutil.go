package test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

//
func ExecuteCommand(t *testing.T, command *cobra.Command) string {
	execFunc := func() {
		os.Setenv("ARGO_NAMESPACE", "default")
		err := command.Execute()
		assert.NoError(t, err)
	}
	return CaptureOutput(execFunc)
}