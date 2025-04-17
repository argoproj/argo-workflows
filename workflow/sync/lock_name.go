package sync

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type lockKind string

const (
	lockKindConfigMap lockKind = "ConfigMap"
	lockKindDatabase  lockKind = "Database"
	lockKindMutex     lockKind = "Mutex"
)

type lockName struct {
	Namespace    string
	ResourceName string
	Key          string
	Kind         lockKind
}

func newLockName(namespace, resourceName, lockKey string, kind lockKind) *lockName {
	return &lockName{
		Namespace:    namespace,
		ResourceName: resourceName,
		Key:          lockKey,
		Kind:         kind,
	}
}

func getSemaphoreLockName(sem *v1alpha1.SemaphoreRef, wfNamespace string) (*lockName, error) {
	switch {
	case sem.ConfigMapKeyRef != nil && sem.Database != nil:
		return nil, fmt.Errorf("invalid semaphore with both ConfigMapKeyRef and Database")
	case sem.ConfigMapKeyRef != nil:
		namespace := sem.Namespace
		if namespace == "" {
			namespace = wfNamespace
		}
		return newLockName(namespace, sem.ConfigMapKeyRef.Name, sem.ConfigMapKeyRef.Key, lockKindConfigMap), nil
	case sem.Database != nil:
		namespace := sem.Namespace
		if namespace == "" {
			namespace = wfNamespace
		}
		return newLockName(namespace, sem.Database.Key, "", lockKindDatabase), nil
	default:
		return nil, fmt.Errorf("cannot get LockName for a Semaphore without a ConfigMapRef or Database")
	}
}

func getMutexLockName(mtx *v1alpha1.Mutex, wfNamespace string) *lockName {
	namespace := mtx.Namespace
	if namespace == "" {
		namespace = wfNamespace
	}
	if mtx.Database {
		return newLockName(namespace, mtx.Name, "", lockKindDatabase)
	}
	return newLockName(namespace, mtx.Name, "", lockKindMutex)
}

func (item *syncItem) lockName(wfNamespace string) (*lockName, error) {
	switch {
	case item.semaphore != nil:
		return getSemaphoreLockName(item.semaphore, wfNamespace)
	case item.mutex != nil:
		return getMutexLockName(item.mutex, wfNamespace), nil
	default:
		return nil, fmt.Errorf("cannot get lockName if not semaphore or mutex")
	}
}

func DecodeLockName(name string) (*lockName, error) {
	log.Infof("DecodeLockName %s", name)
	items := strings.SplitN(name, "/", 3)
	if len(items) < 3 {
		return nil, errors.New(errors.CodeBadRequest, "Invalid lock key: unknown format")
	}

	var lock lockName
	lockKind := lockKind(items[1])
	namespace := items[0]

	switch lockKind {
	case lockKindMutex, lockKindDatabase:
		lock = lockName{Namespace: namespace, Kind: lockKind, ResourceName: items[2]}
	case lockKindConfigMap:
		components := strings.Split(items[2], "/")

		if len(components) != 2 {
			return nil, errors.New(errors.CodeBadRequest, "Invalid ConfigMap lock key: unknown format")
		}

		lock = lockName{Namespace: namespace, Kind: lockKind, ResourceName: components[0], Key: components[1]}
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("Invalid lock key, unexpected kind: %s", lockKind))
	}

	err := lock.validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func (ln *lockName) String() string {
	switch ln.Kind {
	case lockKindMutex, lockKindDatabase:
		return ln.validateEncoding(fmt.Sprintf("%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName))
	default:
		return ln.validateEncoding(fmt.Sprintf("%s/%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName, ln.Key))
	}
}

func (ln *lockName) validate() error {
	if ln.Namespace == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Namespace is missing")
	}
	if ln.Kind == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Kind is missing")
	}
	if ln.ResourceName == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: ResourceName is missing")
	}
	if ln.Kind == lockKindConfigMap && ln.Key == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Key is missing for ConfigMap lock")
	}
	return nil
}

func (ln *lockName) validateEncoding(encoding string) string {
	decoded, err := DecodeLockName(encoding)
	if err != nil {
		panic(fmt.Sprintf("bug: unable to decode lock (%s) that was just encoded: %s", encoding, err))
	}
	if ln.Namespace != decoded.Namespace || ln.Kind != decoded.Kind || ln.ResourceName != decoded.ResourceName || ln.Key != decoded.Key {
		panic("bug: lock that was just encoded does not match encoding")
	}
	return encoding
}

func (ln *lockName) dbKey() string {
	return fmt.Sprintf("%s/%s", ln.Namespace, ln.ResourceName)
}

func needDBSession(lockKeys []string) (bool, error) {
	for _, key := range lockKeys {
		lock, err := DecodeLockName(key)
		if err != nil {
			return false, err
		}
		switch lock.Kind {
		case lockKindDatabase:
			return true, nil
		}
	}
	return false, nil
}
