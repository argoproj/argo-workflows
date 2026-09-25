package namespacedefaults

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v4/util/wait"
)

const (
	// ConfigMapName is the well-known name of the ConfigMap holding namespace-level
	// workflow defaults. The name is fixed rather than discovered by label so that a
	// namespace cannot have more than one: Kubernetes already guarantees that names are
	// unique within a namespace, so "which one wins" is unrepresentable.
	ConfigMapName = "workflow-defaults"

	// Key is the key within that ConfigMap holding the defaults, as the YAML of a
	// Workflow. It matches the workflowDefaults key of the workflow controller
	// ConfigMap, so the same document works in either place.
	Key = "workflowDefaults"
)

// Interface looks up namespace-level workflow defaults.
type Interface interface {
	// Get returns the defaults configured for namespace, or nil if the namespace does
	// not configure any. A ConfigMap that exists but cannot be used is an error rather
	// than a silent nil, because defaults that quietly fail to apply are worse than a
	// visible failure.
	Get(ctx context.Context, namespace string) (*wfv1.Workflow, error)
}

func New(kubernetesInterface kubernetes.Interface) Interface {
	return &namespaceDefaults{kubernetesInterface}
}

type namespaceDefaults struct {
	kubernetesInterface kubernetes.Interface
}

func (s *namespaceDefaults) Get(ctx context.Context, namespace string) (*wfv1.Workflow, error) {
	var cm *v1.ConfigMap
	err := waitutil.Backoff(retry.DefaultRetry(ctx), func() (bool, error) {
		var err error
		cm, err = s.kubernetesInterface.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigMapName, metav1.GetOptions{})
		return !errorsutil.IsTransientErrQuiet(ctx, err), err
	})
	if apierr.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %q in namespace %q: %w", ConfigMapName, namespace, err)
	}

	value, ok := cm.Data[Key]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %q in namespace %q is missing key %q", ConfigMapName, namespace, Key)
	}

	wf := &wfv1.Workflow{}
	if err := yaml.Unmarshal([]byte(value), wf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key %q of ConfigMap %q in namespace %q: %w", Key, ConfigMapName, namespace, err)
	}

	logging.RequireLoggerFromContext(ctx).WithField("namespace", namespace).Debug(ctx, "resolved namespace workflow defaults")
	return wf, nil
}
