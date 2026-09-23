package controller

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Execute's steps hand each other typed values so that the order they run in
// is checked by the compiler. That only holds if each value is built by the
// step that returns it: a literal anywhere else would let a step run without
// the one before it. Go cannot restrict construction within a package, so
// check the source instead.
func TestEnginePassValuesAreBuiltByTheirStep(t *testing.T) {
	builtBy := map[string][]string{
		"hooksRun":           {"processHooks"},
		"evaluation":         {"evaluateAll"},
		"omissionsRecorded":  {"createOmittedNodes"},
		"dispatched":         {"converge"},
		"taskGroupsAssessed": {"assessTaskGroups", "taskGroupsFromEarlierCycles"},
	}

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	fset := token.NewFileSet()
	found := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				ident, ok := lit.Type.(*ast.Ident)
				if !ok {
					return true
				}
				allowed, isPassValue := builtBy[ident.Name]
				if !isPassValue {
					return true
				}
				found[ident.Name] = true
				assert.Contains(t, allowed, fn.Name.Name, "%s is built in %s at %s", ident.Name, fn.Name.Name, fset.Position(lit.Pos()))
				return true
			})
		}
	}
	for typeName := range builtBy {
		assert.True(t, found[typeName], "no construction of %s found; the check has gone stale", typeName)
	}
}
