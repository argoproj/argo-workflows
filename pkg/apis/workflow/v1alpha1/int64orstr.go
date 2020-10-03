package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
type Int64OrString struct {
	Value string `json:"-" protobuf:"bytes,1,opt,name=value"`
}

func ParseInt64OrString(val interface{}) Int64OrString {
	return Int64OrString{Value: fmt.Sprintf("%v", val)}
}

func Int64OrStringPtr(val interface{}) *Int64OrString {
	i := ParseInt64OrString(val)
	return &i
}

func (i *Int64OrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		x := ""
		err := json.Unmarshal(value, &x)
		i.Value = x
		return err
	}
	x := 0
	err := json.Unmarshal(value, &x)
	i.Value = strconv.Itoa(x)
	return err
}

func (i Int64OrString) MarshalJSON() ([]byte, error) {
	intVal, err := strconv.ParseInt(i.Value, 10, 64)
	if err != nil {
		return json.Marshal(i.Value)
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
	return i.Value
}
