package executor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStageLiteralsLiveInRegistry enforces that stages.go is the one place
// stages are declared, so "does this stage already exist" is answerable
// from a single file. Test files are exempt so they can build fakes.
func TestStageLiteralsLiveInRegistry(t *testing.T) {
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, file := range files {
		if file == "stages.go" || strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoError(t, err)
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if ident, ok := lit.Type.(*ast.Ident); ok && ident.Name == "Stage" {
				t.Errorf("%s: Stage literal outside stages.go; declare stages in the registry", fset.Position(lit.Pos()))
			}
			return true
		})
	}
}
