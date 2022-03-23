package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const localProfileKey profileKey = ""

type profileKey string

func newProfileKey(obj metav1.Object) profileKey {
	return profileKey(common.Cluster(obj))
}
