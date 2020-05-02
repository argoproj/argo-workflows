package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getListArgs() listFlags {
	return listFlags{
		allNamespaces: false,
		status:        []string{},
		completed:     false,
		running:       false,
		prefix:        "",
		output:        "wide",
		since:         "",
		chunkSize:     500,
		noHeaders:     false,
		continueToken: "",
		limit:         500,
	}
}

// TestGetListOpts
func TestGetListOpts(t *testing.T) {
	listArgs := getListArgs()
	opts := getListOpts(&listArgs)
	assert.Equal(t, "", opts.LabelSelector)

	listArgs.status = append(listArgs.status, "RUNNING")
	listArgs.completed = true
	opts = getListOpts(&listArgs)
	assert.Equal(t, "workflows.argoproj.io/completed=true,workflows.argoproj.io/phase in (RUNNING)", opts.LabelSelector)

	listArgs.completed = false
	listArgs.running = true
	opts = getListOpts(&listArgs)
	assert.Equal(t, "workflows.argoproj.io/completed!=true,workflows.argoproj.io/phase in (RUNNING)", opts.LabelSelector)
}

// TestGetKubeCursor
func TestGetKubeCursor(t *testing.T) {
	listArgs := getListArgs()
	cursor, wfName, err := getKubeCursor(&listArgs)
	if assert.Nil(t, err) {
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	listArgs.continueToken = "BLAH"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: Hello World
	listArgs.continueToken = "SGVsbG8gd29ybGQ"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"kube_cursor\":\"foo\"}"
	listArgs.continueToken = "eyJrdWJlX2N1cnNvciI6ImZvbyJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "malformed value")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"prefix\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJwcmVmaXgiOiJiYXIifQ"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "using the identical values for `prefix` and `since`")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"since\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJzaW5jZSI6ImJhciJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "using the identical values for `prefix` and `since`")
		assert.Equal(t, "", cursor)
		assert.Equal(t, "", wfName)
	}

	// decoded: "{\"last_workflow_name\":\"foo\",\"kube_cursor\":\"bar\"}"
	listArgs.continueToken = "eyJsYXN0X3dvcmtmbG93X25hbWUiOiJmb28iLCJrdWJlX2N1cnNvciI6ImJhciJ9"
	cursor, wfName, err = getKubeCursor(&listArgs)
	if assert.Nil(t, err) {
		assert.Equal(t, "bar", cursor)
		assert.Equal(t, "foo", wfName)
	}
}

// TestPrintCursor
func TestPrintCursor(t *testing.T) {
	listArgs := getListArgs()
	var buf bytes.Buffer

	printCursor("", "foo", &listArgs, &buf)
	assert.Contains(t, buf.String(), "There are additional suppressed results")

	buf.Reset()
	printCursor("", "", &listArgs, &buf)
	assert.Equal(t, "", buf.String())
}
