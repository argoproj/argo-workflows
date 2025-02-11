package sync

import (
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type LockKind string

const (
	LockKindConfigMap LockKind = "ConfigMap"
	LockKindMutex     LockKind = "Mutex"
)

type LockName struct {
	Namespace    string
	ResourceName string
	Key          string
	Kind         LockKind
}

func NewLockName(namespace, resourceName, lockKey string, kind LockKind) *LockName {
	return &LockName{
		Namespace:    namespace,
		ResourceName: resourceName,
		Key:          lockKey,
		Kind:         kind,
	}
}

func GetLockName(sync *v1alpha1.Synchronization, wfNamespace string) (*LockName, error) {
	switch sync.GetType() {
	case v1alpha1.SynchronizationTypeSemaphore:
		if sync.Semaphore.ConfigMapKeyRef != nil {
			namespace := sync.Semaphore.Namespace
			if namespace == "" {
				namespace = wfNamespace
			}
			return NewLockName(namespace, sync.Semaphore.ConfigMapKeyRef.Name, sync.Semaphore.ConfigMapKeyRef.Key, LockKindConfigMap), nil
		}
		return nil, fmt.Errorf("cannot get LockName for a Semaphore without a ConfigMapRef")
	case v1alpha1.SynchronizationTypeMutex:
		namespace := sync.Mutex.Namespace
		if namespace == "" {
			namespace = wfNamespace
		}
		return NewLockName(namespace, sync.Mutex.Name, "", LockKindMutex), nil
	default:
		return nil, fmt.Errorf("cannot get LockName for a Sync of Unknown type")
	}
}

func DecodeLockName(lockName string) (*LockName, error) {
	items := strings.SplitN(lockName, "/", 3)
	if len(items) < 3 {
		return nil, errors.New(errors.CodeBadRequest, "Invalid lock key: unknown format")
	}

	var lock LockName
	lockKind := LockKind(items[1])
	namespace := items[0]

	switch lockKind {
	case LockKindMutex:
		lock = LockName{Namespace: namespace, Kind: lockKind, ResourceName: items[2]}
	case LockKindConfigMap:
		components := strings.Split(items[2], "/")

		if len(components) != 2 {
			return nil, errors.New(errors.CodeBadRequest, "Invalid ConfigMap lock key: unknown format")
		}

		lock = LockName{Namespace: namespace, Kind: lockKind, ResourceName: components[0], Key: components[1]}
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("Invalid lock key, unexpected kind: %s", lockKind))
	}

	err := lock.Validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func (ln *LockName) EncodeName() string {
	if ln.Kind == LockKindMutex {
		return ln.ValidateEncoding(fmt.Sprintf("%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName))
	}
	return ln.ValidateEncoding(fmt.Sprintf("%s/%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName, ln.Key))
}

func (ln *LockName) Validate() error {
	if ln.Namespace == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Namespace is missing")
	}
	if ln.Kind == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Kind is missing")
	}
	if ln.ResourceName == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: ResourceName is missing")
	}
	if ln.Kind == LockKindConfigMap && ln.Key == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Key is missing for ConfigMap lock")
	}
	return nil
}

func (ln *LockName) ValidateEncoding(encoding string) string {
	decoded, err := DecodeLockName(encoding)
	if err != nil {
		panic(fmt.Sprintf("bug: unable to decode lock that was just encoded: %s", err))
	}
	if ln.Namespace != decoded.Namespace || ln.Kind != decoded.Kind || ln.ResourceName != decoded.ResourceName || ln.Key != decoded.Key {
		panic("bug: lock that was just encoded does not match encoding")
	}
	return encoding
}
