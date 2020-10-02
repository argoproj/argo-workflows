package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
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
		x := ""
		err := json.Unmarshal(value, &x)
		*i = Int64OrString(x)
		return err
	}
	x := 0
	err := json.Unmarshal(value, &x)
	*i = Int64OrString(strconv.Itoa(x))
	return err
}

func (i Int64OrString) MarshalJSON() ([]byte, error) {
	intVal, err := strconv.ParseInt(string(i), 10, 64)
	if err != nil {
		return json.Marshal(string(i))
	}
	return json.Marshal(intVal)
}

func (i Int64OrString) OpenAPISchemaType() []string {
	return []string{"string"}
}

func (i Int64OrString) OpenAPISchemaFormat() string {
	return "int-or-string"
}

func (i Int64OrString) String() string {
	return string(i)
}
