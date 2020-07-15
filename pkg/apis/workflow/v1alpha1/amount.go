package v1alpha1

import "strconv"

/**
This inspired by intstr.IntOrStr and json.Number.
*/

// Amount represent a numeric amount.
type Amount struct {
	Value []byte `json:"value" protobuf:"bytes,1,opt,name=value"`
}

func NewAmount(s string) Amount {
	return Amount{Value: []byte(s)}
}

func (a *Amount) UnmarshalJSON(value []byte) error {
	a.Value = value
	return nil
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return a.Value, nil
}

func (a Amount) OpenAPISchemaType() []string {
	return []string{"number"}
}

func (a Amount) OpenAPISchemaFormat() string { return "" }

func (a *Amount) Float64() (float64, error) {
	return strconv.ParseFloat(string(a.Value), 64)
}
