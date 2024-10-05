package lint

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/lint/mocks"
	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wftemplatemocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate/mocks"
	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var lintFileData = []byte(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello
  templates:
  - name: hello
    steps:
    - - name: hello
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello1"}]

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]

---
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: foo
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`)

func TestLintFile(t *testing.T) {
	file, err := os.CreateTemp("", "*.yaml")
	require.NoError(t, err)
	err = os.WriteFile(file.Name(), lintFileData, 0o600)
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fmtr, err := GetFormatter("simple")
	require.NoError(t, err)

	wfServiceClientMock := &workflowmocks.WorkflowServiceClient{}
	wftServiceSclientMock := &wftemplatemocks.WorkflowTemplateServiceClient{}
	wfServiceClientMock.On("LintWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))

	res, err := Lint(context.Background(), &LintOptions{
		Files: []string{file.Name()},
		ServiceClients: ServiceClients{
			WorkflowsClient: wfServiceClientMock,
		},
		Formatter: fmtr,
	})

	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.msg, fmt.Sprintf(`%s: in "steps-" (Workflow): lint error`, file.Name()))
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNotCalled(t, "LintWorkflowTemplate")
}

func TestLintMultipleKinds(t *testing.T) {
	file, err := os.CreateTemp("", "*.yaml")
	require.NoError(t, err)
	err = os.WriteFile(file.Name(), lintFileData, 0o600)
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fmtr, err := GetFormatter("simple")
	require.NoError(t, err)

	wfServiceClientMock := &workflowmocks.WorkflowServiceClient{}
	wftServiceSclientMock := &wftemplatemocks.WorkflowTemplateServiceClient{}
	wfServiceClientMock.On("LintWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))
	wftServiceSclientMock.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))

	res, err := Lint(context.Background(), &LintOptions{
		Files: []string{file.Name()},
		ServiceClients: ServiceClients{
			WorkflowsClient:         wfServiceClientMock,
			WorkflowTemplatesClient: wftServiceSclientMock,
		},
		Formatter: fmtr,
	})

	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.msg, fmt.Sprintf(`%s: in "steps-" (Workflow): lint error`, file.Name()))
	assert.Contains(t, res.msg, fmt.Sprintf(`%s: in "foo" (WorkflowTemplate): lint error`, file.Name()))
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNumberOfCalls(t, "LintWorkflowTemplate", 1)
}

func TestLintWithOutput(t *testing.T) {
	file, err := os.CreateTemp("", "*.yaml")
	require.NoError(t, err)
	err = os.WriteFile(file.Name(), lintFileData, 0o600)
	require.NoError(t, err)
	defer os.Remove(file.Name())

	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.Write(lintFileData)
	require.NoError(t, err)
	w.Close()
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	fmtr, err := GetFormatter("simple")
	require.NoError(t, err)

	wfServiceClientMock := &workflowmocks.WorkflowServiceClient{}
	wftServiceSclientMock := &wftemplatemocks.WorkflowTemplateServiceClient{}
	wfServiceClientMock.On("LintWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))
	wftServiceSclientMock.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))

	mw := &mocks.MockWriter{}
	mw.On("Write", mock.Anything).Return(0, nil)

	res, err := Lint(context.Background(), &LintOptions{
		Files: []string{file.Name(), "-"},
		ServiceClients: ServiceClients{
			WorkflowsClient:         wfServiceClientMock,
			WorkflowTemplatesClient: wftServiceSclientMock,
		},
		Formatter: fmtr,
		Printer:   mw,
	})

	expected := []string{
		fmt.Sprintf("%s: in \"steps-\" (Workflow): lint error\n%s: in \"foo\" (WorkflowTemplate): lint error\n", file.Name(), file.Name()),
		"stdin: in \"steps-\" (Workflow): lint error\nstdin: in \"foo\" (WorkflowTemplate): lint error\n",
		"",
	}
	mw.AssertCalled(t, "Write", []byte(expected[0]))
	mw.AssertCalled(t, "Write", []byte(expected[1]))
	mw.AssertCalled(t, "Write", []byte(expected[2]))
	require.NoError(t, err)
	assert.False(t, res.Success)
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 2)
	wftServiceSclientMock.AssertNumberOfCalls(t, "LintWorkflowTemplate", 2)
	assert.Equal(t, strings.Join(expected, ""), res.Msg())
}

func TestLintStdin(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.Write(lintFileData)
	require.NoError(t, err)
	w.Close()
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	fmtr, err := GetFormatter("simple")
	require.NoError(t, err)

	wfServiceClientMock := &workflowmocks.WorkflowServiceClient{}
	wftServiceSclientMock := &wftemplatemocks.WorkflowTemplateServiceClient{}
	wfServiceClientMock.On("LintWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))
	wftServiceSclientMock.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))

	res, err := Lint(context.Background(), &LintOptions{
		Files: []string{"-"},
		ServiceClients: ServiceClients{
			WorkflowsClient:         wfServiceClientMock,
			WorkflowTemplatesClient: wftServiceSclientMock,
		},
		Formatter: fmtr,
	})

	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.msg, `stdin: in "steps-" (Workflow): lint error`)
	assert.Contains(t, res.msg, `stdin: in "foo" (WorkflowTemplate): lint error`)
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNumberOfCalls(t, "LintWorkflowTemplate", 1)
}

func TestLintDeviceFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("device files not accessible in windows")
	}

	file, err := os.CreateTemp("", "*.yaml")
	fd := file.Fd()
	require.NoError(t, err)
	err = os.WriteFile(file.Name(), lintFileData, 0o600)
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fmtr, err := GetFormatter("simple")
	require.NoError(t, err)

	wfServiceClientMock := &workflowmocks.WorkflowServiceClient{}
	wftServiceSclientMock := &wftemplatemocks.WorkflowTemplateServiceClient{}
	wfServiceClientMock.On("LintWorkflow", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("lint error"))

	deviceFileName := fmt.Sprintf("/dev/fd/%d", fd)

	res, err := Lint(context.Background(), &LintOptions{
		Files: []string{deviceFileName},
		ServiceClients: ServiceClients{
			WorkflowsClient: wfServiceClientMock,
		},
		Formatter: fmtr,
	})

	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.msg, fmt.Sprintf(`%s: in "steps-" (Workflow): lint error`, deviceFileName))
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNotCalled(t, "LintWorkflowTemplate")
}

func TestGetFormatter(t *testing.T) {
	tests := map[string]struct {
		formatterName  string
		expectedErr    error
		expectedOutput string
	}{
		"default": {
			formatterName:  "",
			expectedErr:    nil,
			expectedOutput: (&LintResults{fmtr: formatterPretty{}}).buildMsg(),
		},
		"pretty": {
			formatterName:  "pretty",
			expectedErr:    nil,
			expectedOutput: (&LintResults{fmtr: formatterPretty{}}).buildMsg(),
		},
		"simple": {
			formatterName:  "simple",
			expectedErr:    nil,
			expectedOutput: (&LintResults{fmtr: formatterSimple{}}).buildMsg(),
		},
		"unknown name": {
			formatterName:  "foo",
			expectedErr:    fmt.Errorf("unknown formatter: foo"),
			expectedOutput: "",
		},
	}

	for tname, test := range tests {
		t.Run(tname, func(t *testing.T) {
			var (
				fmtr Formatter
				err  error
			)

			if test.formatterName != "" {
				fmtr, err = GetFormatter(test.formatterName)
				if test.expectedErr != nil {
					require.EqualError(t, err, test.expectedErr.Error())
					return
				} else {
					require.NoError(t, err)
				}
			}

			r, err := Lint(context.Background(), &LintOptions{Formatter: fmtr})
			require.NoError(t, err)
			assert.Equal(t, test.expectedOutput, r.Msg())
		})
	}
}

func TestGetObjectName(t *testing.T) {
	tests := map[string]struct {
		kind     string
		obj      metav1.Object
		objIndex int
		expected string
	}{
		"WithName": {
			kind: wf.WorkflowTemplateKind,
			obj: &v1alpha1.WorkflowTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			objIndex: 0,
			expected: `"foo" (WorkflowTemplate)`,
		},
		"WithGenerateName": {
			kind: wf.WorkflowKind,
			obj: &v1alpha1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "foo-",
				},
			},
			objIndex: 0,
			expected: `"foo-" (Workflow)`,
		},
		"NoName": {
			kind:     wf.CronWorkflowKind,
			obj:      &v1alpha1.CronWorkflow{},
			objIndex: 2,
			expected: `"object #3" (CronWorkflow)`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, getObjectName(test.kind, test.obj, test.objIndex))
		})
	}
}
