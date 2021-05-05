package event

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	fakekube "k8s.io/client-go/kubernetes/fake"

	eventpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
)

func TestController(t *testing.T) {
	wfeb := wfv1.WorkflowEventBinding{}
	wfebYAML := `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: event-consumer-5
  namespace: my-ns
  labels:
    "workflows.argoproj.io/controller-instanceid": "my-instanceid"
spec:
  event:
    selector: payload.discriminator == "test-discriminator"
  submit:
    workflowTemplateRef:
      name: hello-argoo
`
	wfv1.MustUnmarshal(wfebYAML, &wfeb)
	wfebList := &wfv1.WorkflowEventBindingList{
		Items: []wfv1.WorkflowEventBinding{wfeb},
	}

	clientset := fake.NewSimpleClientset(wfebList)

	s := NewController(instanceid.NewService("my-instanceid"), events.NewEventRecorderManager(fakekube.NewSimpleClientset()), 1, 1)

	payload := `{"discriminator": "test-discriminator"}`
	item, err := wfv1.ParseItem(payload)
	assert.NoError(t, err)

	ctx := context.WithValue(context.TODO(), auth.WfKey, clientset)
	_, err = s.ReceiveEvent(ctx, &eventpkg.EventRequest{Namespace: "my-ns", Payload: &item})
	assert.NoError(t, err)

	assert.Len(t, s.operationQueue, 1, "one event to be processed")

	_, err = s.ReceiveEvent(ctx, &eventpkg.EventRequest{Namespace: "my-ns", Payload: &item})
	assert.EqualError(t, err, "operation queue full", "backpressure when queue is full")

	stopCh := make(chan struct{}, 1)
	stopCh <- struct{}{}
	s.Run(stopCh)

	assert.Len(t, s.operationQueue, 0, "all events were processed")
}

func TestControllerNoWorkflowEventBinding(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	s := NewController(instanceid.NewService("my-instanceid"), events.NewEventRecorderManager(fakekube.NewSimpleClientset()), 1, 1)

	ctx := context.WithValue(context.TODO(), auth.WfKey, clientset)
	_, err := s.ReceiveEvent(ctx, &eventpkg.EventRequest{Namespace: "my-ns"})
	assert.Errorf(t, err, "failed to match any workflow event binding")
	assert.Len(t, s.operationQueue, 0)
}

func TestControllerWrongSelector(t *testing.T) {
	wfeb := wfv1.WorkflowEventBinding{}
	wfebYAML := `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: event-consumer-5
  namespace: my-ns
  labels:
    "workflows.argoproj.io/controller-instanceid": "my-instanceid"
spec:
  submit:
    workflowTemplateRef:
      name: hello-argoo
`
	tests := map[string]struct {
		selector    string
		payload     string
		expectedErr error
	}{
		"Wrong Discriminator": {
			selector:    `payload.discriminator == "wrong-discriminator-2"`,
			payload:     `{"discriminator": "test-discriminator"}`,
			expectedErr: fmt.Errorf("failed to match any workflow event binding"),
		},
		"Selector doesn't evaluate to bool": {
			selector:    `1 + 2 + 3`,
			payload:     `{}`,
			expectedErr: fmt.Errorf("failed to match any workflow event binding"),
		},
		"Selector wrong selector": {
			selector:    `(@)#*!@&(^@%@^*`,
			payload:     `{}`,
			expectedErr: fmt.Errorf("failed to match any workflow event binding"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			wfv1.MustUnmarshal(wfebYAML, &wfeb)
			wfeb.Spec.Event.Selector = tc.selector
			wfebList := &wfv1.WorkflowEventBindingList{
				Items: []wfv1.WorkflowEventBinding{wfeb},
			}
			clientset := fake.NewSimpleClientset(wfebList)

			s := NewController(instanceid.NewService("my-instanceid"), events.NewEventRecorderManager(fakekube.NewSimpleClientset()), 1, 1)
			payload := tc.payload
			item, err := wfv1.ParseItem(payload)
			assert.NoError(t, err)

			ctx := context.WithValue(context.TODO(), auth.WfKey, clientset)
			_, err = s.ReceiveEvent(ctx, &eventpkg.EventRequest{Namespace: "my-ns", Payload: &item})
			if tc.expectedErr == nil {
				assert.NoError(t, err)
				assert.Len(t, s.operationQueue, 1)
			} else {
				assert.Errorf(t, err, tc.expectedErr.Error())
				assert.Len(t, s.operationQueue, 0)
			}
		})
	}
}
