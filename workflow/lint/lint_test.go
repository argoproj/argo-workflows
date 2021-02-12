package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	workflowmocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow/mocks"
	wftemplatemocks "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate/mocks"
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
  name: workflow-template-submittable
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
	file, err := ioutil.TempFile("", "*.yaml")
	assert.NoError(t, err)
	err = ioutil.WriteFile(file.Name(), lintFileData, 0644)
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	fmtr, err := GetFormatter("simple")
	assert.NoError(t, err)

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

	assert.NoError(t, err)
	assert.Equal(t, res.Success, false)
	assert.Equal(t, res.msg, fmt.Sprintf("%s: in object #1: lint error\n", file.Name()))
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNotCalled(t, "LintWorkflowTemplate")
}

func TestLintFile2(t *testing.T) {
	file, err := ioutil.TempFile("", "*.yaml")
	assert.NoError(t, err)
	err = ioutil.WriteFile(file.Name(), lintFileData, 0644)
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	fmtr, err := GetFormatter("simple")
	assert.NoError(t, err)

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

	assert.NoError(t, err)
	assert.Equal(t, res.Success, false)
	assert.Equal(t, res.msg, fmt.Sprintf("%s: in object #1: lint error\n%s: in object #2: lint error\n", file.Name(), file.Name()))
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNumberOfCalls(t, "LintWorkflowTemplate", 1)
}

func TestLintStdin(t *testing.T) {
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	_, err = w.Write(lintFileData)
	assert.NoError(t, err)
	w.Close()
	stdin := os.Stdin
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	fmtr, err := GetFormatter("simple")
	assert.NoError(t, err)

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

	assert.NoError(t, err)
	assert.Equal(t, res.Success, false)
	assert.Equal(t, res.msg, "stdin: in object #1: lint error\nstdin: in object #2: lint error\n")
	wfServiceClientMock.AssertNumberOfCalls(t, "LintWorkflow", 1)
	wftServiceSclientMock.AssertNumberOfCalls(t, "LintWorkflowTemplate", 1)
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
			expectedOutput: defaultFormatter.Format(&LintResults{}),
		},
		"pretty": {
			formatterName:  "pretty",
			expectedErr:    nil,
			expectedOutput: formatterPretty{}.Format(&LintResults{}),
		},
		"simple": {
			formatterName:  "simple",
			expectedErr:    nil,
			expectedOutput: formatterSimple{}.Format(&LintResults{}),
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
					assert.EqualError(t, err, test.expectedErr.Error())
					return
				} else {
					assert.NoError(t, err)
				}
			}

			r, err := Lint(context.Background(), &LintOptions{Formatter: fmtr})
			assert.NoError(t, err)
			assert.Equal(t, test.expectedOutput, r.msg)
		})
	}
}
