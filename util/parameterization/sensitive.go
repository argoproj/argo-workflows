package parameterization

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MatchPath treats '*' as one canonical numeric array index, never an object key.
func MatchPath(pattern, path string) bool {
	a, b := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		if a[i] != "*" || !arrayIndex(b[i]) {
			return false
		}
	}
	return true
}

func arrayIndex(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 0 && strconv.Itoa(n) == s
}

// PathsOverlap also covers ancestors and descendants of sensitive references.
func PathsOverlap(a, b string) bool {
	x, y := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] && (x[i] != "*" || !arrayIndex(y[i])) && (y[i] != "*" || !arrayIndex(x[i])) {
			return false
		}
	}
	return true
}

// ExpandPaths expands against actual data; a malformed array remains an error
// location instead of making the sensitive value disappear from validation.
func ExpandPaths(value any, pattern string) []string {
	result := []string{}
	parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	if pattern == "" {
		return []string{""}
	}
	var walk func(int, string)
	walk = func(i int, path string) {
		if i == len(parts) {
			result = append(result, path)
			return
		}
		if parts[i] != "*" {
			parent, exists := Pointer(value, path)
			if exists && parent != nil {
				switch parent.(type) {
				case map[string]any, []any:
				default:
					result = append(result, path)
					return
				}
			}
			walk(i+1, path+"/"+parts[i])
			return
		}
		v, exists := Pointer(value, path)
		if !exists || v == nil {
			return
		}
		items, ok := v.([]any)
		if !ok {
			result = append(result, path)
			return
		}
		for j := range items {
			walk(i+1, fmt.Sprintf("%s/%d", path, j))
		}
	}
	walk(0, "")
	sort.Strings(result)
	return result
}

func ValidateSensitiveReferences(paths []string, value any) []FieldError {
	errors := []FieldError{}
	seen := map[string]bool{}
	for _, pattern := range paths {
		for _, path := range ExpandPaths(value, pattern) {
			if seen[path] {
				continue
			}
			seen[path] = true
			v, exists := Pointer(value, path)
			if !exists || v == nil {
				continue
			}
			m, ok := v.(map[string]any)
			ref, _ := m["secretRef"].(map[string]any)
			name, _ := ref["name"].(string)
			key, _ := ref["key"].(string)
			if latest, present := m["latestSecretRef"].(map[string]any); present {
				ref = latest
				name, _ = latest["configMap"].(string)
				key, _ = latest["key"].(string)
			}
			if !ok || len(m) != 1 || len(ref) != 2 || name == "" || key == "" || !MatchPath(pattern, path) {
				errors = append(errors, FieldError{Code: "SecretReferenceRequired", Path: path, Message: "sensitive inputs require a Secret reference"})
			}
		}
	}
	return errors
}

// Wildcard declarations require a stable identity at each array boundary.
// This is model semantics, shared by save, UI and reference-only Diff.
func validateSensitiveDeclarations(model Model, schema map[string]any) error {
	for _, pattern := range model.SensitivePaths {
		if !strings.HasPrefix(pattern, "/") || strings.Contains(strings.ReplaceAll(strings.ReplaceAll(pattern, "~0", ""), "~1", ""), "~") {
			return fmt.Errorf("invalid sensitive pointer")
		}
		var current any = schema
		parts := strings.Split(pattern[1:], "/")
		for i, part := range parts {
			node, _ := current.(map[string]any)
			switch {
			case part == "*":
				if node["type"] != "array" {
					return fmt.Errorf("sensitive wildcard requires an array schema")
				}
				parent := "/" + strings.Join(parts[:i], "/")
				found := false
				for _, rule := range model.IdentityRules {
					if rule.Path == parent && strings.HasPrefix(rule.KeyPath, "/") {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("sensitive arrays require stable item identities")
				}
				current = node["items"]
			case node["type"] == "array" && arrayIndex(part):
				current = node["items"]
			default:
				props, _ := node["properties"].(map[string]any)
				current = props[strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")]
			}
		}
	}
	for _, rule := range model.Derived {
		for _, path := range model.SensitivePaths {
			if PathsOverlap(rule.From, path) || PathsOverlap(rule.Path, path) {
				return fmt.Errorf("sensitive references cannot use ordinary derived mappings")
			}
		}
	}
	return nil
}
