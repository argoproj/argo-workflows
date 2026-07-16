package events

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/argoproj/argo-workflows/v4/util/env"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// by default, allow a source to send 10000 events about an object
const defaultSpamBurst = 10000

// eventScheme resolves both core Kubernetes and Argo CRD types, so the recorder's
// GetReference can build an InvolvedObject for Argo objects even when they carry
// empty TypeMeta (informer-cached and tolerant-decoded objects do). With a k8s-only
// scheme, GetReference falls back to scheme.ObjectKinds and errors on Argo types,
// silently dropping the event.
var eventScheme = func() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(s))
	utilruntime.Must(wfv1.AddToScheme(s))
	return s
}()

type EventRecorderManager interface {
	Get(ctx context.Context, namespace string) record.EventRecorder
}

type eventRecorderManager struct {
	kubernetes     kubernetes.Interface
	lock           sync.Mutex
	eventRecorders map[string]record.EventRecorder
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
	m.lock.Lock()
	defer m.lock.Unlock()
	eventRecorder, ok := m.eventRecorders[namespace]
	if ok {
		return eventRecorder
	}

	setupKlogAdapter(ctx)

	eventCorrelationOption := record.CorrelatorOptions{BurstSize: defaultSpamBurst, KeyFunc: customEventAggregatorFuncWithAnnotations}
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(eventCorrelationOption)

	eventBroadcaster.StartStructuredLogging(klog.Level(0)) // Info level
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: m.kubernetes.CoreV1().Events(namespace)})
	m.eventRecorders[namespace] = eventBroadcaster.NewRecorder(eventScheme, apiv1.EventSource{Component: "workflow-controller"})
	return m.eventRecorders[namespace]
}

func NewEventRecorderManager(kubernetes kubernetes.Interface) EventRecorderManager {
	return &eventRecorderManager{
		kubernetes:     kubernetes,
		lock:           sync.Mutex{},
		eventRecorders: make(map[string]record.EventRecorder),
	}
}
