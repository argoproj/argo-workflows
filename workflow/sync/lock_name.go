package sync

import (
	"fmt"
	"github.com/argoproj/argo/errors"
	"strings"
)

type LockKind string

const (
	LockKindConfigMap LockKind = "ConfigMap"
	LockKindMutex LockKind = "Mutex"
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
