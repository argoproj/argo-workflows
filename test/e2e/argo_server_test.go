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
	s.Run("CreateGoodSA", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Create(goodSa)
		assert.NoError(t, err)
	})
	defer func() {
		// Clean up created sa
		_ = s.KubeClient.CoreV1().ServiceAccounts(nsName).Delete(goodSaName, nil)
	}()

	// Create bad serviceaccount
	badSaName := "argotestbad"
	badSa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: badSaName}}
	s.Run("CreateBadSA", func(t *testing.T) {
		_, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Create(badSa)
		assert.NoError(t, err)
	})
	defer func() {
		_ = s.KubeClient.CoreV1().ServiceAccounts(nsName).Delete(badSaName, nil)
	}()

	// Create RBAC Role
	var roleName string
	s.Run("LoadRoleYaml", func(t *testing.T) {
		obj, err := fixtures.LoadObject("@testdata/argo-server-test-role.yaml")
		assert.NoError(t, err)
		role, _ := obj.(*rbacv1.Role)
		roleName = role.Name
		_, err = s.KubeClient.RbacV1().Roles(nsName).Create(role)
		assert.NoError(t, err)
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
	s.Run("CreateRoleBinding", func(t *testing.T) {
		_, err := s.KubeClient.RbacV1().RoleBindings(nsName).Create(roleBinding)
		assert.NoError(t, err)
	})
	defer func() {
		_ = s.KubeClient.RbacV1().RoleBindings(nsName).Delete(roleBindingName, nil)
	}()

	// Sleep 2 seconds to wait for serviceaccount token created.
	// The secret creation slowness is seen in k3d.
	time.Sleep(2 * time.Second)

	// Get token of good serviceaccount
	var goodToken string
	s.Run("GetGoodSAToken", func(t *testing.T) {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(goodSaName, metav1.GetOptions{})
		if assert.NoError(t, err) {
			secretName := sAccount.Secrets[0].Name
			secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(secretName, metav1.GetOptions{})
			assert.NoError(t, err)
			goodToken = string(secret.Data["token"])
		}
	})

	// Get token of bad serviceaccount
	var badToken string
	s.Run("GetBadSAToken", func(t *testing.T) {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(badSaName, metav1.GetOptions{})
		assert.NoError(t, err)
		secretName := sAccount.Secrets[0].Name
		secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(secretName, metav1.GetOptions{})
		assert.NoError(t, err)
		badToken = string(secret.Data["token"])
	})

	token := s.bearerToken
	defer func() { s.bearerToken = token }()

	// Test creating workflow with good token
	var uid string
	s.bearerToken = goodToken
	s.Run("CreateWFGoodToken", func(t *testing.T) {
		uid = s.e(t).POST("/api/v1/workflows/" + nsName).
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
	s.Run("ListWFsGoodToken", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflows/"+nsName).
			WithQuery("listOptions.labelSelector", "argo-e2e").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Gt(0)
	})

	// Test creating workflow with bad token
	s.bearerToken = badToken
	s.Run("CreateWFBadToken", func(t *testing.T) {
		s.e(t).POST("/api/v1/workflows/" + nsName).
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
	s.Run("ListWFsBadToken", func(t *testing.T) {
		s.e(t).GET("/api/v1/workflows/" + nsName).
			Expect().
			Status(403)
	})

	if s.Persistence.IsEnabled() {

		// Simply wait 10 seconds for the wf to be completed
		s.Given().
			WorkflowName("test-wf-good").
			When().
			WaitForWorkflow(10 * time.Second)

		// Test list archived WFs with good token
		s.bearerToken = goodToken
		s.Run("ListArchivedWFsGoodToken", func(t *testing.T) {
			s.e(t).GET("/api/v1/archived-workflows").
				WithQuery("listOptions.labelSelector", "argo-e2e").
				WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
				Expect().
				Status(200).
				JSON().
				Path("$.items").
				Array().Length().Gt(0)
		})

		// Test get archived wf with good token
		s.bearerToken = goodToken
		s.Run("GetArchivedWFsGoodToken", func(t *testing.T) {
			s.e(t).GET("/api/v1/archived-workflows/"+uid).
				WithQuery("listOptions.labelSelector", "argo-e2e").
				Expect().
				Status(200).
				JSON().
				Path("$.metadata.name").
				Equal("test-wf-good")
		})

		// Test list archived WFs with bad token
		// TODO: Uncomment following code after https://github.com/argoproj/argo/issues/2049 is resolved.

		// s.bearerToken = badToken
		// s.Run("ListArchivedWFsBadToken", func(t *testing.T) {
		// 	s.e(t).GET("/api/v1/archived-workflows").
		// 		WithQuery("listOptions.labelSelector", "argo-e2e").
		// 		WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
		// 		Expect().
		// 		Status(200).
		// 		JSON().
		// 		Path("$.items").
		// 		Array().
		// 		Length().
		// 		Equal(0)
		// })

		// Test get archived wf with bad token
		s.bearerToken = badToken
		s.Run("ListArchivedWFsBadToken", func(t *testing.T) {
			s.e(t).GET("/api/v1/archived-workflows/"+uid).
				WithQuery("listOptions.labelSelector", "argo-e2e").
				Expect().
				Status(403)
		})

	}

	// Test delete workflow with bad token
	s.bearerToken = badToken
	s.Run("DeleteWFWithBadToken", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
			Expect().
			Status(403)
	})

	// Test delete workflow with good token
	s.bearerToken = goodToken
	s.Run("DeleteWFWithGoodToken", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
			Expect().
			Status(200)
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
}

func (s *ArgoServerSuite) TestWorkflowService() {

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

	s.Run("List", func(t *testing.T) {
		s.Given().
			WorkflowName("test").
			When().
			WaitForWorkflowToStart(20 * time.Second)

		j := s.e(t).GET("/api/v1/workflows/argo").
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

	s.Run("Get", func(t *testing.T) {
		j := s.e(t).GET("/api/v1/workflows/argo/test").
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

func (s *ArgoServerSuite) TestCronWorkflowService() {
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

	s.Run("List", func(t *testing.T) {
		// make sure list options work correctly
		s.Given().
			CronWorkflow("@testdata/basic.yaml")

		s.e(t).GET("/api/v1/cron-workflows/argo").
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

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/cron-workflows/argo/test").
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
		WaitForWorkflow(15 * time.Second).
		Then().
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})

	s.Run("GetArtifact", func(t *testing.T) {
		s.e(t).GET("/artifacts/argo/basic/basic/main-logs").
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})
	s.Run("GetArtifactByUID", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflows/argo/basic").
			Expect().
			Status(200)

		s.e(t).GET("/artifacts-by-uid/{uid}/basic/main-logs", uid).
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})

	// as the artifact server has some special code for cookies, we best test that too
	s.Run("GetArtifactByUIDUsingCookie", func(t *testing.T) {
		token := s.bearerToken
		defer func() { s.bearerToken = token }()
		s.bearerToken = ""
		s.e(t).GET("/artifacts-by-uid/{uid}/basic/main-logs", uid).
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
	s.Run("Watch", func(t *testing.T) {
		req, err := http.NewRequest("GET", baseUrl+"/api/v1/workflow-events/argo?listOptions.fieldSelector=metadata.name=basic", nil)
		assert.NoError(t, err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
		req.Close = true
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer func() {
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()
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
					if strings.Contains(line, ":) Hello Argo!") {
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

func (s *ArgoServerSuite) TestArchivedWorkflowService() {
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
		Expect(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
		})
	s.Run("List", func(t *testing.T) {
		s.Given().
			Workflow("@smoke/basic-2.yaml").
			When().
			SubmitWorkflow().
			WaitForWorkflow(20 * time.Second)

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

func (s *ArgoServerSuite) TestWorkflowTemplateService() {

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

	s.Run("Delete", func(t *testing.T) {
		s.e(t).DELETE("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200)
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
