package transpiler

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

func keyNotPresentError(key string) error {
	return fmt.Errorf("Could not find %s", key)
}

// intermediate representation used to
// parse into interfaces. The class string is used
// to decode Node into a structure.
type IntermediateRepr struct {
	Class *string
	Node  *yaml.Node
}

func getCWLExpressionInner(str string) *string {
	if len(str) < 3 {
		return nil
	}
	if str[0] != '$' {
		return nil
	}
	if str[1] == '[' && str[len(str)-1] == ']' {
		return &str
	}
	if str[1] == '{' && str[len(str)-1] == '}' {
		return &str
	}
	return nil
}

func (s *Strings) UnmarshalYAML(value *yaml.Node) error {
	strings := make([]string, 0)
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		strings = append(strings, s)
	case yaml.SequenceNode:
		if err := value.Decode(&strings); err != nil {
			return err
		}
	default:
		return errors.New("string | []string expected")
	}
	*s = strings
	return nil
}

func (tys *CommandlineTypes) UnmarshalYAML(value *yaml.Node) error {
	newTys := make([]CommandlineType, 0)
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		var ty CommandlineType
		switch s {
		case "string":
			ty.Kind = CWLStringKind
		case "null":
			ty.Kind = CWLNullKind
		case "boolean":
			ty.Kind = CWLBoolKind
		case "int":
			ty.Kind = CWLIntKind
		case "long":
			ty.Kind = CWLLongKind
		case "float":
			ty.Kind = CWLFloatKind
		case "double":
			ty.Kind = CWLDoubleKind
		case "File":
			ty.Kind = CWLFileKind
		case "Directory":
			ty.Kind = CWLDirectoryKind
		default:
			return fmt.Errorf("%s is not a supported type", s)
		}
		newTys = append(newTys, ty)
	case yaml.MappingNode:
		return errors.New("complex types not supported yet")
	case yaml.SequenceNode:
		return errors.New("array types not supported yet")
	default:
		return errors.New("type not supported")
	}
	*tys = newTys
	return nil
}

func (format *CWLFormat) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		format.Kind = FormatStringKind
		format.String = String(s)
		return nil
	case yaml.SequenceNode:
		s := make([]string, 0)
		if err := value.Decode(&s); err != nil {
			return err
		}
		format.Kind = FormatStringsKind
		format.Strings = s
		return nil
	default:
		return errors.New("string | []string expected")
	}
}

func (listing *LoadListingEnum) UnmarshalYAML(value *yaml.Node) error {
	return errors.New("LoadListingEnum not yet supported")
}

func (input *CommandlineInputParameter) UnmarshalYAML(value *yaml.Node) error {
	type rawParamType CommandlineInputParameter

	err := value.Decode((*rawParamType)(input))
	if err != nil {
		return err
	}
	return nil
}

func (inp *Inputs) UnmarshalYAML(value *yaml.Node) error {
	inputs := make([]CommandlineInputParameter, 0)

	switch value.Kind {
	case yaml.MappingNode:
		m := make(map[string]CommandlineInputParameter)
		err := value.Decode(&m)
		if err != nil {
			return err
		}
		for key, input := range m {
			input.Id = &key
			inputs = append(inputs, input)
		}
	case yaml.SequenceNode:
		err := value.Decode(&inputs)
		if err != nil {
			return err
		}
	default:
		return errors.New("sequence or mapping type expected")
	}
	*inp = inputs
	return nil
}

func (out *Outputs) UnmarshalYAML(value *yaml.Node) error {
	outputs := make([]CommandlineOutputParameter, 0)

	switch value.Kind {
	case yaml.MappingNode:
		m := make(map[string]CommandlineOutputParameter)
		err := value.Decode(&m)
		if err != nil {
			return err
		}
		for key, output := range m {
			output.Id = &key
			outputs = append(outputs, output)
		}
	case yaml.SequenceNode:
		err := value.Decode(&outputs)
		if err != nil {
			return err
		}
	default:
		return errors.New("Sequence or mapping type expected")
	}

	*out = outputs

	return nil
}

func (ir *IntermediateRepr) UnmarshalYAML(value *yaml.Node) error {
	m := make(map[string]interface{})
	err := value.Decode(&m)
	if err != nil {
		return err
	}
	classi, ok := m["class"]
	if ok {
		class, ok := classi.(string)
		if !ok {
			return errors.New("string expected")
		}
		ir.Class = &class
	}
	ir.Node = value
	return nil
}

func (reqs *Requirements) UnmarshalYAML(value *yaml.Node) error {
	rs := make(map[string]IntermediateRepr, 0)
	err := value.Decode(&rs)
	if err != nil {
		rsArray := make([]IntermediateRepr, 0)
		err = value.Decode(&rsArray)
		if err != nil {
			return errors.New("[]requirement or map[class]requirement was expected")
		}
		for _, req := range rsArray {
			if req.Class == nil {
				return errors.New("class expected")
			}
			rs[*req.Class] = req
		}
	}

	newRequests := make([]CWLRequirements, 0)
	for class, req := range rs {
		switch class {
		case "DockerRequirement":
			var d DockerRequirement
			err := req.Node.Decode(&d)
			if err != nil {
				return err
			}
			newRequests = append(newRequests, d)
		default:
			return fmt.Errorf("%s is not implemented", class)
		}
	}
	*reqs = newRequests
	return nil
}

func (hints Hints) UnmarshalYAML(value *yaml.Node) error {
	return errors.New("hints not supported yet")
}

func (args Arguments) UnmarshalYAML(value *yaml.Node) error {
	return errors.New("arguments not supported yet")
}

func (expr *CWLExpression) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return errors.New("can only be string | bool | int | float")
	}
	var f float64
	err := value.Decode(&f)
	if err == nil {
		expr.Kind = FloatKind
		expr.Float = f
		return nil
	}

	var i int
	err = value.Decode(&i)
	if err == nil {
		expr.Kind = IntKind
		expr.Int = i
		return nil
	}

	var b bool
	err = value.Decode(&b)
	if err == nil {
		expr.Kind = BoolKind
		expr.Bool = b
		return nil
	}

	var s string
	err = value.Decode(&s)
	if err == nil {
		exprS := getCWLExpressionInner(s)
		if exprS != nil {
			expr.Kind = ExpressionKind
			expr.Expression = *exprS
			return nil
		}
		expr.Kind = RawKind
		expr.Raw = s
		return nil
	}
	return errors.New("can only be string | bool | int | float")
}

func (cl *CommandlineTool) UnmarshalYAML(value *yaml.Node) error {
	type rawCLITool CommandlineTool
	if err := value.Decode((*rawCLITool)(cl)); err != nil {
		return err
	}
	return nil
}
