package latch

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Interface interface {
	Update(obj metav1.Object)
	Pass(obj metav1.Object) bool
	Remove(obj metav1.Object)
}

type latch struct {
	mutex sync.RWMutex
	store map[types.UID]string
}

func (l *latch) Remove(obj metav1.Object) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.store, obj.GetUID())
}

func (l *latch) Pass(obj metav1.Object) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	resourceVersion, exists := l.store[obj.GetUID()]
	return !exists || resourceVersion <= obj.GetResourceVersion()
}

func (l *latch) Update(obj metav1.Object) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	resourceVersion, exists := l.store[obj.GetUID()]
	if !exists || obj.GetResourceVersion() > resourceVersion {
		l.store[obj.GetUID()] = obj.GetResourceVersion()
	}
}

func New() Interface {
	return &latch{mutex: sync.RWMutex{}, store: make(map[types.UID]string)}
}
