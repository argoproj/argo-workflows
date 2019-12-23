package e2e

import (
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo/test/e2e/fixtures"
)

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
	e              *httpexpect.Expect
	authToken      string
	db             sqlbuilder.Database
	runningLocally bool
}

func (s *ArgoServerSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	//is the server running running locally, or on a cluster
	list, err := s.KubeClient.CoreV1().Pods(fixtures.Namespace).List(metav1.ListOptions{LabelSelector: "app=argo-server"})
	if err != nil {
		panic(err)
	}
	s.runningLocally = len(list.Items) == 0
	// if argo-server we are not running locally, then we are running in the cluster, and we need the kubeconfig
	if !s.runningLocally {
		bytes, err := ioutil.ReadFile(filepath.Join("kubeconfig"))
		if err != nil {
			panic(err)
		}
		s.authToken = base64.StdEncoding.EncodeToString(bytes)
	}
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
	// create database collection
	s.db, err = postgresql.Open(postgresql.ConnectionURL{User: "postgres", Password: "password", Host: "localhost"})
	if err != nil {
		panic(err)
	}
	// delete everything from history
	_, err = s.db.DeleteFrom("argo_workflow_history").Exec()
	if err != nil {
		panic(err)
	}
}

func (s *ArgoServerSuite) TestUnauthorized() {
	if s.runningLocally {
		s.T().SkipNow()
	}
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
	time.Sleep(2 * time.Second)

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

func (s *ArgoServerSuite) TestWorkflowHistory() {
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second)

	s.Given().
		Workflow("@smoke/basic-2.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(15 * time.Second)

	s.e.GET("/workflowhistory").
		Expect().
		Status(200).
		JSON().
		Path("$.items").
		Array().
		Length().
		Equal(2)

	json := s.e.GET("/workflowhistory").
		WithQuery("listOptions.limit", 1).
		WithQuery("listOptions.offset", 1).
		Expect().
		Status(200).
		JSON()
	json.
		Path("$.items").
		Array().
		Length().
		Equal(1)
	json.
		Path("$.metadata.continue").
		Equal("1")
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
