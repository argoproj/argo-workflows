package http1

import (
	corev1 "k8s.io/api/core/v1"
)

type eventWatchClient struct{ serverSentEventsClient }

func (f eventWatchClient) Recv() (*corev1.Event, error) {
	v := &corev1.Event{}
	return v, f.RecvEvent(v)
}
