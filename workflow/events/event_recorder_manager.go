package events

import (
	"sync"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

type EventRecorderManager interface {
	Get(namespace string) record.EventRecorder
}

type eventRecorderManager struct {
	kubernetes     kubernetes.Interface
	lock           sync.Mutex
	eventRecorders map[string]record.EventRecorder
}

func (m *eventRecorderManager) Get(namespace string) record.EventRecorder {
	m.lock.Lock()
	defer m.lock.Unlock()
	eventRecorder, ok := m.eventRecorders[namespace]
	if ok {
		return eventRecorder
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Debugf)
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
