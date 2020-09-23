package sync

import (
	"fmt"
	"strings"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
	Type         LockType
}

func NewLockName(namespace, resourceName, lockKey string, kind LockKind) *LockName {
	return &LockName{
		Namespace:    namespace,
		ResourceName: resourceName,
		Key:          lockKey,
		Kind:         kind,
	}
}

func GetLockName(sync *v1alpha1.Synchronization, namespace string) (*LockName, error) {
	switch sync.GetType() {
	case v1alpha1.SynchronizationTypeSemaphore:
		if sync.Semaphore.ConfigMapKeyRef != nil {
			return NewLockName(namespace, sync.Semaphore.ConfigMapKeyRef.Name, sync.Semaphore.ConfigMapKeyRef.Key, LockKindConfigMap), nil
		}
		return nil, fmt.Errorf("cannot get LockName for a Semaphore without a ConfigMapRef")
	case v1alpha1.SynchronizationTypeMutex:
		return NewLockName(namespace, sync.Mutex.Name, "", LockKindMutex), nil
	default:
		return nil, fmt.Errorf("cannot get LockName for a Sync of Unknown type")
	}
}

func DecodeLockName(lockName string) (*LockName, error) {
	items := strings.Split(lockName, "/")
	var lock LockName
	// For mutex lockname
	if len(items) == 3 && items[1] == string(LockTypeMutex) {
		lock = LockName{Namespace: items[0], Kind: LockKind(items[1]), ResourceName: items[2]}
	} else if len(items) == 4 { // For Semaphore lockname
		lock = LockName{Namespace: items[0], Kind: LockKind(items[1]), ResourceName: items[2], Key: items[3]}
	} else {
		return nil, errors.New(errors.CodeBadRequest, "Invalid Lock Key")
	}
	err := lock.validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func (ln *LockName) getLockKey() string {
	if ln.Kind == LockKindMutex {
		return fmt.Sprintf("%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName)
	}
	return fmt.Sprintf("%s/%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName, ln.Key)
}

func (ln *LockName) validate() error {
	if ln.Namespace == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Namespace is missing")
	}
	if ln.Kind == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Kind is missing")
	}
	if ln.ResourceName == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. ResourceName is missing")
	}
	if ln.Kind != LockKindMutex && ln.Key == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Key is missing")
	}
	return nil
}
