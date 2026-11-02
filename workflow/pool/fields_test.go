package pool

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// fieldTableProblems lists what is wrong with table as the classification of
// struct type typ, recursing into nested tables: fields without an entry,
// entries without a field, and entries that can't do what they say.
func fieldTableProblems(typ reflect.Type, table fieldTable) []string {
	var problems []string
	for sf := range typ.Fields() {
		f, ok := table[sf.Name]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s.%s is not classified", typ, sf.Name))
			continue
		}
		switch {
		case f.fields != nil:
			ft := sf.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() != reflect.Struct {
				problems = append(problems, fmt.Sprintf("%s.%s has nested fields but is not a struct", typ, sf.Name))
				continue
			}
			problems = append(problems, fieldTableProblems(ft, f.fields)...)
		case f.class == SplitEnv && sf.Type != reflect.TypeFor[[]apiv1.EnvVar]():
			problems = append(problems, fmt.Sprintf("%s.%s is split but is not env", typ, sf.Name))
		case f.into != "" && !poolSpecHasField(f.into):
			problems = append(problems, fmt.Sprintf("%s.%s resolves into %s, which PoolSpec doesn't have", typ, sf.Name, f.into))
		case f.into != "" && f.class != Hashed:
			problems = append(problems, fmt.Sprintf("%s.%s resolves into %s but isn't hashed", typ, sf.Name, f.into))
		case f.class == 0:
			problems = append(problems, fmt.Sprintf("%s.%s has no class", typ, sf.Name))
		}
	}
	for name := range table {
		if _, ok := typ.FieldByName(name); !ok {
			problems = append(problems, fmt.Sprintf("%s has no field %s", typ, name))
		}
	}
	slices.Sort(problems)
	return problems
}

// poolSpecHasField reports whether a dotted Go field path exists in PoolSpec.
func poolSpecHasField(path string) bool {
	typ := reflect.TypeFor[wfv1.PoolSpec]()
	for name := range strings.SplitSeq(path, ".") {
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		sf, ok := typ.FieldByName(name)
		if !ok {
			return false
		}
		typ = sf.Type
	}
	return true
}

// TestFieldTablesCoverTypes checks that every field of Template, Container,
// ScriptTemplate and WorkflowSpec is classified, so a new field can't leak
// between the tasks of a pool unnoticed.
func TestFieldTablesCoverTypes(t *testing.T) {
	assert.Empty(t, fieldTableProblems(reflect.TypeFor[wfv1.Template](), templateFields))
	assert.Empty(t, fieldTableProblems(reflect.TypeFor[wfv1.WorkflowSpec](), workflowSpecFields))
	// The template table reaches these through container and script.
	assert.Empty(t, fieldTableProblems(reflect.TypeFor[apiv1.Container](), containerFields))
	assert.Empty(t, fieldTableProblems(reflect.TypeFor[wfv1.ScriptTemplate](), scriptFields))
}

// TestFieldTableProblemsFindsUnclassifiedFields checks the reflection test
// fails when a covered type gains a field, and that copying refuses
// one rather than dropping it silently.
func TestFieldTableProblemsFindsUnclassifiedFields(t *testing.T) {
	type inner struct{ A, B string }
	type covered struct {
		Known  string
		Nested *inner
		Added  string
	}
	table := fieldTable{"Known": hashed, "Nested": nested(fieldTable{"A": hashed}), "Gone": hashed}
	assert.Equal(t, []string{
		"pool.covered has no field Gone",
		"pool.covered.Added is not classified",
		"pool.inner.B is not classified",
	}, fieldTableProblems(reflect.TypeFor[covered](), table))

	src := covered{Known: "k"}
	err := copyFields("", onTemplate, table, reflect.ValueOf(src), reflect.ValueOf(&covered{}).Elem())
	assert.ErrorContains(t, err, "Added has no pool classification")
}

// TestLeaves checks Leaves walks into nested tables and carries each leaf's
// class and resolved field.
func TestLeaves(t *testing.T) {
	byPath := make(map[string]Leaf)
	for _, l := range Leaves() {
		byPath[l.String()] = l
	}
	assert.Equal(t, Rejected, byPath["Template.Script.Container.Lifecycle"].Class)
	assert.Equal(t, SplitEnv, byPath["Template.Container.Env"].Class)
	assert.Equal(t, Rejected, byPath["WorkflowSpec.RetryStrategy.Affinity"].Class)
	assert.Equal(t, Leaf{Path: []string{"WorkflowSpec", "Volumes"}, Class: Hashed, ResolvedInto: "WorkflowSpec.Volumes"}, byPath["WorkflowSpec.Volumes"])
	assert.NotContains(t, byPath, "Template.Container")
}

// TestCheck checks CheckTemplate and CheckWorkflowSpec name the first
// rejected field set under their path, and leave what they check alone.
func TestCheck(t *testing.T) {
	tmpl := &wfv1.Template{Container: &apiv1.Container{Image: "busybox", Env: []apiv1.EnvVar{{Name: "A", Value: "1"}}}}
	require.NoError(t, CheckTemplate("templates.t", tmpl))
	tmpl.Container.StartupProbe = &apiv1.Probe{InitialDelaySeconds: 1}
	before := tmpl.DeepCopy()
	require.EqualError(t, CheckTemplate("templates.t", tmpl), "templates.t.container.startupProbe can't be set on a pooled template")
	assert.Equal(t, before, tmpl)

	require.NoError(t, CheckWorkflowSpec(&wfv1.WorkflowSpec{Templates: []wfv1.Template{*tmpl}}))
	assert.EqualError(t, CheckWorkflowSpec(&wfv1.WorkflowSpec{PodSpecPatch: "{}"}), "spec.podSpecPatch can't be set on a workflow that runs a pooled template")
}
