package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/util/instanceid"
)

func TestController(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	s := NewController(clientset, "my-ns", instanceid.NewService("my-instanceid"), 1, 1)

	_, err := s.ReceiveEvent(context.TODO(), &eventpkg.EventRequest{Namespace: "my-ns", Payload: &wfv1.Item{}})
	assert.NoError(t, err)

	assert.Len(t, s.operationPipeline, 1, "one event to be processed")

	_, err = s.ReceiveEvent(context.TODO(), &eventpkg.EventRequest{})
	assert.EqualError(t, err, "operation pipeline full", "backpressure when pipeline is full")

	stopCh := make(chan struct{}, 1)
	stopCh <- struct{}{}
	s.processEvents(stopCh)

	assert.Len(t, s.operationPipeline, 0, "all events were processed")

}
