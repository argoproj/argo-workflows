package events

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/logging"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// by default, allow a source to send 10000 events about an object
const defaultSpamBurst = 10000

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

type debugfAdapter struct {
	logger logging.Logger
}

// debugfAdapter adapts the logging system to the signature expected by StartLogging.
func (a *debugfAdapter) Debugf(format string, args ...interface{}) {
	// nolint:contextcheck
	a.logger.Debugf(context.Background(), format, args...)
}

func (m *eventRecorderManager) Get(ctx context.Context, namespace string) record.EventRecorder {
	m.lock.Lock()
	defer m.lock.Unlock()
	eventRecorder, ok := m.eventRecorders[namespace]
	if ok {
		return eventRecorder
	}
	eventCorrelationOption := record.CorrelatorOptions{BurstSize: defaultSpamBurst, KeyFunc: customEventAggregatorFuncWithAnnotations}
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(eventCorrelationOption)
	adapter := &debugfAdapter{logger: logging.RequireLoggerFromContext(ctx)}
	eventBroadcaster.StartLogging(adapter.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: m.kubernetes.CoreV1().Events(namespace)})
	m.eventRecorders[namespace] = eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "workflow-controller"})
	return m.eventRecorders[namespace]
}

func NewEventRecorderManager(kubernetes kubernetes.Interface) EventRecorderManager {
	return &eventRecorderManager{
		kubernetes:     kubernetes,
		lock:           sync.Mutex{},
		eventRecorders: make(map[string]record.EventRecorder),
	}
}
