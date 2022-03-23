package controller

import (
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const localProfileKey profileKey = ""

type profileKey string

func newProfileKey(obj metav1.Object) profileKey {
	return profileKey(common.Cluster(obj))
}
