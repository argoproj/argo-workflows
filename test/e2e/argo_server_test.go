package e2e

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
	bearerToken string
}

func getClientConfig() *workflow.ClientConfig {
	bytes, err := ioutil.ReadFile(filepath.Join("kubeconfig"))
	if err != nil {
		panic(err)
	}
	config, err := clientcmd.NewClientConfigFromBytes(bytes)
	if err != nil {
		panic(err)
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		panic(err)
	}
	var clientConfig workflow.ClientConfig
	_ = copier.Copy(&clientConfig, restConfig)
	return &clientConfig
}

func (s *ArgoServerSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	jsonConfig, err := json.Marshal(getClientConfig())
	if err != nil {
		panic(err)
	}
	s.bearerToken = base64.StdEncoding.EncodeToString(jsonConfig)
}

func (s *ArgoServerSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  "http://localhost:2746/api/v1",
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
	s.e(s.T()).GET("/workflows/argo").
		Expect().
		Status(401)
}

func (s *ArgoServerSuite) TestLintWorkflow() {
	s.e(s.T()).POST("/workflows/argo/lint").
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
	s.e(s.T()).POST("/workflows/argo").
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
	s.T().Run("Create", func(t *testing.T) {
		s.e(t).POST("/workflows/argo").
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

	s.T().Run("List", func(t *testing.T) {
		s.e(t).GET("/workflows/argo").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.T().Run("Get", func(t *testing.T) {
		s.e(t).GET("/workflows/argo/test").
			Expect().
			Status(200)
	})

	s.T().Run("Suspend", func(t *testing.T) {
		s.e(t).PUT("/workflows/argo/test/suspend").
			Expect().
			Status(200)

		s.e(t).GET("/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec.suspend").
			Equal(true)
	})

	s.T().Run("Resume", func(t *testing.T) {
		s.e(t).PUT("/workflows/argo/test/resume").
			Expect().
			Status(200)

		s.e(t).GET("/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec").
			Object().
			NotContainsKey("suspend")
	})

	s.T().Run("Terminate", func(t *testing.T) {
		s.e(t).PUT("/workflows/argo/test/terminate").
			Expect().
			Status(200)

		// sleep in a test is bad practice
		time.Sleep(2 * time.Second)

		s.e(t).GET("/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.status.message").
			Equal("terminated")
	})

	s.T().Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/workflows/argo/test").
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestWorkflowHistory() {
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Given().
		Workflow("@smoke/basic-2.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second)

	s.T().Run("List", func(t *testing.T) {
		s.e(t).GET("/workflow-history").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(2)

		j := s.e(t).GET("/workflow-history").
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

	s.T().Run("Get", func(t *testing.T) {
		s.e(t).GET("/workflow-history/argo/not-found").
			Expect().
			Status(404)
		s.e(t).GET("/workflow-history/argo/{uid}", uid).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			Equal("basic")
	})

	s.T().Run("Resubmit", func(t *testing.T) {
		s.e(t).PUT("/workflow-history/argo/{uid}/resubmit", uid).
			Expect().
			Status(200)

		s.e(t).GET("/workflows/argo").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(3)
	})
	s.T().Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/workflow-history/argo/{uid}", uid).
			Expect().
			Status(200)
		s.e(t).DELETE("/workflow-history/argo/{uid}", uid).
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestWorkflowTemplates() {
	s.T().Run("Create", func(t *testing.T) {
		s.e(t).POST("/workflowtemplates/argo").
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

	s.T().Run("List", func(t *testing.T) {
		s.e(t).GET("/workflowtemplates/argo").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.T().Run("Get", func(t *testing.T) {
		s.e(t).GET("/workflowtemplates/argo/test").
			Expect().
			Status(200)
	})

	s.T().Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/workflowtemplates/argo/test").
			Expect().
			Status(200)
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
