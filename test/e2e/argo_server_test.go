package e2e

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/gavv/httpexpect.v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

func (s *ArgoServerSuite) AfterTest(suiteName, testName string) {
	s.E2ESuite.AfterTest(suiteName, testName)
}

func (s *ArgoServerSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseUrl,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.Diagnostics, true),
			},
		}).
		Builder(func(req *httpexpect.Request) {
			if s.bearerToken != "" {
				req.WithHeader("Authorization", "Bearer "+s.bearerToken)
			}
		})
}

func (s *ArgoServerSuite) TestInfo() {
	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/info").
			Expect().
			Status(200).
			JSON().
			Path("$.managedNamespace").
			Equal("argo")
	})
}

func (s *ArgoServerSuite) TestUnauthorized() {
	s.T().SkipNow() // TODO minikube doesn't support token auth
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = "test-token"
	s.e(s.T()).GET("/api/v1/workflows/argo").
		Expect().
		Status(401)
}

func (s *ArgoServerSuite) TestPermission() {
	s.T().SkipNow() // TODO
	nsName := fmt.Sprintf("%s-%d", "test-rbac", time.Now().Unix())
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	s.Run("Create ns", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().Namespaces().Create(ns)
		assert.NoError(t, err)
	})
	defer func() {
		// Clean up created namespace
		_ = s.KubeClient.CoreV1().Namespaces().Delete(nsName, nil)
	}()
	forbiddenNsName := fmt.Sprintf("%s-%s", nsName, "fb")
	forbiddenNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: forbiddenNsName}}
	s.Run("Create forbidden ns", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().Namespaces().Create(forbiddenNs)
		assert.NoError(t, err)
	})
	defer func() {
		_ = s.KubeClient.CoreV1().Namespaces().Delete(forbiddenNsName, nil)
	}()
	// Create serviceaccount in good ns
	saName := "argotest"
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName}}
	s.Run("Create service account in good ns", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Create(sa)
		assert.NoError(t, err)
	})
	// Create serviceaccount in forbidden ns
	forbiddenSaName := "argotest"
	forbiddenSa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: forbiddenSaName}}
	s.Run("Create service account in forbidden ns", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(forbiddenNsName).Create(forbiddenSa)
		assert.NoError(t, err)
	})

	// Create RBAC Role in good ns
	roleName := "argotest-role"
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: roleName},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"workflows", "workflowtemplates", "cronworkflows", "workflows/finalizers", "workflowtemplates/finalizers", "cronworkflows/finalizers"},
				Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
			},
		},
	}
	s.Run("Create Role", func(t *testing.T) {
		_, err := s.KubeClient.RbacV1().Roles(nsName).Create(role)
		assert.NoError(t, err)
	})

	// Create RBAC RoleBinding in good ns
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "argotest-role-binding"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
	s.Run("Create RoleBinding", func(t *testing.T) {
		_, err := s.KubeClient.RbacV1().RoleBindings(nsName).Create(roleBinding)
		assert.NoError(t, err)
	})

	// Get token of serviceaccount in good ns
	var goodToken string
	s.Run("Get good serviceaccount token", func(t *testing.T) {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(saName, metav1.GetOptions{})
		if assert.NoError(t, err) {
			secretName := sAccount.Secrets[0].Name
			secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(secretName, metav1.GetOptions{})
			assert.NoError(t, err)
			// Argo server API expects it to be encoded.
			goodToken = base64.StdEncoding.EncodeToString(secret.Data["token"])
		}
	})

	var forbiddenToken string
	s.Run("Get forbidden serviceaccount token", func(t *testing.T) {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(forbiddenNsName).Get(forbiddenSaName, metav1.GetOptions{})
		assert.NoError(t, err)
		secretName := sAccount.Secrets[0].Name
		secret, err := s.KubeClient.CoreV1().Secrets(forbiddenNsName).Get(secretName, metav1.GetOptions{})
		assert.NoError(t, err)
		// Argo server API expects it to be encoded.
		forbiddenToken = base64.StdEncoding.EncodeToString(secret.Data["token"])
	})

	token := s.bearerToken
	defer func() { s.bearerToken = token }()

	// Test creating workflow in good ns
	s.bearerToken = goodToken
	s.Run("Create workflow in good ns", func(t *testing.T) {
		s.bearerToken = goodToken
		s.e(t).POST("/api/v1/workflows/" + nsName).
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

	// Test list workflows in good ns
	s.Run("List", func(t *testing.T) {
		s.bearerToken = goodToken
		s.e(t).GET("/api/v1/workflows/"+nsName).
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	// Test creating workflow in forbidden ns
	s.Run("Create workflow in forbidden ns", func(t *testing.T) {
		s.bearerToken = goodToken
		s.e(t).POST("/api/v1/workflows/" + forbiddenNsName).
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
            "imagePullPolicy": "IfNotPresent",
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
			Status(403)
	})

	// Test list workflows in good ns with forbidden ns token
	s.bearerToken = forbiddenToken
	s.Run("List", func(t *testing.T) {
		s.bearerToken = forbiddenToken
		s.e(t).GET("/api/v1/workflows/" + nsName).
			Expect().
			Status(403)
	})
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
            "image": "docker/whalesay:latest",
            "imagePullPolicy": "IfNotPresent"
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
            "image": "docker/whalesay:latest",
            "imagePullPolicy": "IfNotPresent"
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
         "argo-e2e": "subject"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "docker/whalesay:latest",
            "imagePullPolicy": "IfNotPresent",
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
		// make sure list options work correctly
		s.Given().
			Workflow("@smoke/basic.yaml")

		s.e(t).GET("/api/v1/workflows/").
			WithQuery("listOptions.labelSelector", "argo-e2e=subject").
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
        "argo-e2e": "subject"
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
              "image": "docker/whalesay:latest",
              "imagePullPolicy": "IfNotPresent"
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
		// make sure list options work correctly
		s.Given().
			CronWorkflow("@testdata/basic.yaml")

		s.e(t).GET("/api/v1/cron-workflows/").
			WithQuery("listOptions.labelSelector", "argo-e2e=subject").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	var resourceVersion string
	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/cron-workflows/argo/not-found").
			Expect().
			Status(404)
		resourceVersion = s.e(t).GET("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func(t *testing.T) {
		s.e(t).PUT("/api/v1/cron-workflows/argo/test").
			WithBytes([]byte(`{"cronWorkflow": {
    "metadata": {
      "name": "test",
      "resourceVersion": "` + resourceVersion + `",
      "labels": {
        "argo-e2e": "true"
      }
    },
    "spec": {
      "schedule": "1 * * * *",
      "workflowSpec": {
        "entrypoint": "whalesay",
        "templates": [
          {
            "name": "whalesay",
            "container": {
              "image": "docker/whalesay:latest",
              "imagePullPolicy": "IfNotPresent"
            }
          }
        ]
      }
    }
  }}`)).
			Expect().
			Status(200).
			JSON().
			Path("$.spec.schedule").
			Equal("1 * * * *")
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

		s.e(t).GET("/artifacts-by-uid/{uid}/basic/main-logs", uid).
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

	time.Sleep(1 * time.Second)

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
				log.WithField("line", line).Debug()
				// make sure we have this enabled
				if line == "" {
					continue
				}
				if strings.Contains(line, `status:`) {
					assert.Contains(t, line, `"offloadNodeStatus":true`)
					// so that we get this
					assert.Contains(t, line, `"nodes":`)
				}
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
		WaitForWorkflow(20 * time.Second).
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

	s.Run("Lint", func(t *testing.T) {
		s.e(t).POST("/api/v1/workflow-templates/argo/lint").
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
            "image": "docker/whalesay:latest",
            "imagePullPolicy": "IfNotPresent"
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

	s.Run("Create", func(t *testing.T) {
		s.e(t).POST("/api/v1/workflow-templates/argo").
			WithBytes([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "argo-e2e": "subject"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "docker/whalesay:latest",
            "imagePullPolicy": "IfNotPresent"
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

		// make sure list options work correctly
		s.Given().
			WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml")

		s.e(t).GET("/api/v1/workflow-templates/argo").
			WithQuery("listOptions.labelSelector", "argo-e2e=subject").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	var resourceVersion string
	s.Run("Get", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflow-templates/argo/not-found").
			Expect().
			Status(404)

		resourceVersion = s.e(t).GET("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func(t *testing.T) {
		s.e(t).PUT("/api/v1/workflow-templates/argo/test").
			WithBytes([]byte(`{"template": {
    "metadata": {
      "name": "test",
      "resourceVersion": "` + resourceVersion + `",
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
            "image": "docker/whalesay:dev",
            "imagePullPolicy": "IfNotPresent"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)).
			Expect().
			Status(200).
			JSON().
			Path("$.spec.templates[0].container.image").
			Equal("docker/whalesay:dev")
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
