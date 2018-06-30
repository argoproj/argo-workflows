package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Type represents the stored type of Item.
type Type int

const (
	Number Type = iota
	String
	Bool
	Map
)

// Item expands a single workflow step into multiple parallel steps
// The value of Item can be a map, string, bool, or number
//
// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type Item struct {
	Type    Type                 `protobuf:"bytes,1,opt,name=type,casttype=Type"`
	NumVal  json.Number          `protobuf:"bytes,2,opt,name=numVal"`
	BoolVal bool                 `protobuf:"bytes,3,opt,name=boolVal"`
	StrVal  string               `protobuf:"bytes,4,opt,name=strVal"`
	MapVal  map[string]ItemValue `protobuf:"bytes,5,opt,name=mapVal"`
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (i *Item) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		i.Type = String
		return json.Unmarshal(value, &i.StrVal)
	}
	if value[0] == '{' {
		i.Type = Map
		return json.Unmarshal(value, &i.MapVal)
	}
	lowerVal := strings.ToLower(string(value))
	if lowerVal == "true" || lowerVal == "false" {
		i.Type = Bool
		return json.Unmarshal(value, &i.BoolVal)
	}
	i.Type = Number
	return json.Unmarshal(value, &i.NumVal)
}

func (i *Item) String() string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	if i.Type == String {
		// chop off the double quotes
		return string(jsonBytes[1 : len(jsonBytes)-1])
	}
	return string(jsonBytes)
}

func (i Item) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, i.String())
}

// MarshalJSON implements the json.Marshaller interface.
func (i Item) MarshalJSON() ([]byte, error) {
	switch i.Type {
	case String:
		return json.Marshal(i.StrVal)
	case Bool:
		return json.Marshal(i.BoolVal)
	case Number:
		return json.Marshal(i.NumVal)
	case Map:
		return json.Marshal(i.MapVal)
	default:
		return []byte{}, fmt.Errorf("impossible Item.Type")
	}
}

// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type ItemValue struct {
	Type    Type        `protobuf:"varint,1,opt,name=type,casttype=Type"`
	NumVal  json.Number `protobuf:"bytes,2,opt,name=numVal"`
	BoolVal bool        `protobuf:"bytes,3,opt,name=boolVal"`
	StrVal  string      `protobuf:"bytes,4,opt,name=strVal"`
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (iv *ItemValue) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		iv.Type = String
		return json.Unmarshal(value, &iv.StrVal)
	}
	if value[0] == '{' || value[0] == '[' {
		return fmt.Errorf("ItemValues can only be strings, numbers, or bools")
	}
	lowerVal := strings.ToLower(string(value))
	if lowerVal == "true" || lowerVal == "false" {
		iv.Type = Bool
		return json.Unmarshal(value, &iv.BoolVal)
	}
	iv.Type = Number
	return json.Unmarshal(value, &iv.NumVal)
}

func (iv *ItemValue) String() string {
	jsonBytes, err := json.Marshal(iv)
	if err != nil {
		panic(err)
	}
	if iv.Type == String {
		// chop off the double quotes
		return string(jsonBytes[1 : len(jsonBytes)-1])
	}
	return string(jsonBytes)
}

func (iv ItemValue) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, iv.String())
}

// MarshalJSON implements the json.Marshaller interface.
func (iv ItemValue) MarshalJSON() ([]byte, error) {
	switch iv.Type {
	case String:
		return json.Marshal(iv.StrVal)
	case Bool:
		return json.Marshal(iv.BoolVal)
	case Number:
		return json.Marshal(iv.NumVal)
	default:
		return []byte{}, fmt.Errorf("impossible ItemValue.Type")
	}
}
