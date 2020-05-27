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
	Type    Type                 `protobuf:"bytes,1,opt,name=type,casttype=Type"`
	NumVal  json.Number          `protobuf:"bytes,2,opt,name=numVal"`
	BoolVal bool                 `protobuf:"bytes,3,opt,name=boolVal"`
	StrVal  string               `protobuf:"bytes,4,opt,name=strVal"`
	MapVal  map[string]ItemValue `protobuf:"bytes,5,opt,name=mapVal"`
	ListVal []ItemValue          `protobuf:"bytes,6,opt,name=listVal"`
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (i *Item) UnmarshalJSON(value []byte) error {
	strValue := string(value)
	if _, err := strconv.Atoi(strValue); err == nil {
		i.Type = Number
		return json.Unmarshal(value, &i.NumVal)
	}

	if _, err := strconv.ParseFloat(strValue, 64); err == nil {
		i.Type = Number
		return json.Unmarshal(value, &i.NumVal)
	}

	if _, err := strconv.ParseBool(strValue); err == nil {
		i.Type = Bool
		return json.Unmarshal(value, &i.BoolVal)
	}
	if value[0] == '[' {
		i.Type = List
		err := json.Unmarshal(value, &i.ListVal)
		fmt.Println(err)
		return err
	}
	if value[0] == '{' {
		i.Type = Map
		return json.Unmarshal(value, &i.MapVal)
	}

	i.Type = String
	return json.Unmarshal(value, &i.StrVal)
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
	fmt.Fprintf(s, i.String()) //nolint
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
	case List:
		return json.Marshal(i.ListVal)
	default:
		return []byte{}, fmt.Errorf("impossible Item.Type")
	}
}

// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type ItemValue struct {
	Type    Type              `protobuf:"varint,1,opt,name=type,casttype=Type"`
	NumVal  json.Number       `protobuf:"bytes,2,opt,name=numVal"`
	BoolVal bool              `protobuf:"bytes,3,opt,name=boolVal"`
	StrVal  string            `protobuf:"bytes,4,opt,name=strVal"`
	MapVal  map[string]string `protobuf:"bytes,5,opt,name=mapVal"`
	ListVal []json.RawMessage `protobuf:"bytes,6,opt,name=listVal"`
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (iv *ItemValue) UnmarshalJSON(value []byte) error {
	strValue := string(value)
	if _, err := strconv.Atoi(strValue); err == nil {
		iv.Type = Number
		return json.Unmarshal(value, &iv.NumVal)
	}

	if _, err := strconv.ParseFloat(strValue, 64); err == nil {
		iv.Type = Number
		return json.Unmarshal(value, &iv.NumVal)
	}

	if _, err := strconv.ParseBool(strValue); err == nil {
		iv.Type = Bool
		return json.Unmarshal(value, &iv.BoolVal)
	}
	if value[0] == '[' {
		iv.Type = List
		err := json.Unmarshal(value, &iv.ListVal)
		fmt.Println(err)
		return err
	}
	if value[0] == '{' {
		iv.Type = Map
		return json.Unmarshal(value, &iv.MapVal)
	}

	iv.Type = String
	return json.Unmarshal(value, &iv.StrVal)

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
	fmt.Fprintf(s, iv.String()) //nolint
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
	case Map:
		return json.Marshal(iv.MapVal)
	default:
		return []byte{}, fmt.Errorf("impossible ItemValue.Type")
	}
}
