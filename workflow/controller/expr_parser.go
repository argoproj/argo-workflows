package controller

import (
	"strings"

	"io"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/valyala/fasttemplate"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type outputRefVisitor struct {
	targetName string
	found      bool
}

func (v *outputRefVisitor) Visit(node *ast.Node) {
	if v.found {
		return
	}
	// We are looking for something that ends in .outputs.result where the part before .outputs is our targetName.
	// Example: steps.flip-coin.outputs.result
	// Example: tasks['A'].outputs.result

	if m, ok := (*node).(*ast.MemberNode); ok {
		// Check for .result
		if prop, ok := m.Property.(*ast.StringNode); ok && prop.Value == "result" {
			// Check parent is .outputs
			if outputs, ok := m.Node.(*ast.MemberNode); ok {
				if outProp, ok := outputs.Property.(*ast.StringNode); ok && outProp.Value == "outputs" {
					// Check parent of .outputs is targetName
					// This covers both .targetName and ['targetName']
					if target, ok := outputs.Node.(*ast.MemberNode); ok {
						if targetProp, ok := target.Property.(*ast.StringNode); ok && targetProp.Value == v.targetName {
							v.found = true
							return
						}
					}
					// Also check if the node itself is the targetName (Identifier)
					if target, ok := outputs.Node.(*ast.IdentifierNode); ok {
						if target.Value == v.targetName {
							v.found = true
							return
						}
					}
				}
			}
		}
	}
}

func HasOutputRef(expression string, targetName string) bool {
	// Simple string check for common variable patterns to handle hyphenated names correctly
	// that might be misinterpreted by the expr parser (e.g. flip-coin as flip minus coin).
	patterns := []string{
		"." + targetName + ".outputs.result",
		"['" + targetName + "'].outputs.result",
		"[\"" + targetName + "\"].outputs.result",
	}
	for _, p := range patterns {
		if strings.Contains(expression, p) {
			return true
		}
	}
	if strings.HasPrefix(expression, targetName+".outputs.result") {
		return true
	}

	tree, err := parser.Parse(expression)
	if err != nil {
		return false
	}

	visitor := &outputRefVisitor{targetName: targetName}
	ast.Walk(&tree.Node, visitor)
	return visitor.found
}

func hasOutputRefTemplate(text string, targetName string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	// Check for {{}}
	t, err := fasttemplate.NewTemplate(text, "{{", "}}")
	if err != nil {
		// Treat as plain expression? No, fasttemplate error usually means unmatched braces.
		// If it's not a template, maybe it's a raw string.
		return false
	}

	found := false
	_, _ = t.ExecuteFunc(io.Discard, func(w io.Writer, tag string) (int, error) {
		// Remove '=' prefix for expressions
		tag = strings.TrimPrefix(tag, "=")
		if HasOutputRef(tag, targetName) {
			found = true
			return 0, io.EOF // Stop
		}
		return 0, nil
	})
	return found
}

func TraverseTemplateForOutputRef(tmpl *wfv1.Template, targetName string) bool {
	// Check Inputs? Inputs usually define params, not use them (except defaults).
	// Default values CAN use references? Yes.
	for _, p := range tmpl.Inputs.Parameters {
		if p.Value != nil && hasOutputRefTemplate(p.Value.String(), targetName) {
			return true
		}
		if p.Default != nil && hasOutputRefTemplate(p.Default.String(), targetName) {
			return true
		}
	}

	// Arguments in DAG/Steps/Hooks
	if tmpl.DAG != nil {
		for _, task := range tmpl.DAG.Tasks {
			if checkArguments(task.Arguments, targetName) {
				return true
			}
			if checkHooks(task.Hooks, targetName) {
				return true
			}
			if task.When != "" && HasOutputRef(task.When, targetName) {
				return true
			}
		}
	}
	if tmpl.Steps != nil {
		for _, group := range tmpl.Steps {
			for _, step := range group.Steps {
				if checkArguments(step.Arguments, targetName) {
					return true
				}
				if checkHooks(step.Hooks, targetName) {
					return true
				}
				if step.When != "" && HasOutputRef(step.When, targetName) {
					return true
				}
			}
		}
	}

	// Template Outputs (e.g. valueFrom.expression referencing step/task outputs)
	for _, p := range tmpl.Outputs.Parameters {
		if p.ValueFrom != nil && p.ValueFrom.Expression != "" {
			if HasOutputRef(p.ValueFrom.Expression, targetName) {
				return true
			}
		}
		if p.Value != nil && hasOutputRefTemplate(p.Value.String(), targetName) {
			return true
		}
	}

	return false
}

func checkArguments(args wfv1.Arguments, targetName string) bool {
	for _, p := range args.Parameters {
		if p.Value != nil && hasOutputRefTemplate(p.Value.String(), targetName) {
			return true
		}
		if p.ValueFrom != nil && p.ValueFrom.Expression != "" {
			if HasOutputRef(p.ValueFrom.Expression, targetName) {
				return true
			}
		}
	}
	for _, a := range args.Artifacts {
		if a.From != "" && hasOutputRefTemplate(a.From, targetName) {
			return true
		}
		if a.FromExpression != "" && HasOutputRef(a.FromExpression, targetName) {
			return true
		}
	}
	return false
}

func checkHooks(hooks wfv1.LifecycleHooks, targetName string) bool {
	for _, hook := range hooks {
		if checkArguments(hook.Arguments, targetName) {
			return true
		}
		if hook.Expression != "" && HasOutputRef(hook.Expression, targetName) {
			return true
		}
	}
	return false
}
