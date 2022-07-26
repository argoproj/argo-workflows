package events

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
)

func TestCustomEventAggregatorFuncWithAnnotations(t *testing.T) {
	event := apiv1.Event{}
	key, msg := customEventAggregatorFuncWithAnnotations(&event)
	assert.Equal(t, "", key)
	assert.Equal(t, "", msg)

	event.Source = apiv1.EventSource{Component: "component1", Host: "host1"}
	event.InvolvedObject.Name = "name1"
	event.Message = "message1"

	key, msg = customEventAggregatorFuncWithAnnotations(&event)
	assert.Equal(t, "component1host1name1", key)
	assert.Equal(t, "message1", msg)

	event.ObjectMeta.Annotations = map[string]string{"key1": "val1", "key2": "val2"}
	key, msg = customEventAggregatorFuncWithAnnotations(&event)
	assert.Equal(t, "component1host1name1val1val2", key)
	assert.Equal(t, "message1", msg)
}
