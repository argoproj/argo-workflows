package e2e

import (
	"encoding/base64"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"

	"github.com/argoproj/argo/test/e2e/fixtures"
)

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
	e         *httpexpect.Expect
	authToken string
}

func (s *ArgoServerSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	kubeConfigBytes, err := ioutil.ReadFile(fixtures.KubeConfig)
	if err != nil {
		panic(err)
	}
	s.authToken = base64.StdEncoding.EncodeToString(kubeConfigBytes)
	s.e = httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  "http://localhost:2746/api/v1",
			Reporter: httpexpect.NewRequireReporter(s.T()),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.T(), true),
			},
		}).
		Builder(func(req *httpexpect.Request) {
			if s.authToken != "" {
				req.WithHeader("Authorization", "Bearer "+s.authToken)
			}
		})
}

func (s *ArgoServerSuite) TestUnauthorized() {
	token := s.authToken
	defer func() { s.authToken = token }()
	s.authToken = ""
	s.e.GET("/workflows/argo").
		Expect().
		Status(401)
}

func (s *ArgoServerSuite) TestLintWorkflow() {
	s.e.POST("/workflows/argo/lint").
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
	s.e.POST("/workflows/argo").
		WithBytes([]byte(`{
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
	s.e.POST("/workflows/argo").
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

	s.e.GET("/workflows/argo").
		Expect().
		Status(200).
		JSON().
		Path("$.items").
		Array().
		Length().
		Equal(1)

	s.e.GET("/workflows/argo/test").
		Expect().
		Status(200)

	s.e.PUT("/workflows/argo/test/suspend").
		Expect().
		Status(200)

	s.e.GET("/workflows/argo/test").
		Expect().
		Status(200).
		JSON().
		Path("$.spec.suspend").
		Equal(true)

	s.e.PUT("/workflows/argo/test/resume").
		Expect().
		Status(200)

	s.e.GET("/workflows/argo/test").
		Expect().
		Status(200).
		JSON().
		Path("$.spec").
		Object().
		NotContainsKey("suspend")

	s.e.PUT("/workflows/argo/test/terminate").
		Expect().
		Status(200)

	// sleep in a test is bad practice
	time.Sleep(1 * time.Second)

	s.e.GET("/workflows/argo/test").
		Expect().
		Status(200).
		JSON().
		Path("$.status.message").
		Equal("terminated")

	s.e.DELETE("/workflows/argo/test").
		Expect().
		Status(200)
}

func (s *ArgoServerSuite) TestWorkflowTemplates() {
	s.e.POST("/workflowtemplates/argo").
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

	s.e.GET("/workflowtemplates/argo").
		Expect().
		Status(200).
		JSON().
		Path("$.items").
		Array().
		Length().
		Equal(1)

	s.e.GET("/workflowtemplates/argo/test").
		Expect().
		Status(200)

	s.e.DELETE("/workflowtemplates/argo/test").
		Expect().
		Status(200)
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
