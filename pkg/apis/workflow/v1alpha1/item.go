package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Type represents the stored type of Item.
type Type int

const (
	Number Type = iota
	String
	Bool
	Map
	List
)

// Item expands a single workflow step into multiple parallel steps
// The value of Item can be a map, string, bool, or number
//
// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type Item struct {
	Value json.RawMessage `json:"value" protobuf:"bytes,1,opt,name=value,casttype=encoding/json.RawMessage"`
}

func ParseItem(s string) (Item, error) {
	item := Item{}
	return item, json.Unmarshal([]byte(s), &item)
}

func (i *Item) GetType() Type {
	strValue := string(i.Value)
	if _, err := strconv.Atoi(strValue); err == nil {
		return Number
	}
	if _, err := strconv.ParseFloat(strValue, 64); err == nil {
		return Number
	}
	if _, err := strconv.ParseBool(strValue); err == nil {
		return Bool
	}
	var list []interface{}
	if err := json.Unmarshal(i.Value, &list); err == nil {
		return List
	}
	var object map[string]interface{}
	if err := json.Unmarshal(i.Value, &object); err == nil {
		return Map
	}
	return String
}

func (i *Item) UnmarshalJSON(value []byte) error {
	return i.Value.UnmarshalJSON(value)
}

func (i *Item) String() string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	if jsonBytes[0] == '"' {
		return string(jsonBytes[1 : len(jsonBytes)-1])
	}
	return string(jsonBytes)
}

func (i Item) Format(s fmt.State, _ rune) {
	_, _ = fmt.Fprintf(s, i.String()) // nolint
}

func (i Item) MarshalJSON() ([]byte, error) {
	return i.Value.MarshalJSON()
}

func (i *Item) DeepCopyInto(out *Item) {
	inBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(inBytes, out)
	if err != nil {
		panic(err)
	}
}

func (i Item) OpenAPISchemaType() []string {
	return []string{}
}

func (i Item) OpenAPISchemaFormat() string { return "" }

// you MUST assert `GetType() == Map` before invocation as this does not return errors
func (i *Item) GetMapVal() map[string]interface{} {
	val := make(map[string]interface{})
	_ = json.Unmarshal(i.Value, &val)
	return val
}

// you MUST assert `GetType() == List` before invocation as this does not return errors
func (i *Item) GetListVal() []interface{} {
	val := make([]interface{}, 0)
	_ = json.Unmarshal(i.Value, &val)
	return val
}

// you MUST assert `GetType() == String` before invocation as this does not return errors
func (i *Item) GetStrVal() string {
	val := ""
	_ = json.Unmarshal(i.Value, &val)
	return val
}
