package parameterization

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/valyala/fasttemplate"
)

// ParameterReferences tracks native parameter references, not occurrences of
// parameter-like text. Task/step scopes track invocation references.
// "*" denotes explicit whole-object or dynamic-key access.
type ParameterReferences map[string]map[string]bool

func (r ParameterReferences) Uses(scope string, changed []string) bool {
	for _, name := range changed {
		if r[scope][name] || r[scope]["*"] {
			return true
		}
	}
	return false
}
func (r ParameterReferences) add(scope, name string) {
	if r[scope] == nil {
		r[scope] = map[string]bool{}
	}
	r[scope][name] = true
}

// References uses the same native tag delimiters and Expr grammar as Argo.
// Analysis never evaluates user expressions or fetches runtime data.
func References(value any) (ParameterReferences, error) {
	raw, documentErr := json.Marshal(value)
	if documentErr != nil {
		return nil, fmt.Errorf("cannot inspect parameter references")
	}
	var document any
	if documentErr = json.Unmarshal(raw, &document); documentErr != nil {
		return nil, documentErr
	}
	result := ParameterReferences{}
	var visit func(any) error
	visit = func(value any) error {
		switch v := value.(type) {
		case map[string]any:
			for key, item := range v {
				// Native substitution processes the serialized template, including
				// object keys (for example parameterized labels).
				if err := visit(key); err != nil {
					return err
				}
				if err := visit(item); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range v {
				if err := visit(item); err != nil {
					return err
				}
			}
		case string:
			tmpl, err := fasttemplate.NewTemplate(v, "{{", "}}")
			if err != nil {
				return fmt.Errorf("invalid native template tag in retry contract")
			}
			_, err = tmpl.ExecuteFunc(io.Discard, func(_ io.Writer, tag string) (int, error) {
				if after, ok := strings.CutPrefix(tag, "="); ok {
					tree, parseErr := parser.Parse(after)
					if parseErr != nil {
						return 0, fmt.Errorf("invalid native expression in retry contract")
					}
					visitor := &referenceVisitor{covered: map[ast.Node]bool{}, nodes: []ast.Node{}}
					ast.Walk(&tree.Node, visitor)
					if visitor.dynamicEnvironment {
						return 0, fmt.Errorf("dynamic environment access has no bounded retry dependency contract")
					}
					for _, node := range visitor.nodes {
						if visitor.covered[node] {
							continue
						}
						scope, name, ok := parameterAccess(node)
						if ok {
							result.add(scope, name)
						}
					}
				} else {
					parts := strings.Split(strings.TrimSpace(tag), ".")
					if len(parts) >= 2 && (parts[0] == "tasks" || parts[0] == "steps") {
						result.add(parts[0], parts[1])
					}
					if len(parts) >= 2 && (parts[0] == "workflow" || parts[0] == "inputs") && parts[1] == "parameters" {
						name := "*"
						if len(parts) == 3 && parts[2] != "json" {
							name = parts[2]
						}
						result.add(parts[0], name)
					}
				}
				return 0, nil
			})
			return err
		}
		return nil
	}
	if visitErr := visit(document); visitErr != nil {
		return nil, visitErr
	}
	return result, nil
}

type referenceVisitor struct {
	nodes              []ast.Node
	covered            map[ast.Node]bool
	dynamicEnvironment bool
}

func (v *referenceVisitor) Visit(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.MemberNode:
		v.covered[n.Node] = true
		v.nodes = append(v.nodes, n)
	case *ast.ChainNode:
		v.covered[n.Node] = true
		v.nodes = append(v.nodes, n)
	case *ast.IdentifierNode:
		v.nodes = append(v.nodes, n)
		if n.Value == "$env" {
			v.dynamicEnvironment = true
		}
	}
}

func parameterAccess(node ast.Node) (string, string, bool) {
	if chain, ok := node.(*ast.ChainNode); ok {
		return parameterAccess(chain.Node)
	}
	properties := []ast.Node{}
	for {
		if chain, ok := node.(*ast.ChainNode); ok {
			node = chain.Node
			continue
		}
		member, ok := node.(*ast.MemberNode)
		if !ok {
			break
		}
		properties = append([]ast.Node{member.Property}, properties...)
		node = member.Node
	}
	root, ok := node.(*ast.IdentifierNode)
	if !ok || (root.Value != "workflow" && root.Value != "inputs" && root.Value != "tasks" && root.Value != "steps") {
		return "", "", false
	}
	if len(properties) == 0 {
		return root.Value, "*", true
	}
	first, static := properties[0].(*ast.StringNode)
	if root.Value == "tasks" || root.Value == "steps" {
		if static {
			return root.Value, first.Value, true
		}
		return root.Value, "*", true
	}
	if static && first.Value != "parameters" {
		return "", "", false
	}
	if !static || len(properties) < 2 {
		return root.Value, "*", true
	}
	name, static := properties[1].(*ast.StringNode)
	if !static || name.Value == "json" {
		return root.Value, "*", true
	}
	return root.Value, name.Value, true
}
