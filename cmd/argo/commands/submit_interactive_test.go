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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient"
	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
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

// Fakes for the API clients used by --interactive. Each embeds its interface, so calling any
// method that isn't overridden panics and the test fails loudly.

type fakeWorkflowTemplateClient struct {
	workflowtemplatepkg.WorkflowTemplateServiceClient
	got *workflowtemplatepkg.WorkflowTemplateGetRequest
	err error
}

func (f *fakeWorkflowTemplateClient) GetWorkflowTemplate(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	f.got = req
	if f.err != nil {
		return nil, f.err
	}
	return &wfv1.WorkflowTemplate{Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "from-wftmpl"}}}}}, nil
}

type fakeClusterWorkflowTemplateClient struct {
	clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient
	got *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest
}

func (f *fakeClusterWorkflowTemplateClient) GetClusterWorkflowTemplate(_ context.Context, req *clusterworkflowtmplpkg.ClusterWorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	f.got = req
	return &wfv1.ClusterWorkflowTemplate{Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "from-cwftmpl"}}}}}, nil
}

type fakeCronWorkflowClient struct {
	cronworkflowpkg.CronWorkflowServiceClient
	got *cronworkflowpkg.GetCronWorkflowRequest
}

func (f *fakeCronWorkflowClient) GetCronWorkflow(_ context.Context, req *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	f.got = req
	return &wfv1.CronWorkflow{Spec: wfv1.CronWorkflowSpec{WorkflowSpec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "from-cronwf"}}}}}}, nil
}

type fakeWorkflowServiceClient struct {
	workflowpkg.WorkflowServiceClient
	got []*workflowpkg.ResolveParameterEnumRequest
}

func (f *fakeWorkflowServiceClient) ResolveParameterEnum(_ context.Context, req *workflowpkg.ResolveParameterEnumRequest, _ ...grpc.CallOption) (*workflowpkg.ResolveParameterEnumResponse, error) {
	// Record a copy: the caller keeps adding answers to the map it passed, as a real call would have serialized it.
	recorded := *req
	recorded.Parameters = maps.Clone(req.Parameters)
	f.got = append(f.got, &recorded)
	if req.ParameterName == "ns" {
		return &workflowpkg.ResolveParameterEnumResponse{Values: []string{"argo", "team-a"}}, nil
	}
	return &workflowpkg.ResolveParameterEnumResponse{Values: []string{"api-" + req.Parameters["ns"]}}, nil
}

type fakeAPIClient struct {
	apiclient.Client
	wftmpl  *fakeWorkflowTemplateClient
	cwftmpl *fakeClusterWorkflowTemplateClient
	cronwf  *fakeCronWorkflowClient
	err     error
}

func (f *fakeAPIClient) NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error) {
	return f.wftmpl, f.err
}

func (f *fakeAPIClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient, error) {
	return f.cwftmpl, f.err
}

func (f *fakeAPIClient) NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error) {
	return f.cronwf, f.err
}

func newFakeAPIClient() *fakeAPIClient {
	return &fakeAPIClient{wftmpl: &fakeWorkflowTemplateClient{}, cwftmpl: &fakeClusterWorkflowTemplateClient{}, cronwf: &fakeCronWorkflowClient{}}
}

func TestGetParametersForInteractiveSubmit(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	for _, kind := range []string{"WorkflowTemplate", "workflowtemplate", "workflowtemplates", "wftmpl"} {
		t.Run(kind, func(t *testing.T) {
			c := newFakeAPIClient()
			params, err := getParametersForInteractiveSubmit(ctx, c, "argo", kind, "my-tmpl")
			require.NoError(t, err)
			assert.Equal(t, "from-wftmpl", params[0].Name)
			assert.Equal(t, &workflowtemplatepkg.WorkflowTemplateGetRequest{Name: "my-tmpl", Namespace: "argo"}, c.wftmpl.got)
		})
	}
	t.Run("ClusterWorkflowTemplate", func(t *testing.T) {
		c := newFakeAPIClient()
		params, err := getParametersForInteractiveSubmit(ctx, c, "argo", "cwftmpl", "my-cwft")
		require.NoError(t, err)
		assert.Equal(t, "from-cwftmpl", params[0].Name)
		assert.Equal(t, "my-cwft", c.cwftmpl.got.Name)
	})
	t.Run("CronWorkflow", func(t *testing.T) {
		c := newFakeAPIClient()
		params, err := getParametersForInteractiveSubmit(ctx, c, "argo", "cronwf", "my-cron")
		require.NoError(t, err)
		assert.Equal(t, "from-cronwf", params[0].Name)
		assert.Equal(t, &cronworkflowpkg.GetCronWorkflowRequest{Name: "my-cron", Namespace: "argo"}, c.cronwf.got)
	})
	t.Run("UnsupportedKind", func(t *testing.T) {
		_, err := getParametersForInteractiveSubmit(ctx, newFakeAPIClient(), "argo", "pod", "x")
		require.EqualError(t, err, `--interactive does not support resource kind "pod"`)
	})
	for _, kind := range []string{"wftmpl", "cwftmpl", "cronwf"} {
		t.Run("ClientError/"+kind, func(t *testing.T) {
			c := newFakeAPIClient()
			c.err = errors.New("offline")
			_, err := getParametersForInteractiveSubmit(ctx, c, "argo", kind, "x")
			require.EqualError(t, err, "offline")
		})
	}
	t.Run("GetError", func(t *testing.T) {
		c := newFakeAPIClient()
		c.wftmpl.err = errors.New("not found")
		_, err := getParametersForInteractiveSubmit(ctx, c, "argo", "wftmpl", "x")
		require.EqualError(t, err, "not found")
	})
}

func TestAddInteractiveParameters(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := &wfv1.WorkflowTemplate{Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{
		enumFromParam("ns", nil),
		enumFromParam("deployment", nil),
		{Name: "given"},
	}}}}

	t.Run("AsksAndAppendsAnswers", func(t *testing.T) {
		svc := &fakeWorkflowServiceClient{}
		opts := &wfv1.SubmitOpts{Parameters: []string{"given=x"}}
		var out bytes.Buffer
		err := addInteractiveParameters(ctx, &templateAPIClient{fakeAPIClient: newFakeAPIClient(), tmpl: tmpl}, svc, "argo", "workflowtemplate/my-tmpl", opts, strings.NewReader("2\n1\n"), &out)
		require.NoError(t, err)
		assert.Equal(t, []string{"given=x", "ns=team-a", "deployment=api-team-a"}, opts.Parameters)
		require.Len(t, svc.got, 2)
		assert.Equal(t, &workflowpkg.ResolveParameterEnumRequest{
			Namespace: "argo", ResourceKind: "workflowtemplate", ResourceName: "my-tmpl", ParameterName: "deployment",
			Parameters: map[string]string{"given": "x", "ns": "team-a"},
		}, svc.got[1])
		assert.Contains(t, out.String(), "1) api-team-a")
	})
	t.Run("MalformedFrom", func(t *testing.T) {
		err := addInteractiveParameters(ctx, newFakeAPIClient(), &fakeWorkflowServiceClient{}, "argo", "no-slash", &wfv1.SubmitOpts{}, strings.NewReader(""), &bytes.Buffer{})
		require.ErrorContains(t, err, "resource identifier 'no-slash' is malformed")
	})
	t.Run("FetchError", func(t *testing.T) {
		c := newFakeAPIClient()
		c.wftmpl.err = errors.New("forbidden")
		err := addInteractiveParameters(ctx, c, &fakeWorkflowServiceClient{}, "argo", "wftmpl/x", &wfv1.SubmitOpts{}, strings.NewReader(""), &bytes.Buffer{})
		require.EqualError(t, err, "forbidden")
	})
	t.Run("PromptError", func(t *testing.T) {
		opts := &wfv1.SubmitOpts{}
		err := addInteractiveParameters(ctx, &templateAPIClient{fakeAPIClient: newFakeAPIClient(), tmpl: tmpl}, &fakeWorkflowServiceClient{}, "argo", "wftmpl/x", opts, strings.NewReader(""), &bytes.Buffer{})
		require.EqualError(t, err, `parameter "ns": no value given`)
		assert.Empty(t, opts.Parameters)
	})
}

// templateAPIClient serves a given WorkflowTemplate.
type templateAPIClient struct {
	*fakeAPIClient
	tmpl *wfv1.WorkflowTemplate
}

func (c *templateAPIClient) NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error) {
	return &staticWorkflowTemplateClient{tmpl: c.tmpl}, nil
}

type staticWorkflowTemplateClient struct {
	workflowtemplatepkg.WorkflowTemplateServiceClient
	tmpl *wfv1.WorkflowTemplate
}

func (s *staticWorkflowTemplateClient) GetWorkflowTemplate(context.Context, *workflowtemplatepkg.WorkflowTemplateGetRequest, ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	return s.tmpl, nil
}
