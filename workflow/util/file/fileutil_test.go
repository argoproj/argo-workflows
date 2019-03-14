package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResubmitWorkflowWithOnExit ensures we do not carry over the onExit node even if successful
func TestCompressContentString(t *testing.T) {
	content := "{\"pod-limits-rrdm8-591645159\":{\"id\":\"pod-limits-rrdm8-591645159\",\"name\":\"pod-limits-rrdm8[0]." +
		"run-pod(0:0)\",\"displayName\":\"run-pod(0:0)\",\"type\":\"Pod\",\"templateName\":\"run-pod\",\"phase\":" +
		"\"Succeeded\",\"boundaryID\":\"pod-limits-rrdm8\",\"startedAt\":\"2019-03-07T19:14:50Z\",\"finishedAt\":" +
		"\"2019-03-07T19:14:55Z\"}}"

	contentByte := []byte(content)
	comByte := CompressContent(contentByte)
	compressedContent := EncodeContent(comByte)
	assert.True(t, len(contentByte) > len(compressedContent))
	deCodeContent, _ := DecodeContent(compressedContent)

	deCompress, _ := DecompressContent(deCodeContent)

	deContent := string(deCompress)

	assert.Equal(t, content, deContent)

	compString := CompressEncodeString(content)

	resultString, _ := DecodeDecompressString(compString)

	assert.Equal(t, content, resultString)
}
