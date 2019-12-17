package e2e

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

const baseUrl = "http://localhost:2746/api/v1"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
}

func (s *ArgoServerSuite) TestWorkflows() {
	t := s.T()
	t.Run("CreateWorkflow/DryRun", func(t *testing.T) {
		resp, err := http.Post(baseUrl+"/workflows/argo", "json", bytes.NewBuffer([]byte(`{
  "createOptions": {
    "dryRun": ["All"]
  },
  "workflow": {
    "metadata": {
      "name": "test",
      "labels": {
         "argo-e2e": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "docker/whalesay:latest"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)))
		if assert.NoError(t, err) {
			// GRPC is non-standard for return codes, 200 rather than 201
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			// make sure we can un-marshall the response
			err = json.Unmarshal(body, &wfv1.Workflow{})
			assert.NoError(t, err)
		}
	})
	t.Run("CreateWorkflow", func(t *testing.T) {
		resp, err := http.Post(baseUrl+"/workflows/argo", "json", bytes.NewBuffer([]byte(`{
  "workflow": {
    "metadata": {
      "name": "test",
      "labels": {
         "argo-e2e": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "docker/whalesay:latest"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)))
		if assert.NoError(t, err) {
			// GRPC is non-standard for return codes, 200 rather than 201
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			// make sure we can un-marshall the response
			err = json.Unmarshal(body, &wfv1.Workflow{})
			assert.NoError(t, err)
		}
	})
	t.Run("ListWorkflows", func(t *testing.T) {
		resp, err := http.Get(baseUrl + "/workflows/argo")
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			workflows := &wfv1.WorkflowList{}
			err = json.Unmarshal(body, workflows)
			assert.NoError(t, err)
			assert.Len(t, workflows.Items, 1)
		}
	})
	t.Run("GetWorkflow", func(t *testing.T) {
		resp, err := http.Get(baseUrl + "/workflows/argo/test")
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			err = json.Unmarshal(body, &wfv1.Workflow{})
			assert.NoError(t, err)
		}
	})
	t.Run("DeleteWorkflow", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", baseUrl+"/workflows/argo/test", nil)
		assert.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
		}
	})
}

func (s *ArgoServerSuite) TestWorkflowTemplates() {
	t := s.T()
	t.Run("CreateWorkflowTemplate", func(t *testing.T) {
		resp, err := http.Post(baseUrl+"/workflowtemplates/argo", "json", bytes.NewBuffer([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "argo-e2e": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "docker/whalesay:latest"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)))
		if assert.NoError(t, err) {
			// GRPC is non-standard for return codes, 200 rather than 201
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			// make sure we can un-marshall the response
			err = json.Unmarshal(body, &wfv1.WorkflowTemplate{})
			assert.NoError(t, err)
		}

	})
	t.Run("ListWorkflowTemplates", func(t *testing.T) {
		resp, err := http.Get(baseUrl + "/workflowtemplates/argo")
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			templates := &wfv1.WorkflowTemplateList{}
			err = json.Unmarshal(body, templates)
			assert.NoError(t, err)
			assert.Len(t, templates.Items, 1)
		}
	})
	t.Run("GetWorkflowTemplates", func(t *testing.T) {
		resp, err := http.Get(baseUrl + "/workflowtemplates/argo/test")
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			err = json.Unmarshal(body, &wfv1.WorkflowTemplate{})
			assert.NoError(t, err)
		}
	})
	t.Run("DeleteWorkflowTemplates", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", baseUrl+"/workflowtemplates/argo/test", nil)
		assert.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if assert.NoError(t, err) {
			assert.Equal(t, "200 OK", resp.Status)
		}
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
