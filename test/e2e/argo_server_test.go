//go:build api
// +build api

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/secrets"

	"github.com/gavv/httpexpect/v2"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const baseUrl = "http://localhost:2746"

// ensure basic HTTP functionality works,
// testing behaviour really is a non-goal
type ArgoServerSuite struct {
	fixtures.E2ESuite
	username    string
	bearerToken string
}

func (s *ArgoServerSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	var err error
	s.bearerToken, err = s.GetServiceAccountToken()
	s.CheckError(err)
}

func (s *ArgoServerSuite) e() *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseUrl,
			Reporter: httpexpect.NewRequireReporter(s.T()),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(s.T(), true),
			},
			Client: httpClient,
		}).
		Builder(func(req *httpexpect.Request) {
			if s.username != "" {
				req.WithBasicAuth(s.username, "garbage")
			} else if s.bearerToken != "" {
				req.WithHeader("Authorization", "Bearer "+s.bearerToken)
			}
		})
}

func (s *ArgoServerSuite) TestInfo() {
	s.Run("Get", func() {
		json := s.e().GET("/api/v1/info").
			Expect().
			Status(200).
			JSON()
		json.
			Path("$.managedNamespace").
			Equal("argo")
		json.
			Path("$.links[0].name").
			Equal("Workflow Link")
		json.
			Path("$.links[0].scope").
			Equal("workflow")
		json.
			Path("$.links[0].url").
			Equal("http://logging-facility?namespace=${metadata.namespace}&workflowName=${metadata.name}&startedAt=${status.startedAt}&finishedAt=${status.finishedAt}")
	})
}

func (s *ArgoServerSuite) TestVersion() {
	s.Run("Version", func() {
		s.e().GET("/api/v1/version").
			Expect().
			Status(200).
			JSON().
			Path("$.version").
			NotNull()
	})
}

func (s *ArgoServerSuite) TestMetricsForbidden() {
	s.bearerToken = ""
	s.e().
		GET("/metrics").
		Expect().
		Status(403)
}

func (s *ArgoServerSuite) TestMetricsOK() {
	body := s.e().
		GET("/metrics").
		Expect().
		Status(200).
		Body()
	body.
		// https://blog.netsil.com/the-4-golden-signals-of-api-health-and-performance-in-cloud-native-applications-a6e87526e74
		// Latency: The time it takes to service a request, with a focus on distinguishing between the latency of successful requests and the latency of failed requests
		Contains(`grpc_server_handling_seconds_bucket`).
		// Traffic: A measure of how much demand is being placed on the service. This is measured using a high-level service-specific metric, like HTTP requests per second in the case of an HTTP REST API.
		Contains(`promhttp_metric_handler_requests_in_flight`).
		// Errors: The rate of requests that fail. The failures can be explicit (e.g., HTTP 500 errors) or implicit (e.g., an HTTP 200 OK response with a response body having too few items).
		Contains(`promhttp_metric_handler_requests_total{code="500"}`)

	if os.Getenv("CI") == "true" {
		body.
			// Saturation: How “full” is the service. This is a measure of the system utilization, emphasizing the resources that are most constrained (e.g., memory, I/O or CPU). Services degrade in performance as they approach high saturation.
			Contains(`process_cpu_seconds_total`).
			Contains(`process_resident_memory_bytes`)
	}
}

func (s *ArgoServerSuite) TestSubmitWorkflowTemplateFromGithubWebhook() {
	s.bearerToken = ""

	data, err := os.ReadFile("testdata/github-webhook-payload.json")
	assert.NoError(s.T(), err)

	s.Given().
		WorkflowTemplate(`
metadata:
  name: github-webhook
spec:
  entrypoint: main
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  templates:
    - name: main
      container:
         image: argoproj/argosay:v2
`).
		WorkflowEventBinding(`
metadata:
  name: github-webhook
spec:
  event:
    selector: metadata["x-github-event"] == ["push"]
  submit:
    workflowTemplateRef:
      name: github-webhook
`).
		When().
		CreateWorkflowTemplates().
		CreateWorkflowEventBinding().
		And(func() {
			s.e().
				POST("/api/v1/events/argo/").
				WithHeader("X-Github-Event", "push").
				WithHeader("X-Hub-Signature", "sha1=c09e61386e81c2669e015049350500448148205c").
				WithBytes(data).
				Expect().
				Status(200)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, _ *wfv1.WorkflowStatus) {
			assert.Equal(t, "github-webhook", metadata.GetLabels()[common.LabelKeyWorkflowTemplate])
		})
}

func (s *ArgoServerSuite) TestSubmitWorkflowTemplateFromEvent() {
	s.Given().
		WorkflowTemplate(`
metadata:
  name: event-consumer
spec:
  entrypoint: main
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  arguments:
    parameters:
      - name: salutation
        value: "hello"
  templates:
    - name: main
      steps:
      - - name: a
          template: argosay
          arguments:
            parameters:
            - name: salutation
              value: "{{workflow.parameters.salutation}}"
            - name: appellation
              value: "{{workflow.parameters.appellation}}"

    - name: argosay
      inputs:
        parameters:
          - name: salutation
          - name: appellation
      container:
         image: argoproj/argosay:v2
         args: [echo, "{{inputs.parameters.salutation}} {{inputs.parameters.appellation}}"]
`).
		WorkflowEventBinding(`
metadata:
  name: event-consumer
spec:
  event:
    selector: payload.appellation != "" && metadata["x-argo-e2e"] == ["true"]
  submit:
    workflowTemplateRef:
      name: event-consumer
    arguments:
      parameters:
        - name: appellation
          valueFrom:
            event: payload.appellation
`).
		When().
		CreateWorkflowEventBinding().
		CreateWorkflowTemplates().
		And(func() {
			s.e().
				POST("/api/v1/events/argo/").
				WithHeader("X-Argo-E2E", "true").
				WithBytes([]byte(`{"appellation": "Mr Chips"}`)).
				Expect().
				Status(200)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, _ *wfv1.WorkflowStatus) {
			assert.Equal(t, "event-consumer", metadata.GetLabels()[common.LabelKeyWorkflowTemplate])
		})
}

func (s *ArgoServerSuite) TestSubmitClusterWorkflowTemplateFromEvent() {
	s.Given().
		ClusterWorkflowTemplate(`
metadata:
  name: event-consumer
spec:
  entrypoint: main
  workflowMetadata:
    labels:
      workflows.argoproj.io/test: "true"
  templates:
    - name: main
      container:
         image: argoproj/argosay:v2
`).
		WorkflowEventBinding(`
metadata:
  name: event-consumer
spec:
  event:
    selector: true
  submit:
    workflowTemplateRef:
      name: event-consumer
      clusterScope: true
`).
		When().
		CreateWorkflowEventBinding().
		CreateClusterWorkflowTemplates().
		And(func() {
			s.e().
				POST("/api/v1/events/argo/").
				WithBytes([]byte(`{}`)).
				Expect().
				Status(200)
		}).
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, _ *wfv1.WorkflowStatus) {
			assert.Equal(t, "event-consumer", metadata.GetLabels()[common.LabelKeyClusterWorkflowTemplate])
		})
}

func (s *ArgoServerSuite) TestEventOnMalformedWorkflowEventBinding() {
	s.Given().
		WorkflowEventBinding(`
metadata:
  name: malformed
`).
		When().
		CreateWorkflowEventBinding().
		And(func() {
			s.e().
				POST("/api/v1/events/argo/").
				WithBytes([]byte(`{}`)).
				Expect().
				Status(500)
		}).
		Then().
		ExpectAuditEvents(
			func(event corev1.Event) bool {
				return event.InvolvedObject.Name == "malformed" && event.InvolvedObject.Kind == workflow.WorkflowEventBindingKind
			}, 1,
			func(t *testing.T, e []corev1.Event) {
				assert.Equal(t, "argo", e[0].InvolvedObject.Namespace)
				assert.Equal(t, "WorkflowEventBindingError", e[0].Reason)
				assert.Equal(t, "failed to dispatch event: failed to evaluate workflow template expression: unable to evaluate expression '': unexpected token EOF (1:1)", e[0].Message)
			},
		)
}

func (s *ArgoServerSuite) TestGetUserInfo() {
	s.e().GET("/api/v1/userinfo").
		Expect().
		Status(200)
}

// we can only really tests these endpoint respond, not worthwhile checking more
func (s *ArgoServerSuite) TestOauth() {
	s.Run("Redirect", func() {
		s.e().GET("/oauth2/redirect").
			Expect().
			Status(501)
	})
	s.Run("Callback", func() {
		s.e().GET("/oauth2/callback").
			Expect().
			Status(501)
	})
}

func (s *ArgoServerSuite) TestUnauthorized() {
	token := s.bearerToken
	s.T().Run("Bearer", func(t *testing.T) {
		s.bearerToken = "test-token"
		defer func() { s.bearerToken = token }()
		s.e().GET("/api/v1/workflows/argo").
			Expect().
			Status(401)
	})
	s.T().Run("Basic", func(t *testing.T) {
		s.username = "garbage"
		defer func() { s.username = "" }()
		s.e().GET("/api/v1/workflows/argo").
			Expect().
			Status(401)
	})
}

func (s *ArgoServerSuite) TestCookieAuth() {
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = ""
	s.e().GET("/api/v1/workflows/argo").
		WithHeader("Cookie", "authorization=Bearer "+token).
		Expect().
		Status(200)
}

// You could have multiple authorization headers, set by wildcard domain cookies in the case of some SSO implementations
func (s *ArgoServerSuite) TestMultiCookieAuth() {
	token := s.bearerToken
	defer func() { s.bearerToken = token }()
	s.bearerToken = ""
	s.e().GET("/api/v1/workflows/argo").
		WithCookie("authorization", "invalid1").
		WithCookie("authorization", "Bearer "+token).
		WithCookie("authorization", "invalid2").
		Expect().
		Status(200)
}

func (s *ArgoServerSuite) createServiceAccount(name string) {
	ctx := context.Background()
	_, err := s.KubeClient.CoreV1().ServiceAccounts(fixtures.Namespace).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	secret, err := s.KubeClient.CoreV1().Secrets(fixtures.Namespace).Create(ctx, secrets.NewTokenSecret(name), metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	s.T().Cleanup(func() {
		_ = s.KubeClient.CoreV1().Secrets(fixtures.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		_ = s.KubeClient.CoreV1().ServiceAccounts(fixtures.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func (s *ArgoServerSuite) TestPermission() {
	ctx := context.Background()
	nsName := fixtures.Namespace
	goodSaName := "argotestgood"
	s.createServiceAccount(goodSaName)
	badSaName := "argotestbad"
	s.createServiceAccount(badSaName)

	// Create RBAC Role
	var roleName string
	s.Run("LoadRoleYaml", func() {
		obj, err := fixtures.LoadObject("@testdata/argo-server-test-role.yaml")
		assert.NoError(s.T(), err)
		role, _ := obj.(*rbacv1.Role)
		roleName = role.Name
		_, err = s.KubeClient.RbacV1().Roles(nsName).Create(ctx, role, metav1.CreateOptions{})
		assert.NoError(s.T(), err)
	})
	defer func() {
		_ = s.KubeClient.RbacV1().Roles(nsName).Delete(ctx, roleName, metav1.DeleteOptions{})
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
		_, err := s.KubeClient.RbacV1().RoleBindings(nsName).Create(ctx, roleBinding, metav1.CreateOptions{})
		assert.NoError(s.T(), err)
	})
	defer func() {
		_ = s.KubeClient.RbacV1().RoleBindings(nsName).Delete(ctx, roleBindingName, metav1.DeleteOptions{})
	}()

	// Sleep 2 seconds to wait for serviceaccount token created.
	// The secret creation slowness is seen in k3d.
	time.Sleep(2 * time.Second)

	// Get token of good serviceaccount
	var goodToken string
	s.Run("GetGoodSAToken", func() {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(ctx, goodSaName, metav1.GetOptions{})
		if assert.NoError(s.T(), err) {
			secretName := secrets.TokenNameForServiceAccount(sAccount)
			secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(ctx, secretName, metav1.GetOptions{})
			assert.NoError(s.T(), err)
			goodToken = string(secret.Data["token"])
		}
	})

	// Get token of bad serviceaccount
	var badToken string
	s.Run("GetBadSAToken", func() {
		sAccount, err := s.KubeClient.CoreV1().ServiceAccounts(nsName).Get(ctx, badSaName, metav1.GetOptions{})
		assert.NoError(s.T(), err)
		secretName := secrets.TokenNameForServiceAccount(sAccount)
		secret, err := s.KubeClient.CoreV1().Secrets(nsName).Get(ctx, secretName, metav1.GetOptions{})
		assert.NoError(s.T(), err)
		badToken = string(secret.Data["token"])
	})

	token := s.bearerToken
	defer func() { s.bearerToken = token }()

	// Test creating workflow with good token
	var uid string
	s.bearerToken = goodToken
	s.Run("CreateWFGoodToken", func() {
		uid = s.e().POST("/api/v1/workflows/" + nsName).
			WithBytes([]byte(`{
  "workflow": {
    "metadata": {
      "name": "test-wf-good",
      "labels": {
         "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "main",
          "container": {
            "image": "argoproj/argosay:v2"
          }
        }
      ],
      "entrypoint": "main"
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
	s.Run("ListWFsGoodToken", func() {
		s.e().GET("/api/v1/workflows/"+nsName).
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

	s.Given().
		When().
		WaitForWorkflow(fixtures.ToBeArchived)

	// Test creating workflow with bad token
	s.bearerToken = badToken
	s.Run("CreateWFBadToken", func() {
		s.e().POST("/api/v1/workflows/" + nsName).
			WithBytes([]byte(`{
  "workflow": {
    "metadata": {
      "name": "test-wf-bad",
      "labels": {
         "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "main",
          "container": {
            "image": "argoproj/argosay:v2"
          }
        }
      ],
      "entrypoint": "main"
    }
  }
}`)).
			Expect().
			Status(403)
	})

	// Test list workflows with bad token
	s.Run("ListWFsBadToken", func() {
		s.e().GET("/api/v1/workflows/" + nsName).
			Expect().
			Status(403)
	})
	// Test delete workflow with bad token
	s.Run("DeleteWFWithBadToken", func() {
		s.e().DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
			Expect().
			Status(403)
	})

	// Test delete workflow with good token
	s.bearerToken = goodToken
	s.Run("DeleteWFWithGoodToken", func() {
		s.e().DELETE("/api/v1/workflows/" + nsName + "/test-wf-good").
			Expect().
			Status(200)
	})

	// we've now deleted the workflow, but it is still in the archive, testing the archive
	// after deleting the workflow makes sure that we are no dependant of the workflow for authorization

	// Test list archived WFs with good token
	s.Run("ListArchivedWFsGoodToken", func() {
		s.e().GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().Length().Gt(0)
	})

	s.bearerToken = badToken
	s.Run("ListArchivedWFsBadToken", func() {
		s.e().GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			WithQuery("listOptions.fieldSelector", "metadata.namespace="+nsName).
			Expect().
			Status(403)
	})

	// Test get archived wf with good token
	s.bearerToken = goodToken
	s.Run("GetArchivedWFsGoodToken", func() {
		s.e().GET("/api/v1/archived-workflows/"+uid).
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			Expect().
			Status(200)
	})

	// Test get archived wf with bad token
	s.bearerToken = badToken
	s.Run("GetArchivedWFsBadToken", func() {
		s.e().GET("/api/v1/archived-workflows/" + uid).
			Expect().
			Status(403)
	})

	// Test deleting archived wf with bad token
	s.Run("DeleteArchivedWFsBadToken", func() {
		s.e().DELETE("/api/v1/archived-workflows/" + uid).
			Expect().
			Status(403)
	})

	// Test deleting archived wf with good token
	s.bearerToken = goodToken
	s.Run("DeleteArchivedWFsGoodToken", func() {
		s.e().DELETE("/api/v1/archived-workflows/" + uid).
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestLintWorkflow() {
	s.e().POST("/api/v1/workflows/argo/lint").
		WithBytes([]byte((`{
  "workflow": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "argoproj/argosay:v2",
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

func (s *ArgoServerSuite) TestHintWhenWorkflowExists() {
	s.e().POST("/api/v1/workflows/argo").
		WithBytes([]byte((`{
  "workflow": {
    "metadata": {
      "name": "hint",
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "entrypoint": "whalesay",
      "templates": [
        {
          "name": "whalesay",
          "container": {
            "image": "argoproj/argosay:v2"
          }
        }
      ]
    }
  }
}`))).
		Expect().
		Status(200)

	s.e().POST("/api/v1/workflows/argo").
		WithBytes([]byte((`{
  "workflow": {
    "metadata": {
      "name": "hint",
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "entrypoint": "whalesay",
      "templates": [
        {
          "name": "whalesay",
          "container": {
            "image": "argoproj/argosay:v2"
          }
        }
      ]
    }
  }
}`))).
		Expect().
		Status(409).
		Body().
		Contains("already exists")
}

func (s *ArgoServerSuite) TestCreateWorkflowDryRun() {
	s.e().POST("/api/v1/workflows/argo").
		WithBytes([]byte(`{
  "createOptions": {
    "dryRun": ["All"]
  },
  "workflow": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "argoproj/argosay:v2",
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
	var name string
	s.Run("Create", func() {
		name = s.e().POST("/api/v1/workflows/argo").
			WithBytes([]byte(`{
  "workflow": {
    "metadata": {
      "generateName": "test-",
      "labels": {
         "workflows.argoproj.io/test": "subject-1"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "image": "argoproj/argosay:v2",
            "args": ["sleep", "10s"]   
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
			Path("$.metadata.name").
			NotNull().
			String().
			Raw()
	})

	s.Given().
		When().
		WaitForWorkflow(fixtures.ToBeRunning)

	s.Run("List", func() {
		j := s.e().GET("/api/v1/workflows/argo").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test=subject-1").
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
		j.Path("$.items[0].status.nodes").
			NotNull()
	})

	s.Run("ListWithFields", func() {
		j := s.e().GET("/api/v1/workflows/argo").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test=subject-1").
			WithQuery("fields", "-items.status.nodes,items.status.finishedAt,items.status.startedAt").
			Expect().
			Status(200).
			JSON()
		j.Path("$.metadata").
			NotNull()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
		j.Path("$.items[0].status").Object().ContainsKey("phase").NotContainsKey("nodes")
	})

	s.Run("Get", func() {
		j := s.e().GET("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200).
			JSON()
		j.Path("$.status.nodes").
			NotNull()
		s.e().GET("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})

	s.Run("GetWithFields", func() {
		j := s.e().GET("/api/v1/workflows/argo/"+name).
			WithQuery("fields", "status.phase").
			Expect().
			Status(200).
			JSON()
		j.Path("$.status").Object().ContainsKey("phase").NotContainsKey("nodes")
	})

	s.Run("Suspend", func() {
		s.e().PUT("/api/v1/workflows/argo/" + name + "/suspend").
			Expect().
			Status(200)

		s.e().GET("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200).
			JSON().
			Path("$.spec.suspend").
			Equal(true)
	})

	s.Run("Resume", func() {
		s.e().PUT("/api/v1/workflows/argo/" + name + "/resume").
			Expect().
			Status(200)

		s.e().GET("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200).
			JSON().
			Path("$.spec").
			Object().
			NotContainsKey("suspend")
	})

	s.Run("Terminate", func() {
		s.e().PUT("/api/v1/workflows/argo/" + name + "/terminate").
			Expect().
			Status(200)

		s.Given().
			WorkflowName(name).
			When().
			WaitForWorkflow()

		s.e().GET("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200).
			JSON().
			Path("$.status.message").
			Equal("Stopped with strategy 'Terminate'")
	})

	s.Run("Resubmit", func() {
		s.e().PUT("/api/v1/workflows/argo/" + name + "/resubmit").
			WithBytes([]byte(`{"memoized": true}`)).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			NotNull()
	})

	s.Run("Delete", func() {
		s.e().DELETE("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200)
		s.e().DELETE("/api/v1/workflows/argo/not-found").
			Expect().
			Status(404)
	})
}

func (s *ArgoServerSuite) TestCronWorkflowService() {
	s.Run("Create", func() {
		s.e().POST("/api/v1/cron-workflows/argo").
			WithBytes([]byte(`{
  "cronWorkflow": {
    "metadata": {
      "name": "test",
      "labels": {
        "workflows.argoproj.io/test": "subject-2"
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
              "image": "argoproj/argosay:v2",
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

	s.Run("Suspend", func() {
		s.e().PUT("/api/v1/cron-workflows/argo/test/suspend").
			Expect().
			Status(200).
			JSON().
			Path("$.spec.suspend").
			Equal(true)
	})

	s.Run("Resume", func() {
		s.e().PUT("/api/v1/cron-workflows/argo/test/resume").
			Expect().
			Status(200).
			JSON().
			Path("$.spec").
			Object().
			NotContainsKey("suspend")
	})

	s.Run("List", func() {
		// make sure list options work correctly
		s.Given().
			CronWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron-wf-basic
spec:
  schedule: "* * * * *"
  concurrencyPolicy: "Allow"
  startingDeadlineSeconds: 0
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  workflowSpec:
    podGC:
      strategy: OnPodCompletion
    entrypoint: whalesay
    templates:
      - name: whalesay
        container:
          image: argoproj/argosay:v2
          imagePullPolicy: IfNotPresent
          command: ["sh", -c]
          args: ["echo hello"]
`)

		s.e().GET("/api/v1/cron-workflows/argo").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test=subject-2").
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
		s.e().GET("/api/v1/cron-workflows/argo/not-found").
			Expect().
			Status(404)
		resourceVersion = s.e().GET("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func() {
		s.e().PUT("/api/v1/cron-workflows/argo/test").
			WithBytes([]byte(`{"cronWorkflow": {
    "metadata": {
      "name": "test",
      "resourceVersion": "` + resourceVersion + `",
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "schedule": "1 * * * *",
      "workflowMetadata": {
        "labels": {"workflows.argoproj.io/test": "true"}
      },
      "workflowSpec": {
        "entrypoint": "whalesay",
        "templates": [
          {
            "name": "whalesay",
            "container": {
              "image": "argoproj/argosay:v2",
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
		s.e().DELETE("/api/v1/cron-workflows/argo/test").
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestArtifactServerArchivedWorkflow() {
	var uid types.UID
	var nodeID string
	s.Given().
		Workflow(`@testdata/artifact-passing-workflow.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
			nodeID = status.Nodes.FindByDisplayName("generate-artifact").ID
		})

	// In this case, the artifact name is a file
	s.Run("GetArtifactByNodeID", func() {
		s.e().GET("/artifact-files/argo/archived-workflows/{uid}/{nodeID}/outputs/hello", uid, nodeID).
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})
}

func (s *ArgoServerSuite) TestArtifactServerArchivedStoppedWorkflow() {
	var uid types.UID
	var nodeID string
	s.Given().
		Workflow(`@testdata/artifact-workflow-stopped.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
			nodeID = status.Nodes.FindByDisplayName("create-artifact").ID
		})

	s.Run("GetLocalArtifactByNodeID", func() {
		s.e().GET("/artifact-files/argo/archived-workflows/{uid}/{nodeID}/outputs/local-artifact", uid, nodeID).
			Expect().
			Status(200).
			Body().
			Contains("testing")
	})

	s.Run("GetGlobalArtifactByNodeID", func() {
		s.e().GET("/artifact-files/argo/archived-workflows/{uid}/{nodeID}/outputs/global-artifact", uid, nodeID).
			Expect().
			Status(200).
			Body().
			Contains("testing global")
	})
}

// make sure we can download an artifact
func (s *ArgoServerSuite) TestArtifactServer() {
	var uid types.UID
	var name string
	s.Given().
		Workflow(`@testdata/artifact-workflow.yaml`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
			uid = metadata.UID
		})

	s.artifactServerRetrievalTests(name, uid)
}

func (s *ArgoServerSuite) artifactServerRetrievalTests(name string, uid types.UID) {
	s.Run("GetArtifact", func() {
		resp := s.e().GET("/artifacts/argo/" + name + "/" + name + "/main-file").
			Expect().
			Status(200)

		resp.Body().
			Contains(":) Hello Argo!")

		resp.Header("Content-Security-Policy").
			Equal("sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self' 'unsafe-inline'")

		resp.Header("X-Frame-Options").
			Equal("SAMEORIGIN")
	})

	// In this case, the artifact name is a file
	s.Run("GetArtifactFile", func() {
		resp := s.e().GET("/artifact-files/argo/workflows/" + name + "/" + name + "/outputs/main-file").
			Expect().
			Status(200)

		resp.Body().
			Contains(":) Hello Argo!")

		resp.Header("Content-Security-Policy").
			Equal("sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self' 'unsafe-inline'")

		resp.Header("X-Frame-Options").
			Equal("SAMEORIGIN")
	})

	// In this case, the artifact name is a directory
	s.Run("GetArtifactFileDirectory", func() {
		resp := s.e().GET("/artifact-files/argo/workflows/" + name + "/" + name + "/outputs/out/").
			Expect().
			Status(200)

		resp.Body().
			Contains("<a href=\"subdirectory/\">subdirectory/</a>")

	})

	// In this case, the filename specified in the request is actually a directory
	s.Run("GetArtifactFileSubdirectory", func() {
		resp := s.e().GET("/artifact-files/argo/workflows/" + name + "/" + name + "/outputs/out/subdirectory/").
			Expect().
			Status(200)

		resp.Body().
			Contains("<a href=\"sub-file-1\">sub-file-1</a>").
			Contains("<a href=\"sub-file-2\">sub-file-2</a>")

	})

	// In this case, the filename specified in the request is a subdirectory file
	s.Run("GetArtifactSubfile", func() {
		resp := s.e().GET("/artifact-files/argo/workflows/" + name + "/" + name + "/outputs/out/subdirectory/sub-file-1").
			Expect().
			Status(200)

		resp.Body().
			Contains(":) Hello Argo!")

		resp.Header("Content-Security-Policy").
			Equal("sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self' 'unsafe-inline'")

		resp.Header("X-Frame-Options").
			Equal("SAMEORIGIN")
	})

	// In this case, the artifact name is a file
	s.Run("GetArtifactBadFile", func() {
		_ = s.e().GET("/artifact-files/argo/workflows/" + name + "/" + name + "/outputs/not-a-file").
			Expect().
			Status(500)
	})

	s.Run("GetArtifactByUID", func() {
		s.e().DELETE("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200)

		s.e().GET("/artifacts-by-uid/{uid}/{name}/main-file", uid, name).
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
		s.e().GET("/artifacts-by-uid/{uid}/{name}/main-file", uid, name).
			WithHeader("Cookie", "authorization=Bearer "+token).
			Expect().
			Status(200)
	})

	s.Run("GetArtifactFileByUID", func() {
		s.e().GET("/artifact-files/argo/archived-workflows/{uid}/{name}/outputs/main-file", uid, name).
			Expect().
			Status(200).
			Body().
			Contains(":) Hello Argo!")
	})
}

func (s *ArgoServerSuite) stream(url string, f func(t *testing.T, line string) (done bool)) {
	t := s.T()
	req, err := http.NewRequest("GET", baseUrl+url, nil)
	assert.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	req.Close = true
	resp, err := httpClient.Do(req)
	assert.NoError(t, err)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	if t.Failed() {
		t.FailNow()
	}
	if f == nil {
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		log.WithField("line", line).Debug()
		// make sure we have this enabled
		if line == "" {
			continue
		}
		if f(t, line) || t.Failed() {
			return
		}
	}
}

// do some basic testing on the stream methods
func (s *ArgoServerSuite) TestWorkflowServiceStream() {
	var name string
	s.Given().
		Workflow("@smoke/basic.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToStart).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			name = metadata.Name
		})

	// use the watch to make sure that the workflow has succeeded
	s.Run("Watch", func() {
		s.stream("/api/v1/workflow-events/argo?listOptions.fieldSelector=metadata.name="+name, func(t *testing.T, line string) (done bool) {
			if strings.Contains(line, `status:`) {
				assert.Contains(t, line, `"offloadNodeStatus":true`)
				// so that we get this
				assert.Contains(t, line, `"nodes":`)
			}
			return strings.Contains(line, "Succeeded")
		})
	})

	// then,  lets see what events we got
	s.Run("WatchEvents", func() {
		s.stream("/api/v1/stream/events/argo?listOptions.fieldSelector=involvedObject.kind=Workflow,involvedObject.name="+name, func(t *testing.T, line string) (done bool) {
			return strings.Contains(line, "WorkflowRunning")
		})
	})

	// then,  lets check the logs
	for _, tt := range []struct {
		name string
		path string
	}{
		{"PodLogs", "/" + name + "/log?logOptions.container=main&logOptions.tailLines=3"},
		{"WorkflowLogs", "/log?podName=" + name + "&logOptions.container=main&logOptions.tailLines=3"},
	} {
		s.Run(tt.name, func() {
			s.stream("/api/v1/workflows/argo/"+name+tt.path, func(t *testing.T, line string) (done bool) {
				if strings.Contains(line, "data: ") {
					assert.Contains(t, line, fmt.Sprintf(`"podName":"%s"`, name))
					return true
				}
				return false
			})
		})
	}
}

func (s *ArgoServerSuite) TestArchivedWorkflowService() {
	var uid types.UID
	var name string
	s.Given().
		Workflow(`
metadata:
  generateName: archie-
  labels:
    foo: 1
spec:
  entrypoint: run-archie
  templates:
    - name: run-archie
      container:
        image: argoproj/argosay:v2`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			uid = metadata.UID
			name = metadata.Name
		})
	var failedUid types.UID
	var failedName string
	s.Given().
		Workflow(`
metadata:
  generateName: jughead-
  labels:
    foo: 3
spec:
  entrypoint: run-jughead
  templates:
    - name: run-jughead
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args: ["echo intentional failure; exit 1"]`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			failedUid = metadata.UID
			failedName = metadata.Name
		})
	s.Given().
		Workflow(`
metadata:
  generateName: betty-
  labels:
    foo: 2
spec:
  entrypoint: run-betty
  templates:
    - name: run-betty
      container:
        image: argoproj/argosay:v2`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeArchived)

	for _, tt := range []struct {
		name     string
		selector string
		wantLen  int
	}{
		{"ListDoesNotExist", "!foo", 0},
		{"ListEquals", "foo=1", 1},
		{"ListDoubleEquals", "foo==1", 1},
		{"ListIn", "foo in (1)", 1},
		{"ListNotEquals", "foo!=1", 2},
		{"ListNotIn", "foo notin (1)", 2},
		{"ListExists", "foo", 3},
		{"ListGreaterThan0", "foo>0", 3},
		{"ListGreaterThan1", "foo>1", 2},
		{"ListLessThan1", "foo<1", 0},
		{"ListLessThan2", "foo<2", 1},
	} {
		s.Run(tt.name, func() {
			path := s.e().GET("/api/v1/archived-workflows").
				WithQuery("listOptions.fieldSelector", "metadata.namespace=argo").
				WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test,"+tt.selector).
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
		j := s.e().GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
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
		s.e().GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
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
		s.e().GET("/api/v1/archived-workflows").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			WithQuery("listOptions.fieldSelector", "metadata.namespace=argo,spec.startedAt>"+time.Now().Add(1*time.Hour).Format(time.RFC3339)).
			WithQuery("listOptions.limit", 2).
			Expect().
			Status(200).
			JSON().
			Path("$.items").Null()
	})

	s.Run("Get", func() {
		s.e().GET("/api/v1/archived-workflows/not-found").
			Expect().
			Status(404)
		j := s.e().GET("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.metadata.name").
			NotNull()
		j.
			Path(fmt.Sprintf("$.metadata.labels[\"%s\"]", common.LabelKeyWorkflowArchivingStatus)).
			Equal("Persisted")
		s.e().GET("/api/v1/workflows/argo/" + name).
			Expect().
			Status(200).
			JSON().
			Path(fmt.Sprintf("$.metadata.labels[\"%s\"]", common.LabelKeyWorkflowArchivingStatus)).
			Equal("Archived")
	})

	s.Run("DeleteForRetry", func() {
		s.e().DELETE("/api/v1/workflows/argo/" + failedName).
			Expect().
			Status(200)
	})

	s.Run("Retry", func() {
		s.e().PUT("/api/v1/archived-workflows/{uid}/retry", failedUid).
			WithBytes([]byte(`{"namespace": "argo"}`)).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			NotNull()
		s.e().PUT("/api/v1/archived-workflows/{uid}/retry", failedUid).
			WithBytes([]byte(`{"namespace": "argo"}`)).
			Expect().
			Status(409)
	})

	s.Run("Resubmit", func() {
		s.e().PUT("/api/v1/archived-workflows/{uid}/resubmit", uid).
			WithBytes([]byte(`{"namespace": "argo", "memoized": false}`)).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			NotNull()
	})

	s.Run("Delete", func() {
		s.e().DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(200)
		s.e().DELETE("/api/v1/archived-workflows/{uid}", uid).
			Expect().
			Status(404)
	})

	s.Run("ListLabelKeys", func() {
		j := s.e().GET("/api/v1/archived-workflows-label-keys").
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Gt(0)
	})

	s.Run("ListLabelValues", func() {
		j := s.e().GET("/api/v1/archived-workflows-label-values").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test").
			Expect().
			Status(200).
			JSON()
		j.
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})

}

func (s *ArgoServerSuite) TestWorkflowTemplateService() {
	s.Run("Lint", func() {
		s.e().POST("/api/v1/workflow-templates/argo/lint").
			WithBytes([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "argoproj/argosay:v2",
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
		s.e().POST("/api/v1/workflow-templates/argo").
			WithBytes([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "subject-3"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "argoproj/argosay:v2",
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

		s.e().GET("/api/v1/workflow-templates/argo").
			WithQuery("listOptions.labelSelector", "workflows.argoproj.io/test=subject-3").
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
		s.e().GET("/api/v1/workflow-templates/argo/not-found").
			Expect().
			Status(404)

		resourceVersion = s.e().GET("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			String().
			Raw()
	})

	s.Run("Update", func() {
		s.e().PUT("/api/v1/workflow-templates/argo/test").
			WithBytes([]byte(`{"template": {
    "metadata": {
      "name": "test",
      "resourceVersion": "` + resourceVersion + `",
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "argoproj/argosay:v2",
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
			Equal("argoproj/argosay:v2")
	})

	s.Run("Delete", func() {
		s.e().DELETE("/api/v1/workflow-templates/argo/test").
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestSubmitWorkflowFromResource() {
	s.Run("CreateWFT", func() {
		s.e().POST("/api/v1/workflow-templates/argo").
			WithBytes([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "subject-4"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "argoproj/argosay:v2",
            "imagePullPolicy": "IfNotPresent"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)).Expect().Status(200)
	})

	s.Run("SubmitWFT", func() {
		s.e().POST("/api/v1/workflows/argo/submit").
			WithBytes([]byte(`{
			  "resourceKind": "WorkflowTemplate",
			  "resourceName": "test",
			  "submitOptions": {
                "labels": "workflows.argoproj.io/test=true"
              }
			}`)).
			Expect().
			Status(200)
	})

	s.Run("CreateCronWF", func() {
		s.e().POST("/api/v1/cron-workflows/argo").
			WithBytes([]byte(`{
  "cronWorkflow": {
    "metadata": {
      "name": "test",
      "labels": {
        "workflows.argoproj.io/test": "subject-5"
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
              "image": "argoproj/argosay:v2",
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
	s.Run("SubmitWFT", func() {
		s.e().POST("/api/v1/workflows/argo/submit").
			WithBytes([]byte(`{
			  "resourceKind": "cronworkflow",
			  "resourceName": "test",
			  "submitOptions": {
                "labels": "workflows.argoproj.io/test=true"
              }
			}`)).
			Expect().
			Status(200)
	})

	s.Run("CreateCWFT", func() {
		s.e().POST("/api/v1/cluster-workflow-templates").
			WithBytes([]byte(`{
  "template": {
    "metadata": {
      "name": "test",
      "labels": {
         "workflows.argoproj.io/test": "subject-6"
      }
    },
    "spec": {
      "templates": [
        {
          "name": "run-workflow",
          "container": {
            "name": "",
            "image": "argoproj/argosay:v2",
            "imagePullPolicy": "IfNotPresent"
          }
        }
      ],
      "entrypoint": "run-workflow"
    }
  }
}`)).Expect().Status(200)
	})

	s.Run("SubmitCWFT", func() {
		s.e().POST("/api/v1/workflows/argo/submit").
			WithBytes([]byte(`{
			  "resourceKind": "ClusterWorkflowTemplate",
			  "resourceName": "test",
			  "submitOptions": {
                "labels": "workflows.argoproj.io/test=true"
              }
			}`)).
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestEventSourcesService() {
	s.Run("CreateEventSource", func() {
		s.e().POST("/api/v1/event-sources/argo").
			WithBytes([]byte(`
{
  "eventsource": {
    "metadata": {
      "name": "test-event-source", 
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "calendar": {
        "example-with-interval": {
          "interval": "10s"
        }
      }
    }
  }
}
`)).
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			NotNull().
			String().
			Raw()
	})
	s.Run("ListEventSources", func() {
		s.e().GET("/api/v1/event-sources/argo").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})
	s.Run("WatchEventSources", func() {
		s.stream("/api/v1/stream/event-sources/argo", func(t *testing.T, line string) (done bool) {
			assert.Contains(t, line, "test-event-source")
			return true
		})
	})
	s.Run("EventSourcesLogs", func() {
		s.T().Skip("we do not install the controllers, so we won't get any logs")
		s.stream("/api/v1/stream/event-sources/argo/logs", func(t *testing.T, line string) (done bool) {
			assert.Contains(t, line, "test-event-source")
			return true
		})
	})
	var resourceVersion string
	s.Run("GetEventSource", func() {
		resourceVersion = s.e().GET("/api/v1/event-sources/argo/test-event-source").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.resourceVersion").
			NotNull().
			String().
			Raw()
	})
	s.Run("UpdateEventSource", func() {
		s.e().PUT("/api/v1/event-sources/argo/test-event-source").
			WithBytes([]byte(`
{
  "eventsource": {
    "metadata": {
      "name": "test-event-source", 
      "resourceVersion": "` + resourceVersion + `",
      "labels": {
        "workflows.argoproj.io/test": "true"
      }
    },
    "spec": {
      "calendar": {
        "example-with-interval": {
          "interval": "10s"
        }
      }
    }
  }
}
`)).
			Expect().
			Status(200)
	})
	s.Run("DeleteEventSource", func() {
		s.e().DELETE("/api/v1/event-sources/argo/test-event-source").
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestSensorService() {
	s.Run("CreateSensor", func() {
		s.e().POST("/api/v1/sensors/argo").
			WithBytes([]byte(`
{
	"sensor":{
		"metadata":{
			"name":"test-sensor",
			"labels": {
				"workflows.argoproj.io/test": "true"
			}
		},
		"spec":{
			"dependencies":[
				{
					"name":"test-dep",
					"eventSourceName":"calendar",
					"eventName":"example-with-interval"
				}
			],
			"triggers":[
				{
					"template":{
						"name":"log-trigger",
						"log":{
							"intervalSeconds":20
						}
					}
				}
			]
		}
	}
}
`)).Expect().
			Status(200)
	})
	s.Run("ListSensors", func() {
		s.e().GET("/api/v1/sensors/argo").
			Expect().
			Status(200).
			JSON().
			Path("$.items").
			Array().
			Length().
			Equal(1)
	})
	s.Run("GetSensor", func() {
		s.e().GET("/api/v1/sensors/argo/test-sensor").
			Expect().
			Status(200).
			JSON().
			Path("$.metadata.name").
			Equal("test-sensor")
	})
	s.Run("WatchSensors", func() {
		s.stream("/api/v1/stream/sensors/argo", func(t *testing.T, line string) (done bool) {
			assert.Contains(t, line, "test-sensor")
			return true
		})
	})
	s.Run("SensorsLogs", func() {
		s.T().Skip("we do not install the controllers, so we won't get any logs")
		s.stream("/api/v1/stream/sensors/argo/logs", func(t *testing.T, line string) (done bool) {
			assert.Contains(t, line, "test-sensor")
			return true
		})
	})
	resourceVersion := s.e().GET("/api/v1/sensors/argo/test-sensor").
		Expect().
		Status(200).
		JSON().
		Path("$.metadata.resourceVersion").
		String().
		Raw()
	s.Run("UpdateSensor", func() {
		s.e().PUT("/api/v1/sensors/argo/test-sensor").
			WithBytes([]byte(`
{
	"sensor":{
		"metadata":{
			"name":"test-sensor",
			"resourceVersion": "` + resourceVersion + `",
			"labels": {
				"workflows.argoproj.io/test": "true"
			}
		},
		"spec": {
			"template": {
				"serviceAccountName": "default"
			},
			"dependencies":[
				{
					"name":"test-dep",
					"eventSourceName":"calendar",
					"eventName":"example-with-interval"
				}
			],
			"triggers":[
				{
					"template":{
						"name":"log-trigger",
						"log":{
							"intervalSeconds":20
						}
					}
				}
			]
		}
	}
}
`)).
			Expect().
			Status(200)
	})
	s.Run("GetSensorAfterUpdating", func() {
		s.e().GET("/api/v1/sensors/argo/test-sensor").
			Expect().
			Status(200).
			JSON().
			Path("$.spec.template.serviceAccountName").
			Equal("default")
	})
	s.Run("DeleteSensor", func() {
		s.e().DELETE("/api/v1/sensors/argo/test-sensor").
			Expect().
			Status(200)
	})
}

func (s *ArgoServerSuite) TestRateLimitHeader() {
	s.Run("GetRateLimit", func() {
		resp := s.e().GET("/api/v1/version").
			Expect().
			Status(200)

		resp.Header("X-RateLimit-Limit").NotEmpty()
		resp.Header("X-RateLimit-Remaining").NotEmpty()
		resp.Header("X-RateLimit-Reset").NotEmpty()
		resp.Header("Retry-After").Empty()
	})
}

func TestArgoServerSuite(t *testing.T) {
	suite.Run(t, new(ArgoServerSuite))
}
