package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/persist/sqldb/mocks"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	v1alpha "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/server/auth"
	testutil "github.com/argoproj/argo/test/util"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/common"
)

const unlabelled = `{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "Workflow",
  "metadata": {
    "namespace": "workflows",
    "name": "unlabelled",
    "labels": {
      "workflows.argoproj.io/phase": "Failed"
    }
  },
  "spec": {
    "entrypoint": "whalesay",
    "templates": [
      {
        "container": {
          "image": "docker/whalesay:latest"
        },
        "name": "whalesay"
      }
    ]
  },
  "status": {
    "phase": "Failed"
  }
}
`

const wf1 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid",
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-9tql2",
        "namespace": "workflows",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}
`
const wf2 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T19:12:55Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid",
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-b6h5m",
        "namespace": "workflows",
        "resourceVersion": "52919656",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-b6h5m",
        "uid": "91066a6c-1ddc-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T19:12:59Z",
        "nodes": {
            "hello-world-b6h5m": {
                "displayName": "hello-world-b6h5m",
                "finishedAt": "2019-12-13T19:12:58Z",
                "id": "hello-world-b6h5m",
                "name": "hello-world-b6h5m",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T19:12:55Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T19:12:55Z"
    }
}
`
const wf3 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid",
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-9tql2-test",
        "namespace": "test",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}
`
const wf4 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T19:12:55Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid",
            "workflows.argoproj.io/completed": "true",
            "workflows.argoproj.io/phase": "Succeeded"
        },
        "name": "hello-world-b6h5m-test",
        "namespace": "test",
        "resourceVersion": "52919656",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-b6h5m",
        "uid": "91066a6c-1ddc-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T19:12:59Z",
        "nodes": {
            "hello-world-b6h5m": {
                "displayName": "hello-world-b6h5m",
                "finishedAt": "2019-12-13T19:12:58Z",
                "id": "hello-world-b6h5m",
                "name": "hello-world-b6h5m",
                "phase": "Succeeded",
                "startedAt": "2019-12-13T19:12:55Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Succeeded",
        "startedAt": "2019-12-13T19:12:55Z"
    }
}
`
const wf5 = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "creationTimestamp": "2019-12-13T23:36:32Z",
        "generateName": "hello-world-",
        "generation": 5,
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid",
            "workflows.argoproj.io/completed": "false",
            "workflows.argoproj.io/phase": "Running"
        },
        "name": "hello-world-9tql2-run",
        "namespace": "workflows",
        "resourceVersion": "53020772",
        "selfLink": "/apis/argoproj.io/v1alpha1/namespaces/workflows/workflows/hello-world-9tql2",
        "uid": "6522aff1-1e01-11ea-b443-42010aa80075"
    },
    "spec": {
        "arguments": {},
        "entrypoint": "whalesay",
        "templates": [
            {
                "arguments": {},
                "container": {
                    "args": [
                        "hello world"
                    ],
                    "command": [
                        "cowsay"
                    ],
                    "image": "docker/whalesay:latest",
                    "name": "",
                    "resources": {}
                },
                "inputs": {},
                "metadata": {},
                "name": "whalesay",
                "outputs": {}
            }
        ]
    },
    "status": {
        "finishedAt": "2019-12-13T23:36:40Z",
        "nodes": {
            "hello-world-9tql2": {
                "displayName": "hello-world-9tql2-run",
                "finishedAt": "2019-12-13T23:36:39Z",
                "id": "hello-world-9tql2",
                "name": "hello-world-9tql2",
                "phase": "Running",
                "startedAt": "2019-12-13T23:36:32Z",
                "templateName": "whalesay",
                "type": "Pod"
            }
        },
        "phase": "Running",
        "startedAt": "2019-12-13T23:36:32Z"
    }
}
`

const failedWf = `
{
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
        "name": "failed",
        "namespace": "workflows",
        "labels": {
            "workflows.argoproj.io/controller-instanceid": "my-instanceid"
        }
    },
    "spec": {
        "entrypoint": "whalesay",
        "templates": [
            {
                "container": {
                    "image": "docker/whalesay:latest"
                },
                "name": "whalesay"
            }
        ]
    },
    "status": {
        "phase": "Failed"
    }
}
`
const workflow1 = `
{
  "namespace": "default",
  "workflow": {
    "apiVersion": "argoproj.io/v1alpha1",
    "kind": "Workflow",
    "metadata": {
	  "generateName": "hello-world-"
    },
    "spec": {
      "entrypoint": "whalesay",
      "templates": [
        {
          "name": "whalesay",
          "container": {
            "image": "docker/whalesay:latest",
            "command": [
              "cowsay"
            ],
            "args": [
              "hello world"
            ]
          }
        }
      ]
    }
  }
}
`
const workflowtmpl = `
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "WorkflowTemplate",
  "metadata": {
    "name": "workflow-template-whalesay-template",
    "namespace": "workflows"
  },
  "spec": {
    "entrypoint": "whalesay-template",
    "arguments": {
      "parameters": [
        {
          "name": "message",
          "value": "hello world"
        }
      ]
    },
    "templates": [
      {
        "name": "whalesay-template",
        "inputs": {
          "parameters": [
            {
              "name": "message"
            }
          ]
        },
        "container": {
          "image": "docker/whalesay",
          "command": [
            "cowsay"
          ],
          "args": [
            "{{inputs.parameters.message}}"
          ]
        }
      }
    ]
  }
}
`
const cronwf = `
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "CronWorkflow",
  "metadata": {
    "name": "hello-world",
	"namespace": "workflows"
  },
  "spec": {
    "schedule": "* * * * *",
    "timezone": "America/Los_Angeles",
    "startingDeadlineSeconds": 0,
    "concurrencyPolicy": "Replace",
    "successfulJobsHistoryLimit": 4,
    "failedJobsHistoryLimit": 4,
    "suspend": false,
    "workflowSpec": {
      "entrypoint": "whalesay",
      "templates": [
        {
          "name": "whalesay",
          "container": {
            "image": "docker/whalesay:latest",
            "command": [
              "cowsay"
            ],
            "args": [
              "🕓 hello world"
            ]
          }
        }
      ]
    }
  }
}
`
const clusterworkflowtmpl = `
{
  "apiVersion": "argoproj.io/v1alpha1",
  "kind": "ClusterWorkflowTemplate",
  "metadata": {
    "name": "cluster-workflow-template-whalesay-template",
    "namespace": "workflows"
  },
  "spec": {
    "entrypoint": "whalesay-template",
    "arguments": {
      "parameters": [
        {
          "name": "message",
          "value": "hello world"
        }
      ]
    },
    "templates": [
      {
        "name": "whalesay-template",
        "inputs": {
          "parameters": [
            {
              "name": "message"
            }
          ]
        },
        "container": {
          "image": "docker/whalesay",
          "command": [
            "cowsay"
          ],
          "args": [
            "{{inputs.parameters.message}}"
          ]
        }
      }
    ]
  }
}
`

func getWorkflowServer() (workflowpkg.WorkflowServiceServer, context.Context) {

	var unlabelledObj, wfObj1, wfObj2, wfObj3, wfObj4, wfObj5, failedWfObj v1alpha1.Workflow
	var wftmpl v1alpha1.WorkflowTemplate
	var cwfTmpl v1alpha1.ClusterWorkflowTemplate
	var cronwfObj v1alpha1.CronWorkflow

	testutil.MustUnmarshallJSON(unlabelled, &unlabelledObj)
	testutil.MustUnmarshallJSON(wf1, &wfObj1)
	testutil.MustUnmarshallJSON(wf1, &wfObj1)
	testutil.MustUnmarshallJSON(wf2, &wfObj2)
	testutil.MustUnmarshallJSON(wf3, &wfObj3)
	testutil.MustUnmarshallJSON(wf4, &wfObj4)
	testutil.MustUnmarshallJSON(wf5, &wfObj5)
	testutil.MustUnmarshallJSON(failedWf, &failedWfObj)
	testutil.MustUnmarshallJSON(workflowtmpl, &wftmpl)
	testutil.MustUnmarshallJSON(cronwf, &cronwfObj)
	testutil.MustUnmarshallJSON(clusterworkflowtmpl, &cwfTmpl)

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("List", mock.Anything).Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{}, nil)
	server := NewWorkflowServer(instanceid.NewService("my-instanceid"), offloadNodeStatusRepo)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := v1alpha.NewSimpleClientset(&unlabelledObj, &wfObj1, &wfObj2, &wfObj3, &wfObj4, &wfObj5, &failedWfObj, &wftmpl, &cronwfObj, &cwfTmpl)
	wfClientset.PrependReactor("create", "workflows", generateNameReactor)
	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet)
	return server, ctx
}

// generateNameReactor implements the logic required for the GenerateName field to work when using
// the fake client. Add it with client.PrependReactor to your fake client.
func generateNameReactor(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
	wf := action.(ktesting.CreateAction).GetObject().(*v1alpha1.Workflow)
	if wf.Name == "" && wf.GenerateName != "" {
		wf.Name = fmt.Sprintf("%s%s", wf.GenerateName, rand.String(5))
	}
	return false, nil, nil
}

func getWorkflow(ctx context.Context, server workflowpkg.WorkflowServiceServer, namespace string, wfName string) (*v1alpha1.Workflow, error) {
	return server.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{Name: wfName, Namespace: namespace})
}

func getWorkflowList(ctx context.Context, server workflowpkg.WorkflowServiceServer, namespace string) (*v1alpha1.WorkflowList, error) {
	return server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: namespace})
}

func TestCreateWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	var req workflowpkg.WorkflowCreateRequest
	testutil.MustUnmarshallJSON(workflow1, &req)
	wf, err := server.CreateWorkflow(ctx, &req)
	if assert.NoError(t, err) {
		assert.NotNil(t, wf)
		assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
	}
}

type testWatchWorkflowServer struct {
	testServerStream
}

func (t testWatchWorkflowServer) Send(*workflowpkg.WorkflowWatchEvent) error {
	panic("implement me")
}

func TestWatchWorkflows(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.NodeSucceeded},
	}
	assert.NoError(t, json.Unmarshal([]byte(wf1), &wf))
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := server.WatchWorkflows(&workflowpkg.WatchWorkflowsRequest{}, &testWatchWorkflowServer{testServerStream{ctx}})
		assert.EqualError(t, err, "context canceled")
	}()
	cancel()
}

func TestGetWorkflowWithNotFound(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		wf, err := getWorkflow(ctx, server, "test", "NotFound")
		if assert.Error(t, err) {
			assert.Nil(t, wf)
		}
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := getWorkflow(ctx, server, "test", "unlabelled")
		assert.Error(t, err)
	})
}

func TestListWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wfl, err := getWorkflowList(ctx, server, "workflows")
	if assert.NoError(t, err) {
		assert.NotNil(t, wfl)
		assert.Equal(t, 4, len(wfl.Items))
	}
	wfl, err = getWorkflowList(ctx, server, "test")
	if assert.NoError(t, err) {
		assert.NotNil(t, wfl)
		assert.Equal(t, 2, len(wfl.Items))
	}
}

func TestDeleteWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		delRsp, err := server.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: "hello-world-b6h5m", Namespace: "workflows"})
		if assert.NoError(t, err) {
			assert.NotNil(t, delRsp)
		}
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: "unlabelled", Namespace: "workflows"})
		assert.Error(t, err)
	})
}

func TestRetryWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		retried, err := server.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{Name: "failed", Namespace: "workflows"})
		if assert.NoError(t, err) {
			assert.NotNil(t, retried)
		}
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{Name: "unlabelled", Namespace: "workflows"})
		assert.Error(t, err)
	})
}

func TestSuspendResumeWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf, err := server.SuspendWorkflow(ctx, &workflowpkg.WorkflowSuspendRequest{Name: "hello-world-9tql2-run", Namespace: "workflows"})
	if assert.NoError(t, err) {
		assert.NotNil(t, wf)
		assert.Equal(t, true, *wf.Spec.Suspend)
		wf, err = server.ResumeWorkflow(ctx, &workflowpkg.WorkflowResumeRequest{Name: wf.Name, Namespace: wf.Namespace})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.Nil(t, wf.Spec.Suspend)
		}
	}
}

func TestSuspendResumeWorkflowWithNotFound(t *testing.T) {
	server, ctx := getWorkflowServer()

	susWfReq := workflowpkg.WorkflowSuspendRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err := server.SuspendWorkflow(ctx, &susWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
	rsmWfReq := workflowpkg.WorkflowResumeRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err = server.ResumeWorkflow(ctx, &rsmWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
}

func TestTerminateWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()

	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2-run")
	assert.Nil(t, err)
	rsmWfReq := workflowpkg.WorkflowTerminateRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	}
	wf, err = server.TerminateWorkflow(ctx, &rsmWfReq)
	assert.NotNil(t, wf)
	assert.Equal(t, v1alpha1.ShutdownStrategyTerminate, wf.Spec.Shutdown)
	assert.Nil(t, err)

	rsmWfReq = workflowpkg.WorkflowTerminateRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err = server.TerminateWorkflow(ctx, &rsmWfReq)
	assert.Nil(t, wf)
	assert.NotNil(t, err)
}

func TestStopWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2-run")
	assert.NoError(t, err)
	rsmWfReq := workflowpkg.WorkflowStopRequest{Name: wf.Name, Namespace: wf.Namespace}
	wf, err = server.StopWorkflow(ctx, &rsmWfReq)
	if assert.NoError(t, err) {
		assert.NotNil(t, wf)
		assert.Equal(t, v1alpha1.NodeRunning, wf.Status.Phase)
	}
}

func TestResubmitWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		wf, err := server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{Name: "hello-world-9tql2", Namespace: "workflows"})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
		}
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{Name: "unlabelled", Namespace: "workflows"})
		assert.Error(t, err)
	})
}

func TestLintWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{}
	testutil.MustUnmarshallJSON(unlabelled, &wf)
	linted, err := server.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Workflow: wf})
	if assert.NoError(t, err) {
		assert.NotNil(t, linted)
		assert.Contains(t, linted.Labels, common.LabelKeyControllerInstanceID)
	}
}

type testPodLogsServer struct {
	testServerStream
}

func (t testPodLogsServer) Send(*workflowpkg.LogEntry) error {
	panic("implement me")
}

func TestPodLogs(t *testing.T) {
	server, ctx := getWorkflowServer()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := server.PodLogs(&workflowpkg.WorkflowLogRequest{
			Name:       "hello-world-9tql2",
			Namespace:  "workflows",
			LogOptions: &corev1.PodLogOptions{},
		}, &testPodLogsServer{testServerStream{ctx}})
		assert.NoError(t, err)
	}()
	cancel()
}

func TestSubmitWorkflowFromResource(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("SubmitFromWorkflowTemplate", func(t *testing.T) {
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "workflowtemplate",
			ResourceName: "workflow-template-whalesay-template",
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		}
	})
	t.Run("SubmitFromCronWorkflow", func(t *testing.T) {
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "cronworkflow",
			ResourceName: "hello-world",
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		}
	})
	t.Run("SubmitFromClusterWorkflowTemplate", func(t *testing.T) {
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "ClusterWorkflowTemplate",
			ResourceName: "cluster-workflow-template-whalesay-template",
		})
		if assert.NoError(t, err) {
			assert.NotNil(t, wf)
			assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		}
	})
}
