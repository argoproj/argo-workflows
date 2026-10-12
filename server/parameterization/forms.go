package parameterization

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func formSchemaAt(root map[string]any, path string) map[string]any {
	var current any = root
	resolve := func() map[string]any {
		seen := map[string]bool{}
		for {
			node, _ := current.(map[string]any)
			if node == nil {
				return nil
			}
			ref, _ := node["$ref"].(string)
			if ref == "" {
				return node
			}
			if !strings.HasPrefix(ref, "#/definitions/") || seen[ref] {
				return nil
			}
			seen[ref] = true
			current, _ = p.Pointer(root, strings.TrimPrefix(ref, "#"))
		}
	}
	if path == "" {
		return resolve()
	}
	if !strings.HasPrefix(path, "/") {
		return nil
	}
	for raw := range strings.SplitSeq(path[1:], "/") {
		if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(raw, "~0", ""), "~1", ""), "~") {
			return nil
		}
		key := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		node := resolve()
		if node == nil {
			return nil
		}
		if node["type"] == "array" {
			i, err := strconv.Atoi(key)
			if key != "*" && (err != nil || i < 0 || strconv.Itoa(i) != key) {
				return nil
			}
			current = node["items"]
		} else {
			props, _ := node["properties"].(map[string]any)
			current = props[key]
		}
	}
	return resolve()
}

// Check presentation references at preview time; hiding/locking fields is not an authorization policy.
func validateForm(form parameterapi.WorkflowFormSpec, model p.Model, template string) error {
	if form.ParameterTemplate != "" && form.ParameterTemplate != template {
		return fmt.Errorf("form targets a different parameter model")
	}
	nodes := map[string]parameterapi.FormLayoutNode{}
	for _, node := range form.Layout {
		if _, exists := nodes[node.ID]; exists || node.ID == "" {
			return fmt.Errorf("form layout IDs must be unique and nonempty")
		}
		if node.Kind != "group" && node.Kind != "tab" && node.Kind != "tree" {
			return fmt.Errorf("unsupported form layout kind")
		}
		nodes[node.ID] = node
	}
	for _, node := range form.Layout {
		seen := map[string]bool{node.ID: true}
		for parent := node.Parent; parent != ""; {
			n, exists := nodes[parent]
			if !exists || seen[parent] {
				return fmt.Errorf("form layout has a missing parent or cycle")
			}
			seen[parent] = true
			parent = n.Parent
		}
	}
	fields := map[string]bool{}
	for _, field := range form.Fields {
		schema := formSchemaAt(model.Schema, field.Path)
		if fields[field.Path] || schema == nil {
			return fmt.Errorf("duplicate or unknown form field: %s", field.Path)
		}
		fields[field.Path] = true
		if field.Group != "" {
			if _, exists := nodes[field.Group]; !exists {
				return fmt.Errorf("form field references an unknown layout group: %s", field.Path)
			}
			segments := strings.Split(field.Path, "/")
			for i := 1; i < len(segments); i++ {
				if formSchemaAt(model.Schema, strings.Join(segments[:i], "/"))["type"] == "array" {
					return fmt.Errorf("group the whole array, not individual array fields: %s", field.Path)
				}
			}
			for _, other := range form.Fields {
				if other.Group != "" && other.Path != field.Path && strings.HasPrefix(field.Path, other.Path+"/") {
					return fmt.Errorf("form layout fields overlap: %s", field.Path)
				}
			}
		}
		allowed := field.Widget == ""
		switch field.Widget {
		case "text":
			allowed = schema["type"] == "string"
		case "number":
			allowed = schema["type"] == "number" || schema["type"] == "integer"
		case "checkbox":
			allowed = schema["type"] == "boolean"
		case "object":
			allowed = schema["type"] == "object"
		case "array":
			allowed = schema["type"] == "array"
		case "table":
			allowed = schema["type"] == "array" && formSchemaAt(model.Schema, field.Path+"/*")["type"] == "object"
		case "yaml":
			allowed = true
		case "select":
			_, hasSource := form.Options[field.Path]
			allowed = schema["enum"] != nil || field.OptionSource != "" || hasSource
		case "secret":
			for _, path := range model.SensitivePaths {
				if path == field.Path {
					allowed = true
				}
			}
		}
		if !allowed {
			return fmt.Errorf("widget is incompatible with form field: %s", field.Path)
		}
		if field.OptionSource != "" {
			if _, exists := form.Options[field.OptionSource]; !exists {
				return fmt.Errorf("form option source is not declared: %s", field.Path)
			}
		}
		for _, condition := range []*parameterapi.FieldPredicate{field.VisibleWhen, field.EnabledWhen} {
			if condition == nil {
				continue
			}
			if (condition.Operator != "Equals" && condition.Operator != "Exists") || formSchemaAt(model.Schema, condition.Path) == nil {
				return fmt.Errorf("invalid form condition: %s", field.Path)
			}
			if wildcardCount(condition.Path) > wildcardCount(field.Path) {
				return fmt.Errorf("form condition has unbound array identity: %s", field.Path)
			}
		}
	}
	sources, err := formOptions(form)
	if err != nil {
		return err
	}
	for path, source := range sources {
		if formSchemaAt(model.Schema, path) == nil {
			return fmt.Errorf("invalid form option field: %s", path)
		}
		for _, dependency := range source.DependsOn {
			if formSchemaAt(model.Schema, dependency) == nil {
				return fmt.Errorf("invalid form option dependency: %s", dependency)
			}
		}
	}
	return nil
}
func wildcardCount(path string) int {
	n := 0
	for part := range strings.SplitSeq(path, "/") {
		if part == "*" {
			n++
		}
	}
	return n
}
func formOptions(form parameterapi.WorkflowFormSpec) (map[string]OptionSource, error) {
	result := map[string]OptionSource{}
	for path, source := range form.Options {
		if strings.HasPrefix(path, "/") {
			result[path] = source
		}
	}
	for _, field := range form.Fields {
		if field.OptionSource != "" {
			source, ok := form.Options[field.OptionSource]
			if !ok {
				return nil, fmt.Errorf("form option source is not declared: %s", field.Path)
			}
			result[field.Path] = source
		}
	}
	return result, nil
}
func matchFormPath(pattern, path string) bool {
	a, b := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(a) != len(b) {
		return false
	}
	for i, key := range a {
		if key == b[i] {
			continue
		}
		n, err := strconv.Atoi(b[i])
		if key != "*" || err != nil || n < 0 || strconv.Itoa(n) != b[i] {
			return false
		}
	}
	return true
}
func selectedOption(sources map[string]OptionSource, path string) (OptionSource, bool) {
	if source, ok := sources[path]; ok {
		return source, true
	}
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := wildcardCount(keys[i]), wildcardCount(keys[j])
		if a == b {
			return keys[i] < keys[j]
		}
		return a < b
	})
	for _, key := range keys {
		if matchFormPath(key, path) {
			return sources[key], true
		}
	}
	return OptionSource{}, false
}
func expandFormOptions(sources map[string]OptionSource, values any) map[string]OptionSource {
	result := map[string]OptionSource{}
	for pattern := range sources {
		parts := strings.Split(pattern, "/")
		var visit func(int, string)
		visit = func(index int, path string) {
			if index == len(parts) {
				source, _ := selectedOption(sources, path)
				result[path] = source
				return
			}
			if parts[index] != "*" {
				visit(index+1, path+"/"+parts[index])
				return
			}
			value, _ := p.Pointer(values, path)
			list, _ := value.([]any)
			for i := range list {
				visit(index+1, path+"/"+strconv.Itoa(i))
			}
		}
		visit(1, "")
	}
	return result
}
