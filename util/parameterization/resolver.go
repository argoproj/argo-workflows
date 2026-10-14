package parameterization

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

var supported = map[string]bool{}

func init() {
	for key := range strings.FieldsSeq("$schema $ref definitions title description type properties additionalProperties required items additionalItems minItems maxItems uniqueItems enum const minimum maximum exclusiveMinimum exclusiveMaximum multipleOf minLength maxLength pattern minProperties maxProperties allOf anyOf oneOf not if then else default examples readOnly writeOnly") {
		supported[key] = true
	}
}

// CheckSchema refuses unsupported validation rather than silently weakening a contract.
func CheckSchema(schema any, path string) []FieldError {
	if _, ok := schema.(bool); ok {
		return nil
	}
	s, ok := schema.(map[string]any)
	if !ok {
		return []FieldError{{Code: "InvalidSchema", Path: path, Message: "expected an object or boolean schema"}}
	}
	var errors []FieldError
	for k, v := range s {
		p := path + "/" + Escape(k)
		if !supported[k] {
			errors = append(errors, FieldError{Code: "UnsupportedKeyword", Path: p, Message: "unsupported schema keyword"})
			continue
		}
		if k == "$ref" {
			r, ok := v.(string)
			if !ok || !strings.HasPrefix(r, "#/definitions/") {
				errors = append(errors, FieldError{Code: "UnsupportedReference", Path: p, Message: "only local definitions references are supported"})
			}
		}
		switch k {
		case "properties", "definitions":
			if m, ok := v.(map[string]any); ok {
				for name, child := range m {
					errors = append(errors, CheckSchema(child, p+"/"+Escape(name))...)
				}
			}
		case "items", "additionalProperties", "additionalItems", "not", "if", "then", "else":
			errors = append(errors, CheckSchema(v, p)...)
		case "oneOf", "anyOf", "allOf":
			if list, ok := v.([]any); ok {
				for i, child := range list {
					errors = append(errors, CheckSchema(child, fmt.Sprintf("%s/%d", p, i))...)
				}
			}
		}
	}
	sort.Slice(errors, func(i, j int) bool { return errors[i].Path < errors[j].Path })
	return errors
}

func Defaults(schema map[string]any, value any, path string, sources map[string]string) any {
	switch v := value.(type) {
	case map[string]any:
		props, _ := schema["properties"].(map[string]any)
		for k, raw := range props {
			child, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			p := path + "/" + Escape(k)
			if _, exists := v[k]; !exists {
				if d, has := child["default"]; has {
					v[k] = Clone(d)
					sources[p] = "default"
				}
			}
			if current, exists := v[k]; exists {
				v[k] = Defaults(child, current, p, sources)
			}
		}
	case []any:
		if child, ok := schema["items"].(map[string]any); ok {
			for i, item := range v {
				v[i] = Defaults(child, item, fmt.Sprintf("%s/%d", path, i), sources)
			}
		}
	}
	return value
}

// ValidateModelSchema validates definitions without inventing required input values.
// Save and execution use the same compiler before validating concrete values.
func ValidateModelSchema(model Model) []FieldError {
	_, errors := compileModelSchema(model)
	return errors
}

func compileModelSchema(model Model) (*gojsonschema.Schema, []FieldError) {
	if err := checkComplexity(model.Schema, 0, new(int)); err != nil {
		return nil, []FieldError{{Code: "InvalidSchema", Path: "/schema", Message: err.Error()}}
	}
	if model.Dialect != Dialect {
		return nil, []FieldError{{Code: "UnsupportedDialect", Path: "/dialect", Message: "only JSON Schema draft-07 is supported"}}
	}
	if errors := CheckSchema(model.Schema, "/schema"); len(errors) > 0 {
		return nil, errors
	}
	// Remote loading is impossible after CheckSchema. The declared dialect is forced.
	expanded, expandErr := expandSchema(model.Schema, model.Schema, map[string]bool{}, 0, new(int))
	if expandErr != nil {
		return nil, []FieldError{{Code: "InvalidSchema", Path: "/schema", Message: expandErr.Error()}}
	}
	schema := expanded.(map[string]any)
	if err := validateSensitiveDeclarations(model, schema); err != nil {
		return nil, []FieldError{{Code: "InvalidSchema", Path: "/sensitivePaths", Message: err.Error()}}
	}
	schema["$schema"] = Dialect
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	loader.Validate = true
	compiled, err := loader.Compile(gojsonschema.NewGoLoader(schema))
	if err != nil {
		return nil, []FieldError{{Code: "InvalidSchema", Path: "/schema", Message: "schema compilation failed"}}
	}
	return compiled, nil
}

func Resolve(model Model, input any) (any, map[string]string, []FieldError) {
	sources := map[string]string{}
	value := Clone(input)
	if err := checkComplexity(value, 0, new(int)); err != nil {
		return value, sources, []FieldError{{Code: "InputComplexity", Path: "", Message: err.Error()}}
	}
	compiled, schemaErrors := compileModelSchema(model)
	if len(schemaErrors) > 0 {
		return value, sources, schemaErrors
	}
	expanded, _ := expandSchema(model.Schema, model.Schema, map[string]bool{}, 0, new(int))
	schema := expanded.(map[string]any)
	markSources(value, "", sources)
	value = Defaults(schema, value, "", sources)
	if errors := ValidateSensitiveReferences(model.SensitivePaths, value); len(errors) > 0 {
		return value, sources, errors
	}
	if err := applyDerived(model.Derived, value, sources); err != nil {
		return value, sources, []FieldError{{Code: "InvalidDerivation", Path: "", Message: err.Error()}}
	}
	result, err := compiled.Validate(gojsonschema.NewGoLoader(value))
	if err != nil {
		return value, sources, []FieldError{{Code: "ValidationFailed", Path: "", Message: "unable to validate input"}}
	}
	var errors []FieldError
	for _, e := range result.Errors() {
		errors = append(errors, FieldError{Code: "SchemaValidation", Path: errorPointer(e.Context(), value), Message: e.Type()})
	}
	for _, rule := range model.IdentityRules {
		for _, identityPath := range ExpandPaths(value, rule.Path) {
			a, exists := Pointer(value, identityPath)
			if !exists {
				continue
			}
			array, ok := a.([]any)
			if !ok {
				errors = append(errors, FieldError{Code: "InvalidIdentityRule", Path: identityPath, Message: "identity applies to arrays"})
				continue
			}
			seen := map[string]bool{}
			for i, item := range array {
				key, ok := Pointer(item, rule.KeyPath)
				s, valid := key.(string)
				if !ok || !valid || s == "" || seen[s] {
					errors = append(errors, FieldError{Code: "InvalidItemIdentity", Path: fmt.Sprintf("%s/%d", identityPath, i), Message: "array identities must be unique non-empty strings"})
				}
				seen[s] = true
			}
		}
	}
	// Sensitive values must be explicit references, including every array item.
	errors = append(errors, ValidateSensitiveReferences(model.SensitivePaths, value)...)

	sort.Slice(errors, func(i, j int) bool { return errors[i].Path < errors[j].Path })
	return value, sources, errors
}

func Arguments(model Model, values any, mappings []Mapping) (map[string]string, error) {
	out := map[string]string{}
	for _, m := range mappings {
		if m.Parameter == "" {
			return nil, fmt.Errorf("target parameter is required")
		}
		if _, exists := out[m.Parameter]; exists {
			return nil, fmt.Errorf("duplicate target parameter %s", m.Parameter)
		}
		sensitive := false
		for _, s := range model.SensitivePaths {
			if PathsOverlap(m.Path, s) {
				sensitive = true
				// Only complete reference objects (or an enclosing document)
				// may enter the explicitly named reference encoding.
				if m.Encoding != "secret-reference-json" || len(strings.Split(m.Path, "/")) > len(strings.Split(s, "/")) {
					return nil, fmt.Errorf("sensitive input %s requires reference-only JSON encoding", s)
				}
			}
		}
		if m.Encoding == "secret-reference-json" && !sensitive {
			return nil, fmt.Errorf("reference encoding requires a declared sensitive input")
		}
		if len(ValidateSensitiveReferences(model.SensitivePaths, values)) > 0 {
			return nil, fmt.Errorf("sensitive values must be references")
		}

		v, exists := Pointer(values, m.Path)
		if !exists {
			return nil, fmt.Errorf("input path %s is absent", m.Path)
		}
		encoded, err := Encode(v, m.Encoding)
		if err != nil {
			return nil, err
		}
		out[m.Parameter] = encoded
	}
	return out, nil
}

// Acyclic references are expanded before both validation and defaulting. Recursive
// schemas are not supported in this first version; fail instead of hanging.
func expandSchema(raw any, root map[string]any, stack map[string]bool, depth int, nodes *int) (any, error) {
	*nodes++
	if *nodes > 10000 {
		return nil, fmt.Errorf("schema expansion exceeds 10000 nodes")
	}
	if depth > 64 {
		return nil, fmt.Errorf("schema expansion exceeds 64 levels")
	}
	s, ok := raw.(map[string]any)
	if !ok {
		return raw, nil
	}
	if ref, ok := s["$ref"].(string); ok {
		if stack[ref] {
			return nil, fmt.Errorf("recursive schema references are unsupported")
		}
		target, exists := Pointer(root, strings.TrimPrefix(ref, "#"))
		if !exists {
			return nil, fmt.Errorf("schema reference does not exist")
		}
		stack[ref] = true
		resolved, err := expandSchema(target, root, stack, depth+1, nodes)
		delete(stack, ref)
		return resolved, err
	}
	out := map[string]any{}
	for k, v := range s {
		switch k {
		case "properties", "definitions":
			props, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid schema properties")
			}
			next := map[string]any{}
			for name, child := range props {
				x, err := expandSchema(child, root, stack, depth+1, nodes)
				if err != nil {
					return nil, err
				}
				next[name] = x
			}
			out[k] = next
		case "items", "additionalProperties", "additionalItems", "not", "if", "then", "else":
			x, err := expandSchema(v, root, stack, depth+1, nodes)
			if err != nil {
				return nil, err
			}
			out[k] = x
		case "allOf", "anyOf", "oneOf":
			list, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("invalid schema alternatives")
			}
			next := []any{}
			for _, child := range list {
				x, err := expandSchema(child, root, stack, depth+1, nodes)
				if err != nil {
					return nil, err
				}
				next = append(next, x)
			}
			out[k] = next
		default:
			out[k] = Clone(v)
		}
	}
	return out, nil
}

func errorPointer(ctx *gojsonschema.JsonContext, value any) string {
	// JsonContext's default dot separator is ambiguous for keys like tags.example/key.
	// Pick a delimiter absent from the complete input, then escape every token.
	delimiter := "\x00"
	encoded := fmt.Sprint(value)
	for strings.Contains(encoded, delimiter) {
		delimiter += "\x00"
	}
	parts := strings.Split(ctx.String(delimiter), delimiter)
	if len(parts) < 2 {
		return ""
	}
	for i := 1; i < len(parts); i++ {
		parts[i] = Escape(parts[i])
	}
	return "/" + strings.Join(parts[1:], "/")
}

// Bound compilation/default expansion work before recursion. Numbers outside the
// interoperable JSON integer range must use a string schema to preserve precision.
func checkComplexity(value any, depth int, nodes *int) error {
	*nodes++
	if depth > 64 || *nodes > 10000 {
		return fmt.Errorf("JSON exceeds 64 levels or 10000 values")
	}
	switch v := value.(type) {
	case map[string]any:
		for _, child := range v {
			if err := checkComplexity(child, depth+1, nodes); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if err := checkComplexity(child, depth+1, nodes); err != nil {
				return err
			}
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 9007199254740991 {
			return fmt.Errorf("number exceeds interoperable JSON range; use a string")
		}
	}
	return nil
}
