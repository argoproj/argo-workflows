package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPUnmarshalJSON_TimeoutSeconds_Number(t *testing.T) {
	jsonData := []byte(`{"timeoutSeconds": 10}`)

	var httpTemplate HTTP
	err := json.Unmarshal(jsonData, &httpTemplate)
	require.NoError(t, err)

	if assert.NotNil(t, httpTemplate.TimeoutSeconds) {
		assert.Equal(t, int64(10), *httpTemplate.TimeoutSeconds)
	}
}

func TestHTTPUnmarshalJSON_TimeoutSeconds_StringNumeric(t *testing.T) {
	jsonData := []byte(`{"timeoutSeconds": "15"}`)

	var httpTemplate HTTP
	err := json.Unmarshal(jsonData, &httpTemplate)
	require.NoError(t, err)

	if assert.NotNil(t, httpTemplate.TimeoutSeconds) {
		assert.Equal(t, int64(15), *httpTemplate.TimeoutSeconds)
	}
}

func TestHTTPUnmarshalJSON_TimeoutSeconds_InvalidString(t *testing.T) {
	jsonData := []byte(`{"timeoutSeconds": "not-a-number"}`)

	var httpTemplate HTTP
	err := json.Unmarshal(jsonData, &httpTemplate)
	require.Error(t, err)
}
