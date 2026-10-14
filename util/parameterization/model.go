// Package parameterization resolves opt-in structured inputs without owning Workflow scheduling.
package parameterization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
)

const Group = "parameters.workflows.example.io"
const Version = "v1alpha1"
const Dialect = "http://json-schema.org/draft-07/schema#"

type Model struct {
	Dialect        string         `json:"dialect"`
	Schema         map[string]any `json:"schema"`
	IdentityRules  []IdentityRule `json:"identityRules,omitempty"`
	Derived        []Derivation   `json:"derived,omitempty"`
	SensitivePaths []string       `json:"sensitivePaths,omitempty"`
	// SecretSchemas validate plaintext inside the authorized secret backend only.
	// The ordinary Schema validates references, never plaintext values.
	SecretSchemas map[string]map[string]any `json:"secretSchemas,omitempty"`
}

// Derivation copies structured data from an explicit source pointer; no executable expressions.
type Derivation = parameterapi.Derivation
type IdentityRule = parameterapi.IdentityRule
type FieldError = parameterapi.FieldError
type Mapping = parameterapi.ParameterMapping
type Change struct {
	Op       string `json:"op"`
	Path     string `json:"path"`
	Before   any    `json:"before"`
	After    any    `json:"after"`
	From     string `json:"from,omitempty"`
	Identity string `json:"itemIdentity,omitempty"`
}

func Digest(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func Clone(value any) any {
	b, _ := json.Marshal(value)
	var result any
	_ = json.Unmarshal(b, &result)
	return result
}
func Escape(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1") }
func Pointer(value any, path string) (any, bool) {
	if path == "" {
		return value, true
	}
	if !strings.HasPrefix(path, "/") {
		return nil, false
	}
	for part := range strings.SplitSeq(path[1:], "/") {
		if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(part, "~0", ""), "~1", ""), "~") {
			return nil, false
		}
		key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch x := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = x[key]
			if !ok {
				return nil, false
			}
		case []any:
			i, e := strconv.Atoi(key)
			if e != nil || i < 0 || i >= len(x) || strconv.Itoa(i) != key {
				return nil, false
			}
			value = x[i]
		default:
			return nil, false
		}
	}
	return value, true
}
func Encode(value any, encoding string) (string, error) {
	switch encoding {
	case "json", "secret-reference-json":
		b, e := json.Marshal(value)
		return string(b), e
	case "string":
		if s, ok := value.(string); ok {
			return s, nil
		}
	case "boolean":
		if b, ok := value.(bool); ok {
			return strconv.FormatBool(b), nil
		}
	case "decimal":
		if n, ok := value.(float64); ok {
			return strconv.FormatFloat(n, 'f', -1, 64), nil
		}
	}
	return "", fmt.Errorf("encoding %q is incompatible with parameter type", encoding)
}
