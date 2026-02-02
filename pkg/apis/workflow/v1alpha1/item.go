package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"

	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
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
// +protobuf.options.(gogoproto.goproto_stringer)=false
type Item struct {
	Value json.RawMessage `json:"-" protobuf:"bytes,1,opt,name=value,casttype=encoding/json.RawMessage"`
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
	var list []any
	if err := json.Unmarshal(i.Value, &list); err == nil {
		return List
	}
	var object map[string]any
	if err := json.Unmarshal(i.Value, &object); err == nil {
		return Map
	}
	return String
}

func (i *Item) UnmarshalJSON(value []byte) error {
	return i.Value.UnmarshalJSON(value)
}

func (i *Item) String() string {
	x, err := json.Marshal(i) // this produces a normalised string, e.g. white-space
	if err != nil {
		panic(err)
	}
	// this convenience to remove quotes from strings will cause many problems
	if x[0] == '"' {
		return jsonutil.Fix(string(x[1 : len(x)-1]))
	}
	return jsonutil.Fix(string(x))
}

func (i Item) Format(s fmt.State, _ rune) {
	_, _ = fmt.Fprintf(s, "%s", i.String())
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

// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (i Item) OpenAPISchemaType() []string {
	return nil
}

func (i Item) OpenAPISchemaFormat() string { return "" }

// you MUST assert `GetType() == Map` before invocation as this does not return errors
func (i *Item) GetMapVal() map[string]Item {
	val := make(map[string]Item)
	_ = json.Unmarshal(i.Value, &val)
	return val
}

// you MUST assert `GetType() == List` before invocation as this does not return errors
func (i *Item) GetListVal() []Item {
	val := make([]Item, 0)
	_ = json.Unmarshal(i.Value, &val)
	return val
}

// you MUST assert `GetType() == String` before invocation as this does not return errors
func (i *Item) GetStrVal() string {
	val := ""
	_ = json.Unmarshal(i.Value, &val)
	return val
}
