package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type lockKind string

const (
	lockKindConfigMap lockKind = "ConfigMap"
	lockKindDatabase  lockKind = "Database"
	lockKindMutex     lockKind = "Mutex"
)

// LockName represents a decoded lock name with its components.
type LockName interface {
	GetNamespace() string
	GetResourceName() string
	GetKey() string
	getKind() lockKind
	getDBKey() string
	String(ctx context.Context) string
}

type lockName struct {
	namespace    string
	resourceName string
	key          string
	kind         lockKind
}

func (ln *lockName) GetNamespace() string    { return ln.namespace }
func (ln *lockName) GetResourceName() string { return ln.resourceName }
func (ln *lockName) GetKey() string          { return ln.key }
func (ln *lockName) getKind() lockKind       { return ln.kind }

func newLockName(namespace, resourceName, lockKey string, kind lockKind) *lockName {
	return &lockName{
		namespace:    namespace,
		resourceName: resourceName,
		key:          lockKey,
		kind:         kind,
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

func (i *syncItem) lockName(wfNamespace string) (*lockName, error) {
	switch {
	case i.semaphore != nil:
		return getSemaphoreLockName(i.semaphore, wfNamespace)
	case i.mutex != nil:
		return getMutexLockName(i.mutex, wfNamespace), nil
	default:
		return nil, fmt.Errorf("cannot get lockName if not semaphore or mutex")
	}
}

func DecodeLockName(ctx context.Context, name string) (LockName, error) {
	log := logging.RequireLoggerFromContext(ctx)
	log.WithField("name", name).Info(ctx, "DecodeLockName")
	items := strings.SplitN(name, "/", 3)
	if len(items) < 3 {
		return nil, errors.New(errors.CodeBadRequest, "Invalid lock key: unknown format")
	}

	var lock lockName
	lockKind := lockKind(items[1])
	namespace := items[0]

	switch lockKind {
	case lockKindMutex, lockKindDatabase:
		lock = lockName{namespace: namespace, kind: lockKind, resourceName: items[2]}
	case lockKindConfigMap:
		components := strings.Split(items[2], "/")

		if len(components) != 2 {
			return nil, errors.New(errors.CodeBadRequest, "Invalid ConfigMap lock key: unknown format")
		}

		lock = lockName{namespace: namespace, kind: lockKind, resourceName: components[0], key: components[1]}
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("Invalid lock key, unexpected kind: %s", lockKind))
	}

	err := lock.validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func (ln *lockName) String(ctx context.Context) string {
	switch ln.kind {
	case lockKindMutex, lockKindDatabase:
		return ln.validateEncoding(ctx, fmt.Sprintf("%s/%s/%s", ln.namespace, ln.kind, ln.resourceName))
	default:
		return ln.validateEncoding(ctx, fmt.Sprintf("%s/%s/%s/%s", ln.namespace, ln.kind, ln.resourceName, ln.key))
	}
}

func (ln *lockName) validate() error {
	if ln.namespace == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Namespace is missing")
	}
	if ln.kind == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Kind is missing")
	}
	if ln.resourceName == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: ResourceName is missing")
	}
	if ln.kind == lockKindConfigMap && ln.key == "" {
		return errors.New(errors.CodeBadRequest, "Invalid lock key: Key is missing for ConfigMap lock")
	}
	return nil
}

func (ln *lockName) validateEncoding(ctx context.Context, encoding string) string {
	decoded, err := DecodeLockName(ctx, encoding)
	if err != nil {
		panic(fmt.Sprintf("bug: unable to decode lock (%s) that was just encoded: %s", encoding, err))
	}
	if ln.namespace != decoded.GetNamespace() || ln.resourceName != decoded.GetResourceName() || ln.key != decoded.GetKey() {
		panic("bug: lock that was just encoded does not match encoding")
	}
	return encoding
}

func (ln *lockName) getDBKey() string {
	return fmt.Sprintf("%s/%s", ln.namespace, ln.resourceName)
}

func needDBSession(ctx context.Context, lockKeys []string) (bool, error) {
	for _, key := range lockKeys {
		lock, err := DecodeLockName(ctx, key)
		if err != nil {
			return false, err
		}
		if lock.getKind() == lockKindDatabase {
			return true, nil
		}
	}
	return false, nil
}
