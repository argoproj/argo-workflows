package persist

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NodeStatusVersion(s wfv1.Nodes) (string, string, error) {
	marshalled, err := json.Marshal(s)
	if err != nil {
		return "", "", err
	}
	h := fnv.New32()
	_, _ = h.Write(marshalled)
	return string(marshalled), fmt.Sprintf("fnv:%v", h.Sum32()), nil
}
