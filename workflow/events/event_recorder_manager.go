package events

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog/v2"

	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// by default, allow a source to send 10000 events about an object
const defaultSpamBurst = 10000

type EventRecorderManager interface {
	Get(ctx context.Context, namespace string) record.EventRecorder
}

type eventRecorderManager struct {
	kubernetes  kubernetes.Interface
	once        sync.Once
	logger      klog.Logger
	broadcaster record.EventBroadcaster
	recorder    record.EventRecorderLogger
}

// customEventAggregatorFuncWithAnnotations enhances the default `EventAggregatorByReasonFunc` by
// including annotation values as part of the event aggregation key.
func customEventAggregatorFuncWithAnnotations(event *apiv1.Event) (string, string) {
	var joinedAnnotationsStr string
	includeAnnotations := env.LookupEnvStringOr("EVENT_AGGREGATION_WITH_ANNOTATIONS", "false")
	if annotations := event.GetAnnotations(); includeAnnotations == "true" && annotations != nil {
		annotationVals := make([]string, 0, len(annotations))
		for _, v := range annotations {
			annotationVals = append(annotationVals, v)
		}
		sort.Strings(annotationVals)
		joinedAnnotationsStr = strings.Join(annotationVals, "")
	}
	return strings.Join([]string{
		event.Source.Component,
		event.Source.Host,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		string(event.InvolvedObject.UID),
		event.InvolvedObject.APIVersion,
		event.Type,
		event.Reason,
		event.ReportingController,
		event.ReportingInstance,
		joinedAnnotationsStr,
	},
		""), event.Message
}

func (m *eventRecorderManager) Get(ctx context.Context, namespace string) record.EventRecorder {
	m.once.Do(func() {
		options := record.CorrelatorOptions{BurstSize: defaultSpamBurst, KeyFunc: customEventAggregatorFuncWithAnnotations}
		// Get may receive a short-lived server request context. The shared
		// broadcaster and its logger must instead live for the process lifetime.
		m.broadcaster = record.NewBroadcaster(record.WithCorrelatorOptions(options),
			record.WithContext(klog.NewContext(context.Background(), m.logger)))
		m.broadcaster.StartStructuredLogging(klog.Level(0))
		m.broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: m.kubernetes.CoreV1().Events("")})
		m.recorder = m.broadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "workflow-controller"})
	})
	// Request-specific logging belongs to this caller's recorder, never to the
	// shared broadcaster or a manager-owned namespace cache.
	logger := logr.New(&logrSink{logger: logging.RequireLoggerFromContext(ctx)})
	return &namespaceEventRecorder{namespace: namespace, recorder: m.recorder.WithLogger(logger)}
}

// namespaceEventRecorder preserves the namespace guard previously supplied by
// Events(namespace), while the shared sink routes using the event's namespace.
// These wrappers are not cached: namespace churn retains no manager resources.
type namespaceEventRecorder struct {
	namespace string
	recorder  record.EventRecorder
}

func (r *namespaceEventRecorder) accepts(object runtime.Object) bool {
	if r.namespace == "" {
		return true
	}
	ref, err := reference.GetReference(scheme.Scheme, object)
	if err != nil {
		// Let the recorder report malformed objects as before.
		return true
	}
	namespace := ref.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return namespace == r.namespace
}

func (r *namespaceEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	if r.accepts(object) {
		r.recorder.Event(object, eventtype, reason, message)
	}
}

func (r *namespaceEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	if r.accepts(object) {
		r.recorder.Eventf(object, eventtype, reason, messageFmt, args...)
	}
}

func (r *namespaceEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	if r.accepts(object) {
		r.recorder.AnnotatedEventf(object, annotations, eventtype, reason, messageFmt, args...)
	}
}

// NewEventRecorderManager uses an info-level text logger for shared event logging.
// Use NewEventRecorderManagerWithLogger to supply the configured component logger.
func NewEventRecorderManager(kubernetes kubernetes.Interface) EventRecorderManager {
	return NewEventRecorderManagerWithLogger(kubernetes, logging.NewSlogLogger(logging.Info, logging.Text))
}

// NewEventRecorderManagerWithLogger shares one process-lifetime broadcaster and sink.
// Its bounded client-go queue and correlation LRUs are shared across namespaces;
// busy namespaces can evict each other's history or delay/drop best-effort events.
// This does not provide shutdown/drain semantics for short-lived managers.
// logger must be the configured process/component logger, not a request logger.
// Only the logger is retained; no caller context controls the shared broadcaster.
func NewEventRecorderManagerWithLogger(kubernetes kubernetes.Interface, logger logging.Logger) EventRecorderManager {
	return &eventRecorderManager{
		kubernetes: kubernetes,
		logger:     logr.New(&logrSink{logger: logger}),
	}
}
