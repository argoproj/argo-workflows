package workflow

import (
	"context"
	"fmt"

	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	v1alpha "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/server/workflow/store"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
        "entrypoint": "whalesay",
        "templates": [
            {
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
        "uid": "91066a6c-1ddc-11ea-b443-42010aa80074"
    },
    "spec": {

        "entrypoint": "whalesay",
        "templates": [
            {

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
        "uid": "6522aff1-1e01-11ea-b443-42010aa80074"
    },
    "spec": {

        "entrypoint": "whalesay",
        "templates": [
            {

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

        "entrypoint": "whalesay",
        "templates": [
            {

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
        "uid": "6522aff1-1e01-11ea-b443-42010aa80073"
    },
    "spec": {

        "entrypoint": "whalesay",
        "templates": [
            {

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
	"workflowMetadata": {
	 "Labels": {
		"labelTest": "test"
	 },
	 "annotations": {
		"annotationTest": "test"
	 }
	},
    "entrypoint": "whalesay-template",
    "arguments": {
      "parameters": [
        {
          "name": "message"
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
    "name": "cluster-workflow-template-whalesay-template"
  },
  "spec": {
	"workflowMetadata": {
	 "Labels": {
		"labelTest": "test"
	 },
	 "annotations": {
		"annotationTest": "test"
	 }
	},
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

	v1alpha1.MustUnmarshal(unlabelled, &unlabelledObj)
	v1alpha1.MustUnmarshal(wf1, &wfObj1)
	v1alpha1.MustUnmarshal(wf2, &wfObj2)
	v1alpha1.MustUnmarshal(wf3, &wfObj3)
	v1alpha1.MustUnmarshal(wf4, &wfObj4)
	v1alpha1.MustUnmarshal(wf5, &wfObj5)
	v1alpha1.MustUnmarshal(failedWf, &failedWfObj)
	v1alpha1.MustUnmarshal(workflowtmpl, &wftmpl)
	v1alpha1.MustUnmarshal(cronwf, &cronwfObj)
	v1alpha1.MustUnmarshal(clusterworkflowtmpl, &cwfTmpl)

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("List", mock.Anything).Return(map[sqldb.UUIDVersion]v1alpha1.Nodes{}, nil)

	archivedRepo := &mocks.WorkflowArchive{}

	archivedRepo.On("GetWorkflow", "", "test", "hello-world-9tql2-test").Return(&v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "hello-world-9tql2-test", Namespace: "test"},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "my-entrypoint",
			Templates: []v1alpha1.Template{
				{Name: "my-entrypoint", Container: &corev1.Container{}},
			},
		},
	}, nil)
	archivedRepo.On("GetWorkflow", "", "test", "not-found").Return(nil, nil)
	archivedRepo.On("GetWorkflow", "", "test", "unlabelled").Return(nil, nil)
	archivedRepo.On("GetWorkflow", "", "workflows", "latest").Return(nil, nil)
	archivedRepo.On("GetWorkflow", "", "workflows", "hello-world-9tql2-not").Return(nil, nil)
	r, err := labels.ParseToRequirements("workflows.argoproj.io/controller-instanceid=my-instanceid")
	if err != nil {
		panic(err)
	}
	archivedRepo.On("CountWorkflows", sutils.ListOptions{Namespace: "workflows", LabelRequirements: r}).Return(int64(2), nil)
	archivedRepo.On("ListWorkflows", sutils.ListOptions{Namespace: "workflows", Limit: -2, LabelRequirements: r}).Return(v1alpha1.Workflows{wfObj2, failedWfObj}, nil)
	archivedRepo.On("CountWorkflows", sutils.ListOptions{Namespace: "test", LabelRequirements: r}).Return(int64(1), nil)
	archivedRepo.On("ListWorkflows", sutils.ListOptions{Namespace: "test", Limit: -1, LabelRequirements: r}).Return(v1alpha1.Workflows{wfObj4}, nil)

	kubeClientSet := fake.NewSimpleClientset()
	kubeClientSet.PrependReactor("create", "selfsubjectaccessreviews", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})
	wfClientset := v1alpha.NewSimpleClientset(&unlabelledObj, &wfObj1, &wfObj2, &wfObj3, &wfObj4, &wfObj5, &failedWfObj, &wftmpl, &cronwfObj, &cwfTmpl)
	wfClientset.PrependReactor("create", "workflows", generateNameReactor)
	ctx := context.WithValue(context.WithValue(context.WithValue(context.TODO(), auth.WfKey, wfClientset), auth.KubeKey, kubeClientSet), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}})
	listOptions := &metav1.ListOptions{}
	instanceIdSvc := instanceid.NewService("my-instanceid")
	instanceIdSvc.With(listOptions)
	wfStore, err := store.NewSQLiteStore(instanceIdSvc)
	if err != nil {
		panic(err)
	}
	if err = wfStore.Add(&wfObj1); err != nil {
		panic(err)
	}
	if err = wfStore.Add(&wfObj3); err != nil {
		panic(err)
	}
	if err = wfStore.Add(&wfObj5); err != nil {
		panic(err)
	}
	namespaceAll := metav1.NamespaceAll
	server := NewWorkflowServer(instanceIdSvc, offloadNodeStatusRepo, archivedRepo, wfClientset, wfStore, wfStore, &namespaceAll)
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
	v1alpha1.MustUnmarshal(workflow1, &req)
	wf, err := server.CreateWorkflow(ctx, &req)
	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
	assert.Contains(t, wf.Labels, common.LabelKeyCreator)
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
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowSucceeded},
	}
	v1alpha1.MustUnmarshal([]byte(wf1), &wf)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := server.WatchWorkflows(&workflowpkg.WatchWorkflowsRequest{}, &testWatchWorkflowServer{testServerStream{ctx}})
		assert.NoError(t, err)
	}()
	cancel()
}

func TestWatchLatestWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowSucceeded},
	}
	v1alpha1.MustUnmarshal([]byte(wf1), &wf)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := server.WatchWorkflows(&workflowpkg.WatchWorkflowsRequest{
			ListOptions: &metav1.ListOptions{
				FieldSelector: util.GenerateFieldSelectorFromWorkflowName("@latest"),
			},
		}, &testWatchWorkflowServer{testServerStream{ctx}})
		assert.NoError(t, err)
	}()
	cancel()
}

func TestGetWorkflowWithNotFound(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		wf, err := getWorkflow(ctx, server, "test", "not-found")
		require.Error(t, err)
		assert.Nil(t, wf)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := getWorkflow(ctx, server, "test", "unlabelled")
		require.Error(t, err)
	})
}

func TestGetLatestWorkflow(t *testing.T) {
	_, ctx := getWorkflowServer()
	wfClient := ctx.Value(auth.WfKey).(versioned.Interface)
	wf, err := getLatestWorkflow(ctx, wfClient, "test")
	require.NoError(t, err)
	assert.Equal(t, "hello-world-9tql2-test", wf.Name)
}

func TestGetWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	s := server.(*workflowServer)
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, "test", "hello-world-9tql2-test", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, wf)
	wf, err = s.getWorkflow(ctx, wfClient, "test", "hello-world-9tql2-test", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestValidateWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	s := server.(*workflowServer)
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, "test", "hello-world-9tql2-test", metav1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, s.validateWorkflow(wf))
}

func TestListWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wfl, err := getWorkflowList(ctx, server, "workflows")
	require.NoError(t, err)
	assert.NotNil(t, wfl)
	assert.Len(t, wfl.Items, 4)
	wfl, err = getWorkflowList(ctx, server, "test")
	require.NoError(t, err)
	assert.NotNil(t, wfl)
	assert.Len(t, wfl.Items, 2)
}

func TestDeleteWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		delRsp, err := server.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: "hello-world-b6h5m", Namespace: "workflows"})
		require.NoError(t, err)
		assert.NotNil(t, delRsp)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: "unlabelled", Namespace: "workflows"})
		require.Error(t, err)
	})
	t.Run("Latest", func(t *testing.T) {
		_, err := server.DeleteWorkflow(ctx, &workflowpkg.WorkflowDeleteRequest{Name: "@latest", Namespace: "workflows"})
		require.NoError(t, err)
	})
}

func TestRetryWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		retried, err := server.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{Name: "failed", Namespace: "workflows"})
		require.NoError(t, err)
		assert.NotNil(t, retried)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{Name: "unlabelled", Namespace: "workflows"})
		require.Error(t, err)
	})
	t.Run("Latest", func(t *testing.T) {
		_, err := server.RetryWorkflow(ctx, &workflowpkg.WorkflowRetryRequest{Name: "latest", Namespace: "workflows"})
		require.Error(t, err)
	})
}

func TestSuspendResumeWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf, err := server.SuspendWorkflow(ctx, &workflowpkg.WorkflowSuspendRequest{Name: "hello-world-9tql2-run", Namespace: "workflows"})
	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.True(t, *wf.Spec.Suspend)
	wf, err = server.ResumeWorkflow(ctx, &workflowpkg.WorkflowResumeRequest{Name: wf.Name, Namespace: wf.Namespace})
	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Nil(t, wf.Spec.Suspend)
}

func TestSuspendResumeWorkflowWithNotFound(t *testing.T) {
	server, ctx := getWorkflowServer()

	susWfReq := workflowpkg.WorkflowSuspendRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err := server.SuspendWorkflow(ctx, &susWfReq)
	assert.Nil(t, wf)
	require.Error(t, err)
	rsmWfReq := workflowpkg.WorkflowResumeRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err = server.ResumeWorkflow(ctx, &rsmWfReq)
	assert.Nil(t, wf)
	require.Error(t, err)
}

func TestTerminateWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()

	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2-run")
	require.NoError(t, err)
	rsmWfReq := workflowpkg.WorkflowTerminateRequest{
		Name:      wf.Name,
		Namespace: wf.Namespace,
	}
	wf, err = server.TerminateWorkflow(ctx, &rsmWfReq)
	assert.NotNil(t, wf)
	assert.Equal(t, v1alpha1.ShutdownStrategyTerminate, wf.Spec.Shutdown)
	require.NoError(t, err)

	rsmWfReq = workflowpkg.WorkflowTerminateRequest{
		Name:      "hello-world-9tql2-not",
		Namespace: "workflows",
	}
	wf, err = server.TerminateWorkflow(ctx, &rsmWfReq)
	assert.Nil(t, wf)
	require.Error(t, err)
}

func TestStopWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf, err := getWorkflow(ctx, server, "workflows", "hello-world-9tql2-run")
	require.NoError(t, err)
	rsmWfReq := workflowpkg.WorkflowStopRequest{Name: wf.Name, Namespace: wf.Namespace}
	wf, err = server.StopWorkflow(ctx, &rsmWfReq)
	require.NoError(t, err)
	assert.NotNil(t, wf)
	assert.Equal(t, v1alpha1.WorkflowRunning, wf.Status.Phase)
}

func TestResubmitWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("Labelled", func(t *testing.T) {
		wf, err := server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{Name: "hello-world-9tql2", Namespace: "workflows"})
		require.NoError(t, err)
		assert.NotNil(t, wf)
	})
	t.Run("Unlabelled", func(t *testing.T) {
		_, err := server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{Name: "unlabelled", Namespace: "workflows"})
		require.Error(t, err)
	})
	t.Run("Latest", func(t *testing.T) {
		wf, err := server.ResubmitWorkflow(ctx, &workflowpkg.WorkflowResubmitRequest{Name: "@latest", Namespace: "workflows"})
		require.NoError(t, err)
		assert.NotNil(t, wf)
	})
}

func TestLintWorkflow(t *testing.T) {
	server, ctx := getWorkflowServer()
	wf := &v1alpha1.Workflow{}
	v1alpha1.MustUnmarshal(unlabelled, &wf)
	linted, err := server.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Workflow: wf})
	require.NoError(t, err)
	assert.NotNil(t, linted)
	assert.Contains(t, linted.Labels, common.LabelKeyControllerInstanceID)
	assert.Contains(t, linted.Labels, common.LabelKeyCreator)
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
			LogOptions: &corev1.PodLogOptions{Container: "main"},
		}, &testPodLogsServer{testServerStream{ctx}})
		assert.NoError(t, err)
	}()
	cancel()
}

func TestSubmitWorkflowFromResource(t *testing.T) {
	server, ctx := getWorkflowServer()
	t.Run("SubmitFromWorkflowTemplate fails if missing parameters", func(t *testing.T) {
		_, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "workflowtemplate",
			ResourceName: "workflow-template-whalesay-template",
		})
		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = spec.arguments.message.value or spec.arguments.message.valueFrom is required")
	})
	t.Run("SubmitFromWorkflowTemplate", func(t *testing.T) {
		opts := v1alpha1.SubmitOpts{
			Parameters: []string{
				"message=hello",
			},
		}
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:     "workflows",
			ResourceKind:  "workflowtemplate",
			ResourceName:  "workflow-template-whalesay-template",
			SubmitOptions: &opts,
		})
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, wf.Labels, common.LabelKeyCreator)
	})
	t.Run("SubmitFromCronWorkflow", func(t *testing.T) {
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "cronworkflow",
			ResourceName: "hello-world",
		})
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, wf.Labels, common.LabelKeyCreator)
	})
	t.Run("SubmitFromClusterWorkflowTemplate", func(t *testing.T) {
		wf, err := server.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
			Namespace:    "workflows",
			ResourceKind: "ClusterWorkflowTemplate",
			ResourceName: "cluster-workflow-template-whalesay-template",
		})
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID)
		assert.Contains(t, wf.Labels, common.LabelKeyCreator)
	})
}
