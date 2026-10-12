package parameterization

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

const InputContractAnnotation = Group + "/input-contract"
const OutputContractAnnotation = Group + "/output-contract"

// FreezeActionContract stores an inert schema, so native interpolation cannot
// treat a schema example or pattern as a Workflow expression.
func FreezeActionContract(schema map[string]any) (string, error) {
	if schema["type"] != "object" || len(CheckSchema(schema, "")) > 0 {
		return "", fmt.Errorf("unsupported action contract")
	}
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		field, ok := raw.(map[string]any)
		if !ok {
			return "", fmt.Errorf("action parameter %s needs a concrete encoding type", name)
		}
		typ, _ := field["type"].(string)
		switch typ {
		case "string", "integer", "number", "boolean", "object", "array", "null":
		default:
			return "", fmt.Errorf("action parameter %s needs a concrete encoding type", name)
		}
	}
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	if _, err := loader.Compile(gojsonschema.NewGoLoader(schema)); err != nil {
		return "", fmt.Errorf("invalid action schema")
	}
	raw, err := json.Marshal(schema)
	if err != nil || len(raw) > 32768 {
		return "", fmt.Errorf("action contract exceeds 32 KiB")
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// ValidateActionValues validates native string parameters without logging their
// contents or changing them. Non-string properties use the declared JSON wire encoding.
func ValidateActionValues(encoded string, params []wfv1.Parameter, result *string) error {
	if encoded == "" {
		return nil
	}
	if len(encoded) > 45000 {
		return fmt.Errorf("invalid action contract")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("invalid action contract")
	}
	var schema map[string]any
	if json.Unmarshal(raw, &schema) != nil {
		return fmt.Errorf("invalid action contract")
	}
	if _, err = FreezeActionContract(schema); err != nil {
		return err
	}
	props, _ := schema["properties"].(map[string]any)
	values := map[string]any{}
	for _, param := range params {
		if param.Value == nil {
			continue
		}
		if _, exists := values[param.Name]; exists {
			return fmt.Errorf("duplicate action field /%s", Escape(param.Name))
		}
		field, ok := props[param.Name].(map[string]any)
		if !ok {
			return fmt.Errorf("unexpected action field /%s", Escape(param.Name))
		}
		value := param.Value.String()
		if field["type"] == "string" {
			values[param.Name] = value
			continue
		}
		if !json.Valid([]byte(value)) {
			return fmt.Errorf("invalid action JSON at /%s", Escape(param.Name))
		}
		decoder := json.NewDecoder(strings.NewReader(value))
		decoder.UseNumber()
		var typed any
		if decoder.Decode(&typed) != nil {
			return fmt.Errorf("invalid action JSON at /%s", Escape(param.Name))
		}
		values[param.Name] = typed
	}
	if result != nil {
		if _, exists := props["result"]; exists {
			if _, collision := values["result"]; collision {
				return fmt.Errorf("ambiguous action result mapping")
			}
			value := wfv1.AnyString(*result)
			field := props["result"].(map[string]any)
			if field["type"] == "string" {
				values["result"] = *result
			} else {
				if !json.Valid([]byte(value)) {
					return fmt.Errorf("invalid action JSON at /result")
				}
				decoder := json.NewDecoder(strings.NewReader(*result))
				decoder.UseNumber()
				var typed any
				if decoder.Decode(&typed) != nil {
					return fmt.Errorf("invalid action JSON at /result")
				}
				values["result"] = typed
			}
		}
	}
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	compiled, err := loader.Compile(gojsonschema.NewGoLoader(schema))
	if err != nil {
		return fmt.Errorf("invalid action contract")
	}
	validated, err := compiled.Validate(gojsonschema.NewGoLoader(values))
	if err != nil {
		return fmt.Errorf("action value validation unavailable")
	}
	if !validated.Valid() {
		return fmt.Errorf("action contract rejected field %s", errorPointer(validated.Errors()[0].Context(), values))
	}
	return nil
}

func ValidateActionInputs(tmpl *wfv1.Template) error {
	if tmpl == nil {
		return nil
	}
	return ValidateActionValues(tmpl.Metadata.Annotations[InputContractAnnotation], tmpl.Inputs.Parameters, nil)
}
func ValidateActionOutputs(tmpl *wfv1.Template, outputs *wfv1.Outputs) error {
	if tmpl == nil {
		return nil
	}
	var params []wfv1.Parameter
	var result *string
	if outputs != nil {
		params = outputs.Parameters
		result = outputs.Result
	}
	return ValidateActionValues(tmpl.Metadata.Annotations[OutputContractAnnotation], params, result)
}
