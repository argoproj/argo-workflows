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
	mutex sync.Mutex
	store map[types.UID]string
}

func (l *latch) Remove(obj metav1.Object) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.store, obj.GetUID())
}

func (l *latch) Pass(obj metav1.Object) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	resourceVersion, exists := l.store[obj.GetUID()]
	if exists {
		delete(l.store, obj.GetUID())
	}
	return !exists || resourceVersion == obj.GetResourceVersion()
}

// set the expected next version to this, overwriting any other versions
func (l *latch) Update(obj metav1.Object) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.store[obj.GetUID()] = obj.GetResourceVersion()
}

func New() Interface {
	return &latch{mutex: sync.Mutex{}, store: make(map[types.UID]string)}
}
