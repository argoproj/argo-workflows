package logs

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/valyala/fasttemplate"
	"k8s.io/client-go/util/jsonpath"
)

type (
	logFormatAnnotation struct {
		Format    string `json:"format"`
		Extractor struct {
			ExtractorType string `json:"type"`
			Fields        map[string]struct {
				Path     string `json:"path"`
				Required bool   `json:"required"`
			} `json:"fields"`
		} `json:"extractor"`
		IgnoreExtractError bool `json:"ignoreExtractError"`
	}

	// LogFormatter implements log format
	LogFormatter interface {
		Format(raw string) (string, error)
	}

	// RawLogFormatter output raw log directly
	RawLogFormatter struct{}

	// JsonpathLogFormatter format log using jsonpath
	JsonpathLogFormatter struct {
		fstTmpl            *fasttemplate.Template
		ignoreExtractError bool
		jsonPaths          map[string]*jsonpath.JSONPath
	}
)

const extractorTypeJsonPath = "jsonpath"

func (f *RawLogFormatter) Format(raw string) (string, error) {
	return raw, nil
}

func (f *JsonpathLogFormatter) Format(raw string) (string, error) {
	var rawJson interface{}
	if err := json.Unmarshal([]byte(raw), &rawJson); err != nil {
		if f.ignoreExtractError {
			return raw, nil
		}
		return "", fmt.Errorf("failed to parse log as json: %w", err)
	}

	params := make(map[string]interface{})
	for key, jsonPath := range f.jsonPaths {
		buf := new(bytes.Buffer)
		if err := jsonPath.Execute(buf, rawJson); err != nil {
			if f.ignoreExtractError {
				return raw, nil
			}
			return "", fmt.Errorf("unable to find log format json path field: %w", err)
		}
		params[key] = buf.String()
	}

	return f.fstTmpl.ExecuteString(params), nil
}

// NewLogFormatter returns a log formatter
func NewLogFormatter(metadata string) (LogFormatter, error) {
	if metadata == "" {
		return &RawLogFormatter{}, nil
	}

	var annotation logFormatAnnotation
	if err := json.Unmarshal([]byte(metadata), &annotation); err != nil {
		return nil, fmt.Errorf("failed to unmarshall log format metadata: %w", err)
	}

	switch annotation.Extractor.ExtractorType {
	case extractorTypeJsonPath:
		fstTmpl, err := fasttemplate.NewTemplate(annotation.Format, "{{", "}}")
		if err != nil {
			return nil, fmt.Errorf("failed to generate log formate template: %w", err)
		}
		jsonPaths := make(map[string]*jsonpath.JSONPath)
		for key, field := range annotation.Extractor.Fields {
			jsonPath := jsonpath.
				New(fmt.Sprintf("log format %s", field.Path)).
				AllowMissingKeys(!field.Required)
			if err := jsonPath.Parse(fmt.Sprintf("{%s}", field.Path)); err != nil {
				return nil, fmt.Errorf("failed to parse log format jsonpath %s: %w", field.Path, err)
			}
			jsonPaths[key] = jsonPath
		}
		return &JsonpathLogFormatter{
			fstTmpl:            fstTmpl,
			ignoreExtractError: annotation.IgnoreExtractError,
			jsonPaths:          jsonPaths,
		}, nil
	default:
		return nil, fmt.Errorf("no implement log format extractor type: %s", annotation.Extractor.ExtractorType)
	}
}
