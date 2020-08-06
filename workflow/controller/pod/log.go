package pod

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func LogChanges(oldPod *apiv1.Pod, newPod *apiv1.Pod) {
	a, _ := json.Marshal(oldPod)
	b, _ := json.Marshal(newPod)
	patch, _ := strategicpatch.CreateTwoWayMergePatch(a, b, &apiv1.Pod{})
	log.Debugln(string(patch))
}
