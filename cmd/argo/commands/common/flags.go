package common

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

// EnumFlagValue represents a CLI flag that can take one of a fixed set of values, and validates
// that the provided value is one of the allowed values.
// There's several libraries for this (e.g. https://github.com/thediveo/enumflag), but they're overkill.
type EnumFlagValue struct {
	AllowedValues []string
	Value         string
}

func (e *EnumFlagValue) Usage() string {
	return fmt.Sprintf("One of: %s", strings.Join(e.AllowedValues, "|"))
}

func (e *EnumFlagValue) String() string {
	return e.Value
}

func (e *EnumFlagValue) Set(v string) error {
	if slices.Contains(e.AllowedValues, v) {
		e.Value = v
		return nil
	}
	return errors.New(e.Usage())
}

func (e *EnumFlagValue) Type() string {
	return "string"
}

var _ pflag.Value = &EnumFlagValue{}

func NewPrintWorkflowOutputValue(value string) EnumFlagValue {
	return EnumFlagValue{
		AllowedValues: []string{"name", "json", "yaml", "wide"},
		Value:         value,
	}
}
