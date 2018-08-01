package metrics

import (
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	resyncPeriod = 5 * time.Minute

	descWorkflowDefaultLabels = []string{"namespace", "name"}

	descWorkflowInfo = prometheus.NewDesc(
		"kube_wf_info",
		"Information about workflow.",
		append(descWorkflowDefaultLabels, "entrypoint"),
		nil,
	)
	descWorkflowStartedAt = prometheus.NewDesc(
		"kube_wf_start_time",
		"Start time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowFinishedAt = prometheus.NewDesc(
		"kube_wf_completion_time",
		"Completion time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowCreated = prometheus.NewDesc(
		"kube_wf_created",
		"Unix creation timestamp",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowStatusPhase = prometheus.NewDesc(
		"kube_wf_status_phase",
		"The workflow current phase.",
		append(descWorkflowDefaultLabels, "phase"),
		nil,
	)
)

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

type workflowLister func() ([]wfv1.Workflow, error)

func (l workflowLister) List() ([]wfv1.Workflow, error) {
	return l()
}

type sharedInformerList []cache.SharedInformer

func newsharedInformerList(client rest.Interface, resource string, namespaces []string, objType runtime.Object) *sharedInformerList {
	sinfs := sharedInformerList{}
	for _, namespace := range namespaces {
		slw := cache.NewListWatchFromClient(client, resource, namespace, fields.Everything())
		sinfs = append(sinfs, cache.NewSharedInformer(slw, objType, resyncPeriod))
	}
	return &sinfs
}

func (sil sharedInformerList) Run(stopCh <-chan struct{}) {
	for _, sinf := range sil {
		go sinf.Run(stopCh)
	}
}

func registerWfCollector(registry prometheus.Registerer, kubeClient wfclientset.Interface, namespaces []string) {
	client := kubeClient.ArgoprojV1alpha1().RESTClient()
	winfs := newsharedInformerList(client, "workflows", namespaces, &wfv1.Workflow{})

	workflowLister := workflowLister(func() (workflows []wfv1.Workflow, err error) {
		for _, pinf := range *winfs {
			for _, m := range pinf.GetStore().List() {
				workflows = append(workflows, *m.(*wfv1.Workflow))
			}
		}
		return workflows, nil
	})

	registry.MustRegister(&workflowCollector{store: workflowLister})
	winfs.Run(context.Background().Done())
}

type wfStore interface {
	List() (workflows []wfv1.Workflow, err error)
}

// workflowCollector collects metrics about all workflows in the cluster.
type workflowCollector struct {
	store wfStore
}

func createKubeClient() (wfclientset.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentConfig.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentConfig.ContentType = "application/vnd.kubernetes.protobuf"
	config.ContentConfig.GroupVersion = &wfv1.SchemeGroupVersion

	kubeClient := wfclientset.NewForConfigOrDie(config)

	return kubeClient, nil
}

// Describe implements the prometheus.Collector interface.
func (wc *workflowCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descWorkflowInfo
	ch <- descWorkflowStartedAt
	ch <- descWorkflowFinishedAt
	ch <- descWorkflowCreated
	ch <- descWorkflowStatusPhase
}

// Collect implements the prometheus.Collector interface.
func (wc *workflowCollector) Collect(ch chan<- prometheus.Metric) {
	workflows, err := wc.store.List()
	if err != nil {
		return
	}
	for _, wf := range workflows {
		wc.collectWorkflow(ch, wf)
	}

}

func (wc *workflowCollector) collectWorkflow(ch chan<- prometheus.Metric, wf wfv1.Workflow) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{wf.Namespace, wf.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	addGauge(descWorkflowInfo, 1, wf.Spec.Entrypoint)

	if phase := wf.Status.Phase; phase != "" {
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeRunning), string(wfv1.NodeRunning))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeSucceeded), string(wfv1.NodeSucceeded))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeSkipped), string(wfv1.NodeSkipped))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeFailed), string(wfv1.NodeFailed))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeError), string(wfv1.NodeError))
	}

	if !wf.CreationTimestamp.IsZero() {
		addGauge(descWorkflowCreated, float64(wf.CreationTimestamp.Unix()))
	}

	if !wf.Status.StartedAt.IsZero() {
		addGauge(descWorkflowStartedAt, float64(wf.Status.StartedAt.Unix()))
	}

	if !wf.Status.FinishedAt.IsZero() {
		addGauge(descWorkflowFinishedAt, float64(wf.Status.FinishedAt.Unix()))
	}

}
