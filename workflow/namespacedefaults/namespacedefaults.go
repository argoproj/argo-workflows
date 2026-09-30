package namespacedefaults

import (
	"context"
	"fmt"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// Key is the key within the ConfigMap holding the defaults, as the YAML of a Workflow.
// It matches the workflowDefaults key of the workflow controller ConfigMap, so the same
// document works in either place.
const Key = "workflowDefaults"

// Func returns the defaults that apply to workflows in a namespace.
//
// Callers take this rather than a *wfv1.Workflow fixed at startup, because defaults are no
// longer controller-wide: a namespace carries its own in a ConfigMap that can change while
// the process runs.
type Func func(ctx context.Context, namespace string) (*wfv1.Workflow, error)

// Interface looks up namespace-level workflow defaults.
type Interface interface {
	// Get returns the defaults configured for namespace, or nil if the namespace does not
	// configure any. A ConfigMap that exists but cannot be used is an error rather than a
	// silent nil, because defaults that quietly fail to apply are worse than a visible
	// failure.
	Get(ctx context.Context, namespace string) (*wfv1.Workflow, error)
}

// New returns a lookup backed by an informer's indexer, for the workflow controller.
//
// The indexer is taken as a getter rather than a value because this is constructed from
// updateConfig, while the ConfigMap informer is not created until the controller runs.
// Resolving it per call keeps the two lifecycles independent. indexName is passed in so
// that this package does not depend on the controller's index definitions.
func New(indexer func() cache.Indexer, indexName string) Interface {
	return &indexedDefaults{indexer: indexer, indexName: indexName}
}

// NewLister returns a lookup that lists from the API server, for callers without an
// informer. The argo-server is the only one: it validates and submits far less often than
// the controller reconciles, so a List per call is proportionate where it would not be on
// the controller's hot path.
func NewLister(kubernetesInterface kubernetes.Interface) Interface {
	return &listedDefaults{kubernetesInterface: kubernetesInterface}
}

// Resolve calls f for namespace, tolerating a nil f. Some callers are constructed without
// defaults at all, notably the offline and kube API clients.
func Resolve(ctx context.Context, f Func, namespace string) (*wfv1.Workflow, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx, namespace)
}

// Merged layers namespace defaults over controller defaults for one namespace, returning
// nil when neither configures anything. Both the controller and the server need this, so
// the precedence lives in one place rather than being re-derived at each call site.
func Merged(ctx context.Context, i Interface, controllerDefaults *wfv1.Workflow, namespace string) (*wfv1.Workflow, error) {
	var namespaceDefaults *wfv1.Workflow
	if i != nil {
		var err error
		namespaceDefaults, err = i.Get(ctx, namespace)
		if err != nil {
			return nil, err
		}
	}
	if namespaceDefaults == nil && controllerDefaults == nil {
		return nil, nil
	}

	// MergeTo lets the target win, so merging the more specific layer first leaves the
	// namespace values in place and lets the controller layer fill only what is still empty.
	merged := &wfv1.Workflow{}
	if namespaceDefaults != nil {
		if err := util.MergeTo(namespaceDefaults, merged); err != nil {
			return nil, err
		}
	}
	if controllerDefaults != nil {
		if err := util.MergeTo(controllerDefaults, merged); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

// labelSelector matches the ConfigMaps this package reads.
func labelSelector() string {
	return common.LabelKeyConfigMapType + "=" + common.LabelValueTypeConfigMapWorkflowDefaults
}

type indexedDefaults struct {
	indexer   func() cache.Indexer
	indexName string
}

func (s *indexedDefaults) Get(ctx context.Context, namespace string) (*wfv1.Workflow, error) {
	indexer := s.indexer()
	if indexer == nil {
		return nil, fmt.Errorf("cannot resolve namespace workflow defaults: ConfigMap informer is not running")
	}

	objs, err := indexer.ByIndex(s.indexName, common.LabelValueTypeConfigMapWorkflowDefaults)
	if err != nil {
		return nil, fmt.Errorf("failed to look up workflow defaults ConfigMaps: %w", err)
	}

	// The index is keyed on the label value alone, so this returns the labelled ConfigMaps
	// from every namespace the controller manages.
	var found []*v1.ConfigMap
	for _, obj := range objs {
		cm, ok := obj.(*v1.ConfigMap)
		if !ok || cm.Namespace != namespace {
			continue
		}
		found = append(found, cm)
	}
	return workflowFrom(ctx, namespace, found)
}

type listedDefaults struct {
	kubernetesInterface kubernetes.Interface
}

func (s *listedDefaults) Get(ctx context.Context, namespace string) (*wfv1.Workflow, error) {
	// ConfigMaps("").List would list every namespace, so an empty namespace would pick
	// up someone else's defaults or trip the at-most-one rule. The indexed lookup
	// returns nil here because no ConfigMap has an empty namespace; match it.
	if namespace == "" {
		return nil, nil
	}

	list, err := s.kubernetesInterface.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow defaults ConfigMaps in namespace %q: %w", namespace, err)
	}

	found := make([]*v1.ConfigMap, 0, len(list.Items))
	for i := range list.Items {
		found = append(found, &list.Items[i])
	}
	return workflowFrom(ctx, namespace, found)
}

// workflowFrom turns the labelled ConfigMaps of one namespace into the defaults they
// describe. Both lookups share it so that the controller and the server agree on what is
// an error and on what a defaults document is allowed to contain.
func workflowFrom(ctx context.Context, namespace string, found []*v1.ConfigMap) (*wfv1.Workflow, error) {
	if len(found) == 0 {
		return nil, nil
	}
	if len(found) > 1 {
		names := make([]string, 0, len(found))
		for _, cm := range found {
			names = append(names, cm.Name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("namespace %q has %d ConfigMaps labelled %s (%s), there must be at most one",
			namespace, len(found), labelSelector(), strings.Join(names, ", "))
	}

	cm := found[0]
	value, ok := cm.Data[Key]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %q in namespace %q is missing key %q", cm.Name, namespace, Key)
	}

	wf := &wfv1.Workflow{}
	// Strict, to match config/controller.go for the same document shape: a misspelled field
	// should be rejected rather than silently applying nothing.
	if err := yaml.UnmarshalStrict([]byte(value), wf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key %q of ConfigMap %q in namespace %q: %w", Key, cm.Name, namespace, err)
	}

	// MergeTo copies the whole object, not just the spec, so restrict what a defaults
	// document is allowed to contribute before it goes anywhere near a Workflow. Otherwise
	// whoever can write this ConfigMap could set controller-owned state - status,
	// finalizers, ownerReferences - on every Workflow in the namespace. Labels and
	// annotations are kept because the controller-level docs show them.
	wf.Status = wfv1.WorkflowStatus{}
	wf.ObjectMeta = metav1.ObjectMeta{
		Labels:      wf.Labels,
		Annotations: wf.Annotations,
	}

	logging.RequireLoggerFromContext(ctx).WithField("namespace", namespace).Debug(ctx, "resolved namespace workflow defaults")
	return wf, nil
}
