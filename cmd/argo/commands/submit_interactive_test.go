package commands

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func enumFromParam(name string, value *string) wfv1.Parameter {
	p := wfv1.Parameter{Name: name, EnumFrom: &wfv1.EnumFrom{Resource: wfv1.ResourceSelector{APIVersion: "v1", Kind: "Namespace"}, JSONPath: "{.metadata.name}"}}
	if value != nil {
		p.Value = wfv1.AnyStringPtr(*value)
	}
	return p
}

//go:fix inline

// fakeResolver serves options per parameter and records the values it was called with.
type fakeResolver struct {
	options map[string][]string
	errs    map[string]error
	calls   []map[string]string
}

func (f *fakeResolver) resolve(_ context.Context, name string, values map[string]string) (*workflowpkg.ResolveParameterEnumResponse, error) {
	copied := map[string]string{}
	maps.Copy(copied, values)
	f.calls = append(f.calls, copied)
	if err := f.errs[name]; err != nil {
		return nil, err
	}
	return &workflowpkg.ResolveParameterEnumResponse{Values: f.options[name], Truncated: name == "big"}, nil
}

func prompt(t *testing.T, params []wfv1.Parameter, given []string, input string, r *fakeResolver) ([]string, string, error) {
	t.Helper()
	if r == nil {
		r = &fakeResolver{}
	}
	var out bytes.Buffer
	answers, err := promptForParameters(logging.TestContext(t.Context()), strings.NewReader(input), &out, params, given, r.resolve)
	return answers, out.String(), err
}

func TestPromptForParameters(t *testing.T) {
	nsOptions := &fakeResolver{options: map[string][]string{"ns": {"argo", "default", "team-a"}}}

	t.Run("ChooseByNumber", func(t *testing.T) {
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "3\n", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=team-a"}, answers)
		assert.Contains(t, out, "  1) argo\n  2) default\n  3) team-a\n")
	})
	t.Run("ChooseByName", func(t *testing.T) {
		answers, _, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "default\n", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=default"}, answers)
	})
	t.Run("EnterKeepsDefault", func(t *testing.T) {
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", new("argo"))}, nil, "\n", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=argo"}, answers)
		assert.Contains(t, out, "choose 1-3 [argo]: ")
	})
	t.Run("InvalidChoiceAsksAgain", func(t *testing.T) {
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "9\nnope\n2\n", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=default"}, answers)
		assert.Equal(t, 2, strings.Count(out, "enter a number from 1 to 3"))
	})
	t.Run("EmptyPlaceholderIsNotAChoice", func(t *testing.T) {
		answers, _, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", new(""))}, nil, "\n1\n", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=argo"}, answers)
	})
	t.Run("GivenParametersAreNotAskedFor", func(t *testing.T) {
		r := &fakeResolver{options: nsOptions.options}
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, []string{"ns=team-a"}, "", r)
		require.NoError(t, err)
		assert.Empty(t, answers)
		assert.Empty(t, out)
		assert.Empty(t, r.calls)
	})
	t.Run("CascadingUsesEarlierAnswers", func(t *testing.T) {
		r := &fakeResolver{options: map[string][]string{"ns": {"argo", "team-a"}, "deployment": {"api"}}}
		answers, _, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil), {Name: "tier", Value: wfv1.AnyStringPtr("web")}, enumFromParam("deployment", nil)}, []string{"other=x"}, "2\n\n1\n", r)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=team-a", "tier=web", "deployment=api"}, answers)
		require.Len(t, r.calls, 2)
		assert.Equal(t, map[string]string{"tier": "web", "other": "x"}, r.calls[0])
		assert.Equal(t, map[string]string{"ns": "team-a", "tier": "web", "other": "x"}, r.calls[1])
	})
	t.Run("ResolveErrorFallsBackToText", func(t *testing.T) {
		r := &fakeResolver{errs: map[string]error{"ns": errors.New("forbidden")}}
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "typed-ns\n", r)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=typed-ns"}, answers)
		assert.Contains(t, out, "could not load options, type a value instead: forbidden")
	})
	t.Run("ResolveErrorShowsServerMessage", func(t *testing.T) {
		r := &fakeResolver{errs: map[string]error{"ns": status.Error(codes.PermissionDenied, "enumFrom cannot list Secrets")}}
		_, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "x\n", r)
		require.NoError(t, err)
		assert.Contains(t, out, "type a value instead: enumFrom cannot list Secrets\n")
	})
	t.Run("NoOptionsFallsBackToText", func(t *testing.T) {
		answers, out, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "typed-ns\n", &fakeResolver{})
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=typed-ns"}, answers)
		assert.Contains(t, out, "no options found")
	})
	t.Run("TruncatedIsShown", func(t *testing.T) {
		r := &fakeResolver{options: map[string][]string{"big": {"a", "b"}}}
		_, out, err := prompt(t, []wfv1.Parameter{enumFromParam("big", nil)}, nil, "1\n", r)
		require.NoError(t, err)
		assert.Contains(t, out, "showing the first 2 options only")
	})
	t.Run("StaticEnum", func(t *testing.T) {
		p := wfv1.Parameter{Name: "approve", Value: wfv1.AnyStringPtr("NO"), Enum: []wfv1.AnyString{wfv1.AnyString("YES"), wfv1.AnyString("NO")}}
		answers, _, err := prompt(t, []wfv1.Parameter{p}, nil, "1\n", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"approve=YES"}, answers)
	})
	t.Run("PlainTextWithDefaultAndDescription", func(t *testing.T) {
		p := wfv1.Parameter{Name: "message", Default: wfv1.AnyStringPtr("hello"), Description: wfv1.AnyStringPtr("what to say")}
		answers, out, err := prompt(t, []wfv1.Parameter{p}, nil, "\n", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"message=hello"}, answers)
		assert.Contains(t, out, "message - what to say\n  value [hello]: ")
	})
	t.Run("PlainTextEmptyAnswer", func(t *testing.T) {
		answers, _, err := prompt(t, []wfv1.Parameter{{Name: "message"}}, nil, "\n", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"message="}, answers)
	})
	t.Run("LastLineWithoutNewline", func(t *testing.T) {
		answers, _, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "2", nsOptions)
		require.NoError(t, err)
		assert.Equal(t, []string{"ns=default"}, answers)
	})
	t.Run("EndOfInputWithoutDefault", func(t *testing.T) {
		_, _, err := prompt(t, []wfv1.Parameter{{Name: "message"}}, nil, "", nil)
		require.EqualError(t, err, `parameter "message": no value given`)
	})
	t.Run("EndOfInputWithDefault", func(t *testing.T) {
		answers, _, err := prompt(t, []wfv1.Parameter{{Name: "message", Value: wfv1.AnyStringPtr("hi")}}, nil, "", nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"message=hi"}, answers)
	})
	t.Run("EndOfInputAfterInvalidChoice", func(t *testing.T) {
		_, _, err := prompt(t, []wfv1.Parameter{enumFromParam("ns", nil)}, nil, "nope", nsOptions)
		require.EqualError(t, err, `parameter "ns": "nope" is not one of the options`)
	})
}

func TestSubmitInteractiveRequiresFrom(t *testing.T) {
	cmd := NewSubmitCommand()
	require.NoError(t, cmd.Flags().Set("interactive", "true"))
	err := cmd.Args(cmd, []string{"my-wf.yaml"})
	require.EqualError(t, err, "--interactive requires --from, because options are resolved from the stored resource")
}
