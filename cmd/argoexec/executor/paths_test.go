package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// The command's varRunArgo must be the one shared directory a pod task
// uses, so a test that points it elsewhere moves every path with it.
func TestPodPaths_UsesGivenVarRunArgo(t *testing.T) {
	dir := t.TempDir()
	want := executor.PodPaths()
	want.VarRunArgo = dir
	assert.Equal(t, want, PodPaths(dir))
}
