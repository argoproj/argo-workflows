package rbac

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

const orderLabel = workflow.WorkflowFullName + "/rbac-order"
const groupsLabel = workflow.WorkflowFullName + "/rbac-groups"
const defaultLabel = workflow.WorkflowFullName + "/rbac-default"

type Interface interface {
	ServiceAccount(groups []string) (*corev1.LocalObjectReference, error)
}

func New(serviceAccountIf v1.ServiceAccountInterface) Interface {
	return &rbac{serviceAccountIf}
}

type rbac struct {
	serviceAccountIf v1.ServiceAccountInterface
}

func (c rbac) ServiceAccount(groups []string) (*corev1.LocalObjectReference, error) {
	hasGroup := make(map[string]bool)
	for _, group := range groups {
		hasGroup[group] = true
	}
	list, err := c.serviceAccountIf.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var serviceAccounts []corev1.ServiceAccount
	for _, a := range list.Items {
		for _, l := range []string{orderLabel, groupsLabel, defaultLabel} {
			if _, ok := a.GetAnnotations()[l]; ok {
				serviceAccounts = append(serviceAccounts, a)
				break
			}
		}
	}
	sort.Slice(serviceAccounts, func(i, j int) bool {
		x, _ := strconv.Atoi(serviceAccounts[i].GetAnnotations()[orderLabel])
		y, _ := strconv.Atoi(serviceAccounts[j].GetAnnotations()[orderLabel])
		return x < y
	})
	for _, a := range serviceAccounts {
		for _, g := range strings.Split(a.GetAnnotations()[groupsLabel], ",") {
			if hasGroup[g] {
				return &corev1.LocalObjectReference{Name: a.Name}, nil
			}
		}
	}
	for _, a := range serviceAccounts {
		if _, ok := a.GetLabels()[defaultLabel]; ok {
			return &corev1.LocalObjectReference{Name: a.Name}, nil
		}
	}
	return nil, errors.New("no service account found annotated with the user's groups, nor any service account annotated as default")
}

var _ Interface = rbac{}
