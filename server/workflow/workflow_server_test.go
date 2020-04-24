package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/argoproj/pkg/errors"
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
	"github.com/argoproj/argo/util/instanceid"
)

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
	  "generateName": "hello-world-",
	  "labels": {
        "workflows.argoproj.io/controller-instanceid": "my-instanceid"
	  }
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

	var wfObj1, wfObj2, wfObj3, wfObj4, wfObj5, failedWfObj v1alpha1.Workflow
	var wftmpl v1alpha1.WorkflowTemplate
	var cwfTmpl v1alpha1.ClusterWorkflowTemplate
	var cronwfObj v1alpha1.CronWorkflow

	errors.CheckError(json.Unmarshal([]byte(wf1), &wfObj1))
	errors.CheckError(json.Unmarshal([]byte(wf2), &wfObj2))
	errors.CheckError(json.Unmarshal([]byte(wf3), &wfObj3))
	errors.CheckError(json.Unmarshal([]byte(wf4), &wfObj4))
	errors.CheckError(json.Unmarshal([]byte(wf5), &wfObj5))
	errors.CheckError(json.Unmarshal([]byte(failedWf), &failedWfObj))
	errors.CheckError(json.Unmarshal([]byte(workflowtmpl), &wftmpl))
	errors.CheckError(json.Unmarshal([]byte(cronwf), &cronwfObj))
	errors.CheckError(json.Unmarshal([]byte(clusterworkflowtmpl), &cwfTmpl))

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("List", mock.Anything).Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{}, nil)
	server := NewWorkflowServer(instanceid.NewService("my-instanceid"), offloadNodeStatusRepo)
	kubeClientSet := fake.NewSimpleClientset()
	wfClientset := v1alpha.NewSimpleClientset(&wfObj1, &wfObj2, &wfObj3, &wfObj4, &wfObj5, &failedWfObj, &wftmpl, &cronwfObj, &cwfTmpl)
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

	req := workflowpkg.WorkflowGetRequest{
		Name:      wfName,
		Namespace: namespace,
	}

	return server.GetWorkflow(ctx, &req)
}

func getWorkflowList(ctx context.Context, server workflowpkg.WorkflowServiceServer, namespace string) (*v1alpha1.WorkflowList, error) {
	return server.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{Namespace: namespace})

}

func TestCreateWorkflow(t *testing.T) {

	server, ctx := getWorkflowServer()
	var req workflowpkg.WorkflowCreateRequest
	_ = json.Unmarshal([]byte(workflow1), &req)

	wf, err := server.CreateWorkflow(ctx, &req)

	assert.NotNil(t, wf)
	assert.Nil(t, err)

}

type testWatchWorkflowServer struct {
	testServerStream
}

func (t testWatchWorkflowServer) Send(*workflowpkg.WorkflowWatchEvent) error {
	panic("implement me")
}

func TestWatchWorkflows(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{}
	assert.NoError(t, json.Unmarshal([]byte(wf1), &wf))
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := server.WatchWorkflows(&workflowpkg.WatchWorkflowsRequest{}, &testWatchWorkflowServer{testServerStream{ctx}})
		assert.NoError(t, err)
	}()
	cancel()
}

func TestGetWorkflowWithNotFound(t *testing.T) {

	server, ctx := getWorkflowServer()

	wf, err := getWorkflow(ctx, server, "test", "NotFound")
	assert.Nil(t, wf)
	assert.NotNil(t, err)

}

func TestListWorkflow(t *testing.T) {

	server, ctx := getWorkflowServer()

	wfl, err := getWorkflowList(ctx, server, "workflows")
	assert.NotNil(t, wfl)
	assert.Equal(t, 4, len(wfl.Items))
	assert.Nil(t, err)

	wfl, err = getWorkflowList(ctx, server, "test")
	assert.NotNil(t, wfl)
	assert.Equal(t, 2, len(wfl.Items))
	assert.Nil(t, err)
}

func TestDeleteWorkflow(t *testing.T) {

	server, ctx := getWorkflowServer()

	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-b6h5m")
	assert.Nil(t, err)
	delReq := workflowpkg.WorkflowDeleteRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	}

	delRsp, err := server.DeleteWorkflow(ctx, &delReq)

	assert.NoError(t, err)
	assert.NotNil(t, delRsp)

	wfl, err := getWorkflowList(ctx, server, "workflows")
	if assert.NoError(t, err) {
		assert.Len(t, wfl.Items, 3)
	}
}

func TestRetryWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	req := workflowpkg.WorkflowRetryRequest{Name: "failed", Namespace: "workflows"}
	retried, err := server.RetryWorkflow(ctx, &req)
	if assert.NoError(t, err) {
		assert.NotNil(t, retried)
	}
}

func TestSuspendResumeWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()

	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2-run")
	assert.Nil(t, err)
	susWfReq := workflowpkg.WorkflowSuspendRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	}
	wf, err = server.SuspendWorkflow(ctx, &susWfReq)
	assert.NotNil(t, wf)
	assert.Equal(t, true, *wf.Spec.Suspend)
	assert.Nil(t, err)
	rsmWfReq := workflowpkg.WorkflowResumeRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	}
	wf, err = server.ResumeWorkflow(ctx, &rsmWfReq)

	assert.NotNil(t, wf)
	assert.Nil(t, wf.Spec.Suspend)
	assert.Nil(t, err)
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
	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2")
	assert.Nil(t, err)
	wf, err = server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	})
	if assert.NoError(t, err) {
		assert.NotNil(t, wf)
	}
}

func TestLintWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{}
	assert.NoError(t, json.Unmarshal([]byte(wf1), &wf))
	linted, err := server.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Workflow: wf})
	if assert.NoError(t, err) {
		assert.NotNil(t, linted)
	}
}

type testPodLogsServer struct {
	testServerStream
}

func (t testPodLogsServer) Send(entry *workflowpkg.LogEntry) error {
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
		}
	})

}
