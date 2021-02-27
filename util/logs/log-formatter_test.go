package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogFormatter(t *testing.T) {
	t.Run("RawLogFormatter", func(t *testing.T) {
		f, _ := NewLogFormatter("")
		assert.IsType(t, &RawLogFormatter{}, f)
	})
	t.Run("JsonpathLogFormatter", func(t *testing.T) {
		f, _ := NewLogFormatter(`{
  "format": "hello {{blah}}",
  "extractor": {
    "type": "jsonpath",
    "fields": {
      "blah": { "path": ".message", "required": true }
    }
  },
  "ignoreExtractError": true
}`)
		assert.IsType(t, &JsonpathLogFormatter{}, f)
	})
	t.Run("unknown extractor type", func(t *testing.T) {
		_, err := NewLogFormatter(`{"extractor":{"type": "unknown"}}`)
		assert.Error(t, err)
	})
	t.Run("invalid metadata", func(t *testing.T) {
		_, err := NewLogFormatter("invalid")
		assert.Error(t, err)
	})
}

func TestRawLogFormatter(t *testing.T) {
	t.Run("raw", func(t *testing.T) {
		f, _ := NewLogFormatter("")
		output, _ := f.Format("aaa")
		assert.Equal(t, output, "aaa")
	})
}

func TestJsonpathLogFormatter(t *testing.T) {
	t.Run("JsonpathLogFormatter", func(t *testing.T) {
		f, _ := NewLogFormatter(`{
  "format": "[{{level}}] {{message}} {{stacktrace}}",
  "extractor": {
    "type": "jsonpath",
    "fields": {
      "level": { "path": ".level", "required": true },
      "message": { "path": ".message", "required": true },
      "stacktrace": { "path": ".stacktrace", "required": false }
    }
  },
  "ignoreExtractError": true
}`)
		output, _ := f.Format(`{"level": 1, "message": "hello"}`)
		assert.Equal(t, "[1] hello ", output)
		output, _ = f.Format(`{"level": 1, "message": "hello", "stacktrace": "aaa"}`)
		assert.Equal(t, "[1] hello aaa", output)
		output, err := f.Format(`{"message": "no required level field"}`)
		assert.NoError(t, err)
		assert.Equal(t, `{"message": "no required level field"}`, output)
		output, err = f.Format(`not json`)
		assert.NoError(t, err)
		assert.Equal(t, `not json`, output)
	})
	t.Run("ignoreExtractError false", func(t *testing.T) {
		f, _ := NewLogFormatter(`{
  "format": "{{level}}",
  "extractor": {
    "type": "jsonpath",
    "fields": {
      "level": { "path": ".level", "required": true }
    }
  },
  "ignoreExtractError": false
}`)
		_, err := f.Format(`not json`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse log as json")
		_, err = f.Format(`{"message": "no required level field"}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to find log format json path field")
	})
}
