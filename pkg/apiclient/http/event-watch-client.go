package http

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
)

type eventWatchClient struct{ clientStream }

func (f eventWatchClient) Recv() (*corev1.Event, error) {
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	out := &corev1.Event{}
	return out, json.Unmarshal(data, out)
}
