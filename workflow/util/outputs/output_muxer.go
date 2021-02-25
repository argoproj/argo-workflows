package outputs

import (
	"encoding/json"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Demux(message string) (string, *wfv1.Outputs, error) {
	parts := strings.SplitN(message, "|", 2)
	message = parts[0]
	if len(parts) < 2 {
		return message, nil, nil
	}
	outputs := &wfv1.Outputs{}
	return message, outputs, json.Unmarshal([]byte(parts[1]), outputs)
}

func Mux(message string, outputs *wfv1.Outputs) (string, error) {
	data, err := json.Marshal(outputs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%s", message, string(data)), nil
}
