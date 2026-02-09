package json

import (
	"bytes"
	"encoding/json"
	"io"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// Marshaler is a type which satisfies the grpc-gateway Marshaler interface
type Marshaler struct{}

// ContentType implements gwruntime.Marshaler.
func (j *Marshaler) ContentType() string {
	return "application/json"
}

// Marshal implements gwruntime.Marshaler.
func (j *Marshaler) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// NewDecoder implements gwruntime.Marshaler.
func (j *Marshaler) NewDecoder(r io.Reader) gwruntime.Decoder {
	return json.NewDecoder(r)
}

// NewEncoder implements gwruntime.Marshaler.
func (j *Marshaler) NewEncoder(w io.Writer) gwruntime.Encoder {
	return json.NewEncoder(w)
}

// Unmarshal implements gwruntime.Marshaler.
func (j *Marshaler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// DisallowUnknownFields configures the JSON decoder to error out if unknown
// fields come along, instead of dropping them by default.
func DisallowUnknownFields(d *json.Decoder) *json.Decoder {
	d.DisallowUnknownFields()
	return d
}

// Opt is a decoding option for decoding from JSON format.
type Opt func(*json.Decoder) *json.Decoder

// Unmarshal is a convenience wrapper around json.Unmarshal to support json decode options
func Unmarshal(j []byte, o any, opts ...Opt) error {
	d := json.NewDecoder(bytes.NewReader(j))
	for _, opt := range opts {
		d = opt(d)
	}
	return d.Decode(&o)
}

// UnmarshalStrict is a convenience wrapper around json.Unmarshal with strict unmarshal options
func UnmarshalStrict(j []byte, o any) error {
	return Unmarshal(j, o, DisallowUnknownFields)
}

// IsJSON tests whether or not the suppied byte array is valid JSON
func IsJSON(j []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(j, &js) == nil
}
