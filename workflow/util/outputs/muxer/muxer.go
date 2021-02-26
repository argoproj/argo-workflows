package muxer

import (
	"encoding/json"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Demux(message string) (string, *wfv1.Outputs, error) {
	i := strings.Index(message, `|{`) // we use `|` as the delimiter, but we'll actually always see `|{`, this reduces the risk of match which should not be allowed
	if i < 0 {
		return message, nil, nil
	}
	outputs := &wfv1.Outputs{}
	return message[0:i], outputs, json.Unmarshal([]byte(message[i+1:]), outputs)
}

func Mux(message string, outputs *wfv1.Outputs) (string, error) {
	if outputs == nil {
		return message, nil
	}
	data, err := json.Marshal(outputs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%s", message, string(data)), nil
}
