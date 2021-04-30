package util

import (
	"encoding/json"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// Deprecated: use MustUnmarshall
func MustUnmarshallJSON(text string, v interface{}) {
	wfv1.MustUnmarshal(text, v)
}

func MustMarshallJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
