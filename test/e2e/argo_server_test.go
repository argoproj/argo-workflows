package e2e

import (
	"bufio"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
	"github.com/argoproj/argo/util/kubeconfig"
)

const baseUrl = "http://localhost:2746"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
	bearerToken string
}

func (s *ArgoServerSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	var err error
	s.bearerToken, err = kubeconfig.GetBearerToken(s.RestConfig)
	if err != nil {
		panic(err)
	}
}

func (s *ArgoServerSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseUrl,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(t, true),
			},
		}).
		Builder(func(req *httpexpect.Request) {
			if s.bearerToken != "" {
				req.WithHeader("Authorization", "Bearer "+s.bearerToken)
			}
		})
}

func (s *ArgoServerSuite) TestUnauthorized() {
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = ""
	s.e(s.T()).GET("/api/v1/workflows/argo").
		Expect().
		Status(401)
}

func (s *ArgoServerSuite) TestLintWorkflow() {
	s.e(s.T()).POST("/api/v1/workflows/argo/lint").
		WithBytes([]byte((`{
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
            "image": "docker/whalesay:latest"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`))).
		Expect().
		Status(200)
}

func (s *ArgoServerSuite) TestCreateWorkflowDryRun() {
	s.e(s.T()).POST("/api/v1/workflows/argo").
		WithQuery("createOptions.dryRun", "[All]").
		WithBytes([]byte(`{
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
            "image": "docker/whalesay:latest"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)).
		Expect().
		Status(200)
}

func (s *ArgoServerSuite) TestWorkflows() {
	s.Run("Create", func(t *testing.T) {
		s.e(t).POST("/api/v1/workflows/argo").
			WithBytes([]byte(`{
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
            "image": "docker/whalesay:latest",
            "command": ["sh"],
            "args": ["-c", "sleep 10"]
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)).
			Expect().
			Status(200)
	})

	s.Run("List", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflows/").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200)
		s.e(t).GET("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})

	s.Run("Suspend", func(t *testing.T) {
		s.e(t).PUT("/api/v1/workflows/argo/test/suspend").
			Expect().
			Status(200)

		s.e(t).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec.suspend").
			Equal(true)
	})

	s.Run("Resume", func(t *testing.T) {
		s.e(t).PUT("/api/v1/workflows/argo/test/resume").
			Expect().
			Status(200)

		s.e(t).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec").
			Object().
			NotContainsKey("suspend")
	})

	s.Run("Terminate", func(t *testing.T) {
		s.e(t).PUT("/api/v1/workflows/argo/test/terminate").
			Expect().
			Status(200)

		// sleep in a test is bad practice
		time.Sleep(2 * time.Second)

		s.e(t).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.status.message").
			Equal("terminated")
	})

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflows/argo/test").
			Expect().
			Status(200)
		s.e(t).DELETE("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestCronWorkflows() {
	s.Run("Create", func(t *testing.T) {
		s.e(t).POST("/api/v1/cron-workflows/argo").
			WithBytes([]byte(`{
  "cronWorkflow": {
    "metadata": {
      "name": "test",
      "labels": {
        "argo-e2e": "true"
      }
    },
    "spec": {
      "schedule": "* * * * *",
      "workflowSpec": {
        "entrypoint": "whalesay",
        "templates": [
          {
            "name": "whalesay",
            "container": {
              "image": "docker/whalesay:latest"
            }
          }
        ]
      }
    }
  }
}`)).
			Expect().
			Status(200)
	})

	s.Run("List", func(t *testing.T) {
		s.e(t).GET("/api/v1/cron-workflows/").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/cron-workflows/argo/not-found").
			Expect().
			Status(404)
		s.e(t).GET("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			Equal("test")

	})

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200)
	})
}

// make sure we can download an artifact
func (s *ArgoServerSuite) TestWorkflowArtifact() {
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})

	s.Run("GetArtifact", func(t *testing.T) {
		s.e(t).GET("/artifacts/argo/basic/basic/main-logs").
			WithQuery("Authorization", s.bearerToken).
			Expect().
			Status(200).
			Body().
			Contains("🐙 Hello Argo!")
	})

	s.Run("GetArtifactByUid", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflows/argo/basic").
			Expect().
			Status(200)

		s.e(t).GET("/artifacts-by-uid/argo/{uid}/basic/main-logs", uid).
			WithQuery("Authorization", s.bearerToken).
			Expect().
			Status(200).
			Body().
			Contains("🐙 Hello Argo!")
	})

}

// do some basic testing on the stream methods
func (s *ArgoServerSuite) TestWorkflowStream() {

	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow()

	// use the watch to make sure that the workflow has succeeded
	s.Run("Watch", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflow-events/argo?listOptions.fieldSelector=metadata.name=basic", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		if assert.Equal(t, 200, resp.StatusCode) {
			assert.Equal(t, resp.Header.Get("Content-Type"), "text/event-stream")
			s := bufio.NewScanner(resp.Body)
			for s.Scan() {
				line := s.Text()
				if strings.Contains(line, "Succeeded") {
					break
				}
			}
		}
	})

	// then,  lets check the logs
	s.Run("PodLogs", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflows/argo/basic/basic/log?logOptions.container=main&logOptions.tailLines=3", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		if assert.NoError(t, err) {
			defer func() { _ = resp.Body.Close() }()
			if assert.Equal(t, 200, resp.StatusCode) {
				assert.Equal(t, resp.Header.Get("Content-Type"), "text/event-stream")
				s := bufio.NewScanner(resp.Body)
				for s.Scan() {
					line := s.Text()
					if strings.Contains(line, "🐙 Hello Argo!") {
						break
					}
				}
			}
		}
	})

	s.Run("PodLogsNotFound", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflows/argo/basic/not-found/log?logOptions.container=not-found", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		if assert.NoError(t, err) {
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, 404, resp.StatusCode)
		}
	})
}

func (s *ArgoServerSuite) TestArchivedWorkflow() {
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Given().
		Workflow("@smoke/basic-2.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second)

	s.Run("List", func(t *testing.T) {
		s.e(t).GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(2)

		j := s.e(t).GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			WithQuery("listOptions.limit", 1).
			WithQuery("listOptions.offset", 1).
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
		j.
			Path("$.metadata.continue").
			Equal("1")
	})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/archived-workflows/not-found").
			Expect().
			Status(404)
		s.e(t).GET("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			Equal("basic")
	})

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200)
		s.e(t).DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestWorkflowTemplates() {
	s.Run("Create", func(t *testing.T) {
		s.e(t).POST("/api/v1/workflow-templates/argo").
			WithBytes([]byte(`{
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
}`)).
			Expect().
			Status(200)
	})

	s.Run("List", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflow-templates/argo").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200)
	})

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200)
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
