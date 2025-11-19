package v1alpha1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	v1 "k8s.io/api/core/v1"
)

type HTTPHeaderSource struct {
	SecretKeyRef *v1.SecretKeySelector `json:"secretKeyRef,omitempty" protobuf:"bytes,1,opt,name=secretKeyRef"`
}

type HTTPHeaders []HTTPHeader

// HTTPBodySource contains the source of the HTTP body.
type HTTPBodySource struct {
	Bytes []byte `json:"bytes,omitempty" protobuf:"bytes,1,opt,name=bytes"`
}

func (h HTTPHeaders) ToHeader() http.Header {
	outHeader := make(http.Header)
	for _, header := range h {
		// When this is used, header valueFrom should already be resolved
		if header.ValueFrom != nil {
			continue
		}
		outHeader[header.Name] = []string{header.Value}
	}
	return outHeader
}

type HTTPHeader struct {
	Name      string            `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value     string            `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	ValueFrom *HTTPHeaderSource `json:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}

type HTTP struct {
	// Method is HTTP methods for HTTP Request
	Method string `json:"method,omitempty" protobuf:"bytes,1,opt,name=method"`
	// URL of the HTTP Request
	URL string `json:"url" protobuf:"bytes,2,opt,name=url"`
	// Headers are an optional list of headers to send with HTTP requests
	Headers HTTPHeaders `json:"headers,omitempty" protobuf:"bytes,3,rep,name=headers"`
	// TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty" protobuf:"bytes,4,opt,name=timeoutSeconds"`
	// SuccessCondition is an expression if evaluated to true is considered successful
	SuccessCondition string `json:"successCondition,omitempty" protobuf:"bytes,6,opt,name=successCondition"`
	// Body is content of the HTTP Request
	Body string `json:"body,omitempty" protobuf:"bytes,5,opt,name=body"`
	// BodyFrom is  content of the HTTP Request as Bytes
	BodyFrom *HTTPBodySource `json:"bodyFrom,omitempty" protobuf:"bytes,8,opt,name=bodyFrom"`
	// InsecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" protobuf:"bytes,7,opt,name=insecureSkipVerify"`
}

func (h *HTTP) GetBodyBytes() []byte {
	if h.BodyFrom != nil {
		return h.BodyFrom.Bytes
	}
	return nil
}

// Custom JSON unmarshaller in order to accept both string and int64 values
func (h *HTTP) UnmarshalJSON(data []byte) error {
	type HTTPAlias HTTP

	aux := &struct {
		RawTimeoutSeconds interface{} `json:"timeoutSeconds,omitempty"`
		*HTTPAlias
	}{
		HTTPAlias: (*HTTPAlias)(h),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch rawValue := aux.RawTimeoutSeconds.(type) {

	case nil:
		return nil

	// JSON numbers are always converted to float64
	case float64:
		timeoutInt := int64(rawValue)
		h.TimeoutSeconds = &timeoutInt
		return nil

	case string:
		if rawValue == "" {
			h.TimeoutSeconds = nil
			return nil
		}

		parsedTimeout, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid http.timeoutSeconds %q: %w", rawValue, err)
		}
		h.TimeoutSeconds = &parsedTimeout
		return nil

	default:
		return fmt.Errorf("invalid type for timeoutSeconds: %T", rawValue)
	}
}
