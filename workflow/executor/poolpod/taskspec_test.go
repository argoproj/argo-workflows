package poolpod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// newTask creates a task directory named name.
func newTask(t *testing.T, name string) Task {
	t.Helper()
	task := NewTask(filepath.Join(t.TempDir(), name))
	require.NoError(t, task.Create())
	return task
}

func TestTask_SpecRoundTrips(t *testing.T) {
	task := newTask(t, "a")
	spec := TaskSpec{
		NodeID:              "n1",
		Template:            &wfv1.Template{Name: "t", Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: "/p"}}}}},
		Command:             []string{"sh", "-c"},
		Args:                []string{"echo $(A)"},
		Env:                 []EnvVar{{Name: "A", Value: "a"}},
		IncludeScriptOutput: true,
	}
	require.NoError(t, task.WriteSpec(spec))
	got, err := task.Spec()
	require.NoError(t, err)
	assert.Equal(t, spec, got)
}

// main can't run a task without these, so neither side accepts a spec
// that lacks them.
func TestTask_SpecRefusesAnIncompleteTask(t *testing.T) {
	task := newTask(t, "a")
	tmpl := &wfv1.Template{}
	for name, spec := range map[string]TaskSpec{
		"no node ID":  {Template: tmpl, Command: []string{"true"}},
		"no template": {NodeID: "n", Command: []string{"true"}},
		"no command":  {NodeID: "n", Template: tmpl},
	} {
		require.Error(t, task.WriteSpec(spec), name)
	}

	require.NoError(t, os.WriteFile(filepath.Join(task.Dir(), TaskFile), []byte(`{"nodeID":"n"}`), 0o644))
	_, err := task.Spec()
	require.Error(t, err)
}

func TestTask_SpecMissing(t *testing.T) {
	_, err := newTask(t, "a").Spec()
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCheckTaskName(t *testing.T) {
	require.NoError(t, CheckTaskName("wf-abc-123"))
	for _, name := range []string{"", ".", "..", ".current-1", "a/b", `a\b`, "/a", CurrentFile, MainFile} {
		assert.Error(t, CheckTaskName(name), "%q", name)
	}
}

func TestExpand_AsTheKubeletDoes(t *testing.T) {
	vars := map[string]string{"A": "a", "B": "b", "EMPTY": ""}
	for input, want := range map[string]string{
		"plain":           "plain",
		"$(A)":            "a",
		"x$(A)y$(B)z":     "xaybz",
		"$(EMPTY)":        "",
		"$(UNDEFINED)":    "$(UNDEFINED)",
		"$$(A)":           "$(A)",
		"$$$(A)":          "$a",
		"$$":              "$",
		"$A":              "$A",
		"$":               "$",
		"a$":              "a$",
		"$(A":             "$(A",
		"$($(A))":         "$($(A))",
		"$()":             "$()",
		"$(A)$(A)":        "aa",
		"echo $(date) $$": "echo $(date) $",
	} {
		assert.Equal(t, want, expand(input, vars), "%q", input)
	}
}

func TestResolve(t *testing.T) {
	spec := TaskSpec{
		Command: []string{"$(SHELL_NAME)", "-c"},
		Args:    []string{"echo $(GREETING) $(UNDEFINED) $$(GREETING)", "/argo/staging/t/script"},
		Env: []EnvVar{
			{Name: "GREETING", Value: "hello $(WHO)"},
			{Name: "WHO", Value: "task"},
			{Name: "SHELL_NAME", Value: "sh"},
		},
	}
	argv, env := spec.Resolve([]string{"WHO=main", "FROM_POOL=yes"})
	assert.Equal(t, []string{"sh", "-c", "echo hello main $(UNDEFINED) $(GREETING)", "/argo/staging/t/script"}, argv,
		"each env value sees only main's env and the entries before it; the command and args see all of it")
	assert.Equal(t, []string{"WHO=main", "FROM_POOL=yes", "GREETING=hello main", "WHO=task", "SHELL_NAME=sh"}, env,
		"the task's entries follow main's, so the last value of a name wins")
}

// The kubelet doesn't expand an image's entrypoint or cmd, only what the
// pod spec says.
func TestResolve_ImageCommandIsNotExpanded(t *testing.T) {
	spec := TaskSpec{
		ImageCommand: []string{"sh", "-c", "echo $(date) $$"},
		Args:         []string{"$(A)"},
		Env:          []EnvVar{{Name: "A", Value: "a"}},
	}
	argv, _ := spec.Resolve(nil)
	assert.Equal(t, []string{"sh", "-c", "echo $(date) $$", "a"}, argv)
}

func TestResolve_EnvIsNeverNil(t *testing.T) {
	_, env := TaskSpec{Command: []string{"true"}}.Resolve(nil)
	assert.NotNil(t, env, "a nil env would make the command inherit main's whole environment")
}
