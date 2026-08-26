package dag

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/template"
)

func intstrPtr(s string) *intstr.IntOrString {
	v := intstr.FromString(s)
	return &v
}

// TestExpandSequence_BackwardCounting verifies that withSequence with start > end
// produces items counting backwards (e.g., start=5, end=1 → [5,4,3,2,1]).
// The old code in operator.go explicitly handled this with a reverse loop.
func TestExpandSequence_BackwardCounting(t *testing.T) {
	seq := &wfv1.Sequence{
		Start: intstrPtr("5"),
		End:   intstrPtr("1"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.NotNil(t, items, "withSequence start=5 end=1 should produce items, not nil")
	require.Len(t, items, 5, "should produce 5 items counting from 5 down to 1")

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"5"`, `"4"`, `"3"`, `"2"`, `"1"`}, values)
}

// TestExpandSequence_BackwardCountingWithFormat verifies backward counting
// with a custom format string.
func TestExpandSequence_BackwardCountingWithFormat(t *testing.T) {
	seq := &wfv1.Sequence{
		Start:  intstrPtr("3"),
		End:    intstrPtr("0"),
		Format: "item-%02d",
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 4)

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"item-03"`, `"item-02"`, `"item-01"`, `"item-00"`}, values)
}

// TestExpandSequence_ProducesStringItems verifies that withSequence without a format
// string produces JSON string items (e.g., "0", "1", "2"), not JSON numbers (0, 1, 2).
// The old code always used ParseItem(`"` + fmt.Sprintf("%d", i) + `"`), which wraps
// the value in quotes, producing string items.
func TestExpandSequence_ProducesStringItems(t *testing.T) {
	seq := &wfv1.Sequence{
		Count: intstrPtr("3"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 3)

	for i, item := range items {
		var parsed any
		err := json.Unmarshal(item.Value, &parsed)
		require.NoError(t, err)

		// The value must be a JSON string, not a number.
		// Old behavior: ParseItem(`"0"`) → JSON string "0"
		// Bug behavior: json.Marshal(0) → JSON number 0
		strVal, ok := parsed.(string)
		assert.True(t, ok, "item %d should be a JSON string, got %T: %s", i, parsed, string(item.Value))
		assert.Equal(t, string(rune('0'+i)), strVal)
	}
}

// TestExpandSequence_ForwardCounting confirms forward counting still works.
func TestExpandSequence_ForwardCounting(t *testing.T) {
	seq := &wfv1.Sequence{
		Start: intstrPtr("0"),
		End:   intstrPtr("2"),
	}

	items, err := expandSequence(seq)
	require.NoError(t, err)
	require.Len(t, items, 3)

	var values []string
	for _, item := range items {
		values = append(values, string(item.Value))
	}
	assert.Equal(t, []string{`"0"`, `"1"`, `"2"`}, values)
}

func TestExpandSequence_CountAndRange(t *testing.T) {
	strVals := func(items []wfv1.Item) []string {
		out := make([]string, len(items))
		for i, item := range items {
			out[i] = item.GetStrVal()
		}
		return out
	}

	items, err := expandSequence(&wfv1.Sequence{Count: intstrPtr("10")})
	require.NoError(t, err)
	require.Len(t, items, 10)
	assert.Equal(t, "0", strVals(items)[0])
	assert.Equal(t, "9", strVals(items)[9])

	items, err = expandSequence(&wfv1.Sequence{Start: intstrPtr("101"), Count: intstrPtr("10")})
	require.NoError(t, err)
	require.Len(t, items, 10)
	assert.Equal(t, "101", strVals(items)[0])
	assert.Equal(t, "110", strVals(items)[9])

	items, err = expandSequence(&wfv1.Sequence{Start: intstrPtr("50"), End: intstrPtr("60")})
	require.NoError(t, err)
	require.Len(t, items, 11)
	assert.Equal(t, "50", strVals(items)[0])
	assert.Equal(t, "60", strVals(items)[10])

	items, err = expandSequence(&wfv1.Sequence{Start: intstrPtr("60"), End: intstrPtr("50")})
	require.NoError(t, err)
	require.Len(t, items, 11)
	assert.Equal(t, "60", strVals(items)[0])
	assert.Equal(t, "50", strVals(items)[10])

	items, err = expandSequence(&wfv1.Sequence{Count: intstrPtr("0")})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestExpandedTaskName(t *testing.T) {
	name, err := expandedTaskName("sleep", 10, "ten")
	require.NoError(t, err)
	assert.Equal(t, "sleep(10:ten)", name)

	// Parentheses in the item text are stripped so the index stays recoverable.
	name, err = expandedTaskName("sleep", 3, "(a)")
	require.NoError(t, err)
	assert.Equal(t, "sleep(3:a)", name)
	assert.Equal(t, 3, util.RecoverIndexFromNodeName(name))

	_, err = expandedTaskName("bad(name", 1, "x")
	require.Error(t, err)
}

// templateSubstitutor substitutes {{...}} tags with the argo template engine,
// as the controller's wfOperationCtx does.
type templateSubstitutor struct{}

func (templateSubstitutor) Substitute(s string, scope map[string]string) (string, error) {
	tmpl, err := template.NewTemplate(s)
	if err != nil {
		return "", err
	}
	replaceMap := make(map[string]any, len(scope))
	for k, v := range scope {
		replaceMap[k] = v
	}
	return tmpl.Replace(context.Background(), replaceMap, true)
}

// Expanded task names and substituted {{item}} values for each item shape,
// as the pre-Engine controller produced them.
func TestProcessItem_ItemShapes(t *testing.T) {
	tests := []struct {
		name          string
		withParam     string
		expectedName  string
		expectedParam string
	}{
		{"number", `[42]`, `task-name(0:42)`, `42`},
		{"boolean", `[true]`, `task-name(0:true)`, `true`},
		{"map", `[{"number": 2, "string": "foo", "list": [0, "1"], "json": {"number": 2, "string": "foo", "list": [0, "1"]}}]`,
			`task-name(0:json:{"list":[0,"1"],"number":2,"string":"foo"},list:[0,"1"],number:2,string:foo)`,
			`{"json":{"list":[0,"1"],"number":2,"string":"foo"},"list":[0,"1"],"number":2,"string":"foo"}`},
		{"list", `[[1, "two", 3]]`, `task-name(0:[1 two 3])`, `[1,"two",3]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := wfv1.DAGTask{
				Name:      "task-name",
				Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "item", Value: wfv1.AnyStringPtr("{{item}}")}}},
			}
			taskBytes, err := json.Marshal(task)
			require.NoError(t, err)
			var items []wfv1.Item
			wfv1.MustUnmarshal([]byte(tt.withParam), &items)

			var newTask wfv1.DAGTask
			newTaskName, err := processItem(context.Background(), taskBytes, task.Name, 0, items[0], &newTask, nil, templateSubstitutor{})
			require.NoError(t, err)
			assert.Equal(t, tt.expectedName, newTaskName)
			assert.Equal(t, tt.expectedParam, newTask.Arguments.Parameters[0].Value.String())
		})
	}
}
