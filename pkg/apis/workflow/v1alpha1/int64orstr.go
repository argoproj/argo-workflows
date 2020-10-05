package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

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
		*i = Int64OrString(v)
		return err
	}
	v := 0
	err := json.Unmarshal(value, &v)
	*i = Int64OrString(strconv.Itoa(v))
	return err
}

func (i Int64OrString) MarshalJSON() ([]byte, error) {
	v, err := strconv.ParseInt(string(i), 10, 64)
	if err != nil {
		return json.Marshal(string(i))
	}
	return json.Marshal(v)
}

func (i Int64OrString) String() string {
	return string(i)
}
