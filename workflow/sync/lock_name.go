package sync

import (
	"fmt"
	"regexp"
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
	Selectors    []v1alpha1.SyncSelector
}

func NewLockName(namespace, resourceName, lockKey string, kind LockKind, selectors []v1alpha1.SyncSelector) *LockName {
	return &LockName{
		Namespace:    namespace,
		ResourceName: resourceName,
		Selectors:    selectors,
		Key:          lockKey,
		Kind:         kind,
	}
}

func GetLockName(sync *v1alpha1.Synchronization, namespace string) (*LockName, error) {
	switch sync.GetType() {
	case v1alpha1.SynchronizationTypeSemaphore:
		if sync.Semaphore.ConfigMapKeyRef != nil {
			return NewLockName(namespace, sync.Semaphore.ConfigMapKeyRef.Name, sync.Semaphore.ConfigMapKeyRef.Key, LockKindConfigMap, sync.Semaphore.Selectors), nil
		}
		return nil, fmt.Errorf("cannot get LockName for a Semaphore without a ConfigMapRef")
	case v1alpha1.SynchronizationTypeMutex:
		return NewLockName(namespace, sync.Mutex.Name, "", LockKindMutex, sync.Mutex.Selectors), nil
	default:
		return nil, fmt.Errorf("cannot get LockName for a Sync of Unknown type")
	}
}

func DecodeLockName(lockName string) (*LockName, error) {
	splittedLockName := strings.Split(lockName, "?")
	lockNameTrimedSelectors := splittedLockName[0]
	selectors := ParseSelectors(strings.Join(splittedLockName[1:], "?"))
	items := strings.SplitN(lockNameTrimedSelectors, "/", 3)
	if len(items) < 3 {
		return nil, errors.New(errors.CodeBadRequest, "Invalid lock key: unknown format")
	}

	var lock LockName
	lockKind := LockKind(items[1])
	namespace := items[0]

	switch lockKind {
	case LockKindMutex:
		lock = LockName{Namespace: namespace, Kind: lockKind, ResourceName: items[2], Selectors: selectors}
	case LockKindConfigMap:
		components := strings.Split(items[2], "/")

		if len(components) != 2 {
			return nil, errors.New(errors.CodeBadRequest, "Invalid ConfigMap lock key: unknown format")
		}

		lock = LockName{Namespace: namespace, Kind: lockKind, ResourceName: components[0], Key: components[1], Selectors: selectors}
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("Invalid lock key, unexpected kind: %s", lockKind))
	}

	err := lock.Validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func StringifySelectors(selectors []v1alpha1.SyncSelector) string {
	joinedSelectors := ""
	for _, selector := range selectors {
		// at this point template should be already replaced
		if selector.Template != "" {
			// escape & and = chars to decode easily later
			re := regexp.MustCompile("&|=")
			escapedSelectorName := re.ReplaceAllString(selector.Name, "-")
			escapedSelectorValue := re.ReplaceAllString(selector.Template, "-")

			joinedSelectors = joinedSelectors + fmt.Sprintf("%s=%s&", escapedSelectorName, escapedSelectorValue)
		}
	}
	return strings.TrimRight(joinedSelectors, "&")
}

func ParseSelectors(selectors string) []v1alpha1.SyncSelector {
	parsedSelectors := []v1alpha1.SyncSelector{}
	splittedSelectors := strings.Split(selectors, "&")

	for _, selectorStr := range splittedSelectors {
		keyValPair := strings.Split(selectorStr, "=")
		if len(keyValPair) == 2 {
			parsedSelectors = append(parsedSelectors, v1alpha1.SyncSelector{
				Name:     keyValPair[0],
				Template: keyValPair[1],
			})
		}
		// otherwise consider invalid, do nothing
	}
	return parsedSelectors
}

func (ln *LockName) EncodeName() string {
	encodingBuilder := &strings.Builder{}

	encodingBuilder.WriteString(fmt.Sprintf("%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName))
	if ln.Kind == LockKindConfigMap {
		encodingBuilder.WriteString(fmt.Sprintf("/%s", ln.Key))
	}
	if selectors := StringifySelectors(ln.Selectors); len(selectors) > 0 {
		encodingBuilder.WriteString(fmt.Sprintf("?%s", selectors))
	}
	return ln.ValidateEncoding(encodingBuilder.String())
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
