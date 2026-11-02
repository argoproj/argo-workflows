package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// AnyString is a string type whose JSON type is just string.
// It will unmarshall int64, int32, float64, float32, boolean, a plain string and represents it as string.
// It will marshall back to string - marshalling is not symmetric.
type AnyString string

func ParseAnyString(val any) AnyString {
	return AnyString(fmt.Sprintf("%v", val))
}

func AnyStringPtr(val any) *AnyString {
	i := ParseAnyString(val)
	return &i
}

func (i *AnyString) UnmarshalJSON(value []byte) error {
	var v any
	err := json.Unmarshal(value, &v)
	if err != nil {
		return err
	}
	switch v := v.(type) {
	case float64:
		*i = AnyString(strconv.FormatFloat(v, 'f', -1, 64))
	case float32:
		*i = AnyString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case int64:
		*i = AnyString(strconv.FormatInt(v, 10))
	case int32:
		*i = AnyString(strconv.FormatInt(int64(v), 10))
	case bool:
		*i = AnyString(strconv.FormatBool(v))
	case string:
		*i = AnyString(v)
	}
	return nil
}

func (i AnyString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(i))
}

func (i AnyString) String() string {
	return string(i)
}
