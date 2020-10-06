package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// This is similar to `intstr.IntOrString` (which should be called `intstr.Int32OrString`!!).
// It intended just to tolerate unmarshalling int64. Therefore:
//
// * It's JSON type is just string, not `int-or-string`.
// * It will unmarshall int64 (rather than only int32) and represents it as string.
// * It will marshall back to string - marshalling is not symmetric.
type Int64OrString string

func ParseInt64OrString(val interface{}) Int64OrString {
	return Int64OrString(fmt.Sprintf("%v", val))
}

func Int64OrStringPtr(val interface{}) *Int64OrString {
	i := ParseInt64OrString(val)
	return &i
}

func (i *Int64OrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		v := ""
		err := json.Unmarshal(value, &v)
		if err != nil {
			return err
		}
		*i = Int64OrString(v)
		return nil
	}
	v := 0
	err := json.Unmarshal(value, &v)
	if err != nil {
		return err
	}
	*i = Int64OrString(strconv.Itoa(v))
	return nil
}

func (i Int64OrString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(i))
}

func (i Int64OrString) String() string {
	return string(i)
}
