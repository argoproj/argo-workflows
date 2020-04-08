package e2e

import (
	"bufio"
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
	s.bearerToken, err = s.GetServiceAccountToken()
	s.CheckError(err)
}

type httpLogger struct {
}

func (d *httpLogger) Logf(fmt string, args ...interface{}) {
	log.Debugf(fmt, args...)
}

func (s *ArgoServerSuite) e(t *testing.T) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseUrl,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(&httpLogger{}, true),
			},
		}).
		Builder(func(req *httpexpect.Request) {
			if s.bearerToken != "" {
				req.WithHeader("Authorization", "Bearer "+s.bearerToken)
			}
		})
}

func (s *ArgoServerSuite) TestInfo() {
	s.Run("Get", func() {
		json := s.e(s.T()).GET("/api/v1/info").
			Expect().
			Status(200).
			JSON()
		json.
			Path("$.managedNamespace").
			Equal("argo")
		json.
			Path("$.links[0].name").
			Equal("Example Workflow Link")
		json.
			Path("$.links[0].scope").
			Equal("workflow")
		json.
			Path("$.links[0].url").
			Equal("http://logging-facility?namespace=${metadata.namespace}&workflowName=${metadata.name}")
	})
}

func (s *ArgoServerSuite) TestUnauthorized() {
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = "test-token"
	s.e(s.T()).GET("/api/v1/workflows/argo").
		Expect().
		Status(401)
}
func (s *ArgoServerSuite) TestCookieAuth() {
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = ""
	s.e(s.T()).GET("/api/v1/workflows/argo").
		WithHeader("Cookie", "authorization=Bearer "+token).
		Expect().
		Status(200)
}

func (s *ArgoServerSuite) TestPermission() {
	nsName := fixtures.Namespace
	// Create good serviceaccount
	goodSaName := "argotestgood"
	goodSa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: goodSaName}}
	s.Run("CreateGoodSA", func() {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Create(goodSa)
		assert.NoError(s.T(), err)
	})
	defer func() {
		// Clean up created sa
		_ = s.KubeClient.CoreV1().ServiceAccounts(nsName).Delete(goodSaName, nil)
	}()

	// Create bad serviceaccount
	badSaName := "argotestbad"
	badSa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: badSaName}}
	s.Run("CreateBadSA", func() {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Create(badSa)
		assert.NoError(s.T(), err)
	})
	defer func() {
		_ = s.KubeClient.CoreV1().ServiceAccounts(nsName).Delete(badSaName, nil)
	}()

	// Create RBAC Role
	var roleName string
	s.Run("LoadRoleYaml", func() {
		obj, err := fixtures.LoadObject("@testdata/argo-server-test-role.yaml")
		assert.NoError(s.T(), err)
		role, _ := obj.(*rbacv1.Role)
		roleName = role.Name
		_, err = s.KubeClient.RbacV1().Roles(nsName).Create(role)
		assert.NoError(s.T(), err)
	})
	defer func() {
		_ = s.KubeClient.RbacV1().Roles(nsName).Delete(roleName, nil)
	}()

	// Create RBAC RoleBinding
	roleBindingName := "argotest-role-binding"
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: roleBindingName},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: goodSaName}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
	s.Run("CreateRoleBinding", func() {
		_, err := s.KubeClient.RbacV1().RoleBindings(nsName).Create(roleBinding)
		assert.NoError(s.T(), err)
	})
	defer func() {
		_ = s.KubeClient.RbacV1().RoleBindings(nsName).Delete(roleBindingName, nil)
	}()

	// Sleep 2 seconds to wait for serviceaccount token created.
	// The secret creation slowness is seen in k3d.
	time.Sleep(2 * time.Second)

	// Get token of good serviceaccount
	var goodToken string
	s.Run("GetGoodSAToken", func() {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(goodSaName, metav1.GetOptions{})
		if assert.NoError(s.T(), err) {
			secretName := sAccount.Secrets[0].Name
			secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(secretName, metav1.GetOptions{})
			assert.NoError(s.T(), err)
			goodToken = string(secret.Data["token"])
		}
	})

	// Get token of bad serviceaccount
	var badToken string
	s.Run("GetBadSAToken", func() {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(badSaName, metav1.GetOptions{})
		assert.NoError(s.T(), err)
		secretName := sAccount.Secrets[0].Name
		secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(secretName, metav1.GetOptions{})
		assert.NoError(s.T(), err)
		badToken = string(secret.Data["token"])
	})

	token := s.bearerToken
	defer func() { s.bearerToken = token }()

	// Test creating workflow with good token
	var uid string
	s.bearerToken = goodToken
	s.Run("CreateWFGoodToken", func() {
		uid = s.e(s.T()).POST("/api/v1/workflows/" + nsName).
			WithBytes([]byte(`{
  "workflow": {
    "metadata": {
      "name": "test-wf-good",
      "labels": {
         "argo-e2e": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "cowsay:v1",
            "command": ["sh"],
            "args": ["-c", "sleep 1"]
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
			Path("$.metadata.uid").
			Raw().(string)
	})

	// Test list workflows with good token
	s.bearerToken = goodToken
	s.Run("ListWFsGoodToken", func() {
		s.e(s.T()).GET("/api/v1/workflows/"+nsName).
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	// Test creating workflow with bad token
	s.bearerToken = badToken
	s.Run("CreateWFBadToken", func() {
		s.e(s.T()).POST("/api/v1/workflows/" + nsName).
			WithBytes([]byte(`{
  "workflow": {
    "metadata": {
      "name": "test-wf-bad",
      "labels": {
         "argo-e2e": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "cowsay:v1",
            "imagePullPolicy": "IfNotPresent",
            "command": ["sh"],
            "args": ["-c", "sleep 1"]
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

	// Test list workflows with bad token
	s.bearerToken = badToken
	s.Run("ListWFsBadToken", func() {
		s.e(s.T()).GET("/api/v1/workflows/" + nsName).
			Expect().
			Status(403)
	})

	if s.Persistence.IsEnabled() {

		// Simply wait 10 seconds for the wf to be completed
		s.Given().
			WorkflowName("test-wf-good").
			When().
			WaitForWorkflow(30 * time.Second)

		// Test delete workflow with bad token
		s.bearerToken = badToken
		s.Run("DeleteWFWithBadToken", func() {
			s.e(s.T()).DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
				Expect().
				Status(403)
		})

		// Test delete workflow with good token
		s.bearerToken = goodToken
		s.Run("DeleteWFWithGoodToken", func() {
			s.e(s.T()).DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
				Expect().
				Status(200)
		})

		// we've now deleted the workflow, but it is still in the archive, testing the archive
		// after deleting the workflow makes sure that we are no dependant of the workflow for authorization

		if s.Persistence.IsEnabled() {
			// Test list archived WFs with good token
			s.bearerToken = goodToken
			s.Run("ListArchivedWFsGoodToken", func() {
				s.e(s.T()).GET("/api/v1/archived-workflows").
					WithQuery("listOptions.labelSelector", "argo-e2e").
					WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
					Expect().
					Status(200).
					JSON().
					Path("$.items").
					Array().Length().Gt(0)
			})

			s.bearerToken = badToken
			s.Run("ListArchivedWFsBadToken", func() {
				s.e(s.T()).GET("/api/v1/archived-workflows").
					WithQuery("listOptions.labelSelector", "argo-e2e").
					WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
					Expect().
					Status(403)
			})

			// Test get archived wf with good token
			s.bearerToken = goodToken
			s.Run("GetArchivedWFsGoodToken", func() {
				s.e(s.T()).GET("/api/v1/archived-workflows/"+uid).
					WithQuery("listOptions.labelSelector", "argo-e2e").
					Expect().
					Status(200)
			})

			// Test get archived wf with bad token
			s.bearerToken = badToken
			s.Run("GetArchivedWFsBadToken", func() {
				s.e(s.T()).GET("/api/v1/archived-workflows/" + uid).
					Expect().
					Status(403)
			})

			// Test deleting archived wf with bad token
			s.bearerToken = badToken
			s.Run("DeleteArchivedWFsBadToken", func() {
				s.e(s.T()).DELETE("/api/v1/archived-workflows/" + uid).
					Expect().
					Status(403)
			})
			// Test deleting archived wf with good token
			s.bearerToken = goodToken
			s.Run("DeleteArchivedWFsGoodToken", func() {
				s.e(s.T()).DELETE("/api/v1/archived-workflows/" + uid).
					Expect().
					Status(200)
			})
		}
	}
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
            "image": "cowsay:v1",
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
            "image": "cowsay:v1",
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
		Path("$.metadata").
		Object().
		NotContainsKey("uid")
}

func (s *ArgoServerSuite) TestWorkflowService() {

	s.Run("Create", func() {
		s.e(s.T()).POST("/api/v1/workflows/argo").
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
            "image": "cowsay:v1",
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

	s.Run("List", func() {
		s.Given().
			WorkflowName("test").
			When().
			WaitForWorkflowToStart(20 * time.Second)

		j := s.e(s.T()).GET("/api/v1/workflows/argo").
			WithQuery("listOptions.labelSelector", "argo-e2e=subject").
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
		if s.Persistence.IsEnabled() {
			// check we are loading offloaded node status
			j.Path("$.items[0].status.offloadNodeStatusVersion").
				NotNull()
		}
		j.Path("$.items[0].status.nodes").
			NotNull()
	})

	s.Run("ListWithFields", func() {
		j := s.e(s.T()).GET("/api/v1/workflows/argo").
			WithQuery("listOptions.labelSelector", "argo-e2e=subject").
			WithQuery("fields", "-items.status.nodes,items.status.finishedAt,items.status.startedAt").
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
		j.Path("$.items[0].status").Object().ContainsKey("phase").NotContainsKey("nodes")
	})

	s.Run("Get", func() {
		j := s.e(s.T()).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON()
		if s.Persistence.IsEnabled() {
			// check we are loading offloaded node status
			j.
				Path("$.status.offloadNodeStatusVersion").
				NotNull()
		}
		j.Path("$.status.nodes").
			NotNull()
		s.e(s.T()).GET("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})

	s.Run("GetWithFields", func() {
		j := s.e(s.T()).GET("/api/v1/workflows/argo/test").
			WithQuery("fields", "status.phase").
			Expect().
			Status(200).
			JSON()
		j.Path("$.status").Object().ContainsKey("phase").NotContainsKey("nodes")
	})

	s.Run("Suspend", func() {
		s.e(s.T()).PUT("/api/v1/workflows/argo/test/suspend").
			Expect().
			Status(200)

		s.e(s.T()).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec.suspend").
			Equal(true)
	})

	s.Run("Resume", func() {
		s.e(s.T()).PUT("/api/v1/workflows/argo/test/resume").
			Expect().
			Status(200)

		s.e(s.T()).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.spec").
			Object().
			NotContainsKey("suspend")
	})

	s.Run("Terminate", func() {
		s.e(s.T()).PUT("/api/v1/workflows/argo/test/terminate").
			Expect().
			Status(200)

		// sleep in a test is bad practice
		time.Sleep(2 * time.Second)

		s.e(s.T()).GET("/api/v1/workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.status.message").
			Equal("Stopped with strategy 'Terminate'")
	})

	s.Run("Delete", func() {
		s.e(s.T()).DELETE("/api/v1/workflows/argo/test").
			Expect().
			Status(200)
		s.e(s.T()).DELETE("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestCronWorkflowService() {
	s.Run("Create", func() {
		s.e(s.T()).POST("/api/v1/cron-workflows/argo").
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
              "image": "cowsay:v1",
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

	s.Run("List", func() {
		// make sure list options work correctly
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic
  labels:
    argo-e2e: true
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowMetadata:
    labels:
      argo-e2e: true
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: cowsay:v1
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]
`)

		s.e(s.T()).GET("/api/v1/cron-workflows/argo").
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
	s.Run("Get", func() {
		s.e(s.T()).GET("/api/v1/cron-workflows/argo/not-found").
			Expect().
			Status(404)
		resourceVersion = s.e(s.T()).GET("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func() {
		s.e(s.T()).PUT("/api/v1/cron-workflows/argo/test").
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
      "workflowMetadata": {
        "labels": {"argo-e2e": "true"}
      },
      "workflowSpec": {
        "entrypoint": "whalesay",
        "templates": [
          {
            "name": "whalesay",
            "container": {
              "image": "cowsay:v1",
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

	s.Run("Delete", func() {
		s.e(s.T()).DELETE("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200)
	})
}

// make sure we can download an artifact
func (s *ArgoServerSuite) TestArtifactServer() {
	if !s.Persistence.IsEnabled() {
		s.T().SkipNow()
	}
	var uid types.UID
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(20 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})

	s.Run("GetArtifact", func() {
		s.e(s.T()).GET("/artifacts/argo/basic/basic/main-logs").
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})
	s.Run("GetArtifactByUID", func() {
		s.e(s.T()).DELETE("/api/v1/workflows/argo/basic").
			Expect().
			Status(200)

		s.e(s.T()).GET("/artifacts-by-uid/{uid}/basic/main-logs", uid).
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})

	// as the artifact server has some special code for cookies, we best test that too
	s.Run("GetArtifactByUIDUsingCookie", func() {
		token := s.bearerToken
		defer func() { s.bearerToken = token }()
		s.bearerToken = ""
		s.e(s.T()).GET("/artifacts-by-uid/{uid}/basic/main-logs", uid).
			WithHeader("Cookie", "authorization=Bearer "+token).
			Expect().
			Status(200)
	})

}

// do some basic testing on the stream methods
func (s *ArgoServerSuite) TestWorkflowServiceStream() {

	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflowToStart(10 * time.Second)

	// use the watch to make sure that the workflow has succeeded
	s.Run("Watch", func() {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflow-events/argo?listOptions.fieldSelector=metadata.name=basic", nil)
		assert.NoError(s.T(), err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), resp)
		defer func() {
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
		if assert.Equal(s.T(), 200, resp.StatusCode) {
			assert.Equal(s.T(), resp.Header.Get("Content-Type"), "text/event-stream")
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				log.WithField("line", line).Debug()
				// make sure we have this enabled
				if line == "" {
					continue
				}
				if strings.Contains(line, `status:`) {
					assert.Contains(s.T(), line, `"offloadNodeStatus":true`)
					// so that we get this
					assert.Contains(s.T(), line, `"nodes":`)
				}
				if strings.Contains(line, "Succeeded") {
					break
				}
			}
		}
	})

	// then,  lets check the logs
	s.Run("PodLogs", func() {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflows/argo/basic/basic/log?logOptions.container=main&logOptions.tailLines=3", nil)
		assert.NoError(s.T(), err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		if assert.NoError(s.T(), err) {
			defer func() { _ = resp.Body.Close() }()
			if assert.Equal(s.T(), 200, resp.StatusCode) {
				assert.Equal(s.T(), resp.Header.Get("Content-Type"), "text/event-stream")
				s := bufio.NewScanner(resp.Body)
				for s.Scan() {
					line := s.Text()
					if strings.Contains(line, ":) Hello Argo!") {
						break
					}
				}
			}
		}
	})
}

func (s *ArgoServerSuite) TestArchivedWorkflowService() {
	if !s.Persistence.IsEnabled() {
		s.T().SkipNow()
	}
	var uid types.UID
	s.Given().
		Workflow(`
metadata:
  name: archie
  labels:
    argo-e2e: 1
spec:
  entrypoint: run-archie
  templates:
    - name: run-archie
      container:
        image: cowsay:v1
        command: [cowsay, ":) Hello Argo!"]
        imagePullPolicy: IfNotPresent`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(20 * time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Given().
		Workflow(`
metadata:
  name: betty
  labels:
    argo-e2e: 2
spec:
  entrypoint: run-betty
  templates:
    - name: run-betty
      container:
        image: cowsay:v1
        command: [cowsay, ":) Hello Argo!"]
        imagePullPolicy: IfNotPresent`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(20 * time.Second)

	for _, tt := range []struct {
		name     string
		selector string
		wantLen  int
	}{
		{"ListDoesNotExist", "!argo-e2e", 0},
		{"ListEquals", "argo-e2e=1", 1},
		{"ListDoubleEquals", "argo-e2e==1", 1},
		{"ListIn", "argo-e2e in (1)", 1},
		{"ListNotEquals", "argo-e2e!=1", 1},
		{"ListNotIn", "argo-e2e notin (1)", 1},
		{"ListExists", "argo-e2e", 2},
		{"ListGreaterThan0", "argo-e2e>0", 2},
		{"ListGreaterThan1", "argo-e2e>1", 1},
		{"ListLessThan1", "argo-e2e<1", 0},
		{"ListLessThan2", "argo-e2e<2", 1},
	} {
		s.Run(tt.name, func() {
			path := s.e(s.T()).GET("/api/v1/archived-workflows").
				WithQuery("listOptions.fieldSelector", "metadata.namespace=argo").
				WithQuery("listOptions.labelSelector", tt.selector).
				Expect().
				Status(200).
				JSON().
				Path("$.items")

			if tt.wantLen == 0 {
				path.Null()
			} else {
				path.Array().
					Length().
					Equal(tt.wantLen)
			}
		})
	}

	s.Run("ListWithLimitAndOffset", func() {
		j := s.e(s.T()).GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			WithQuery("listOptions.fieldSelector", "metadata.namespace=argo").
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

	s.Run("ListWithMinStartedAtGood", func() {
		fieldSelector := "metadata.namespace=argo,spec.startedAt>" + time.Now().Add(-1*time.Hour).Format(time.RFC3339) + ",spec.startedAt<" + time.Now().Add(1*time.Hour).Format(time.RFC3339)
		s.e(s.T()).GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			WithQuery("listOptions.fieldSelector", fieldSelector).
			WithQuery("listOptions.limit", 2).
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(2)
	})

	s.Run("ListWithMinStartedAtBad", func() {
		s.e(s.T()).GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "argo-e2e").
			WithQuery("listOptions.fieldSelector", "metadata.namespace=argo,spec.startedAt>"+time.Now().Add(1*time.Hour).Format(time.RFC3339)).
			WithQuery("listOptions.limit", 2).
			Expect().
			Status(200).
			JSON().
			Path("$.items").Null()
	})

	s.Run("Get", func() {
		s.e(s.T()).GET("/api/v1/archived-workflows/not-found").
			Expect().
			Status(404)
		s.e(s.T()).GET("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			Equal("archie")
	})

	s.Run("Delete", func() {
		s.e(s.T()).DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200)
		s.e(s.T()).DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestWorkflowTemplateService() {

	s.Run("Lint", func() {
		s.e(s.T()).POST("/api/v1/workflow-templates/argo/lint").
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
            "image": "cowsay:v1",
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

	s.Run("Create", func() {
		s.e(s.T()).POST("/api/v1/workflow-templates/argo").
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
            "image": "cowsay:v1",
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

	s.Run("List", func() {

		// make sure list options work correctly
		s.Given().
			WorkflowTemplate("@smoke/workflow-template-whalesay-template.yaml").
			When().
			CreateWorkflowTemplates()

		s.e(s.T()).GET("/api/v1/workflow-templates/argo").
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
	s.Run("Get", func() {
		s.e(s.T()).GET("/api/v1/workflow-templates/argo/not-found").
			Expect().
			Status(404)

		resourceVersion = s.e(s.T()).GET("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func() {
		s.e(s.T()).PUT("/api/v1/workflow-templates/argo/test").
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
            "image": "cowsay:v2",
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
			Equal("cowsay:v2")
	})

	s.Run("Delete", func() {
		s.e(s.T()).DELETE("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200)
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
