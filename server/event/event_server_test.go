package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	fakekube "k8s.io/client-go/kubernetes/fake"

	eventpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
)

func TestController(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.WithValue(context.TODO(), auth.WfKey, clientset)
	instanceIDService := instanceid.NewService("my-instanceid")
	eventRecorderManager := events.NewEventRecorderManager(fakekube.NewSimpleClientset())
	newController := func(asyncDispatch bool) *Controller {
		return NewController(instanceIDService, eventRecorderManager, 1, 1, asyncDispatch)
	}
	e1 := &eventpkg.EventRequest{Namespace: "my-ns", Payload: &wfv1.Item{}}
	e2 := &eventpkg.EventRequest{}
	t.Run("Async", func(t *testing.T) {
		s := newController(true)

		_, err := s.ReceiveEvent(ctx, e1)
		require.NoError(t, err)

		assert.Len(t, s.operationQueue, 1, "one event to be processed")

		_, err = s.ReceiveEvent(ctx, e2)
		require.EqualError(t, err, "rpc error: code = Unavailable desc = operation queue full", "backpressure when queue is full")

		stopCh := make(chan struct{}, 1)
		stopCh <- struct{}{}
		s.Run(stopCh)

		assert.Empty(t, s.operationQueue, "all events were processed")
	})
	t.Run("Sync", func(t *testing.T) {
		s := newController(false)

		_, err := s.ReceiveEvent(ctx, e1)
		require.NoError(t, err)
		_, err = s.ReceiveEvent(ctx, e2)
		require.NoError(t, err)
	})
	t.Run("SyncError", func(t *testing.T) {
		s := newController(false)

		_, err := s.ReceiveEvent(ctx, &eventpkg.EventRequest{Namespace: "my-ns", Payload: &wfv1.Item{Value: json.RawMessage("!")}})
		require.EqualError(t, err, "rpc error: code = Internal desc = failed to create workflow template expression environment: json: error calling MarshalJSON for type *v1alpha1.Item: invalid character '!' looking for beginning of value")
	})
}
