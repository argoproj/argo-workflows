package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	kwait "k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoerr "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type configMapCache struct {
	namespace  string
	name       string
	kubeClient kubernetes.Interface
	lock       sync.RWMutex
}

func NewConfigMapCache(ns string, ki kubernetes.Interface, n string) MemoizationCache {
	return &configMapCache{
		namespace:  ns,
		name:       n,
		kubeClient: ki,
		lock:       sync.RWMutex{},
	}
}

func (c *configMapCache) logError(ctx context.Context, err error, fields logging.Fields, message string) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).WithError(err).Debug(ctx, message)
}

func (c *configMapCache) logInfo(ctx context.Context, fields logging.Fields, message string) {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).Info(ctx, message)
}

func (c *configMapCache) validateConfigmap(ctx context.Context, cm *apiv1.ConfigMap) error {
	label, foundLabel := cm.GetLabels()[common.LabelKeyConfigMapType]
	errString := ""
	if !foundLabel {
		errString = fmt.Sprintf("memoization configmap doesn't have %s label, refusing to use it", common.LabelKeyConfigMapType)
	} else if label != common.LabelValueTypeConfigMapCache {
		errString = fmt.Sprintf("memoization configmap doesn't have label %s = %s, refusing to use it", common.LabelKeyConfigMapType, common.LabelValueTypeConfigMapCache)
	}
	if errString != "" {
		err := errors.New(errString)
		c.logError(ctx, err, logging.Fields{}, errString)
		return err
	}
	return nil
}

func (c *configMapCache) Load(ctx context.Context, key string) (*Entry, error) {
	var entry *Entry
	err := retry.OnError(kwait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    5,
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return argoerr.IsTransientErr(ctx, err) || (apierr.IsConflict(err) && strings.Contains(err.Error(), ""))
	}, func() error {
		var innerErr error
		entry, innerErr = c.load(ctx, key)
		return innerErr
	})
	return entry, err
}

func (c *configMapCache) load(ctx context.Context, key string) (*Entry, error) {
	if !cacheKeyRegex.MatchString(key) {
		return nil, fmt.Errorf("invalid cache key: %s", key)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			c.logError(ctx, err, logging.Fields{}, "config map cache miss: config map does not exist")
			return nil, nil
		}
		return nil, err
	}
	err = c.validateConfigmap(ctx, cm)
	if err != nil {
		return nil, err
	}

	c.logInfo(ctx, logging.Fields{}, "config map cache loaded")
	hitTime := time.Now()
	rawEntry, ok := cm.Data[key]
	if !ok || rawEntry == "" {
		c.logInfo(ctx, logging.Fields{}, "config map cache miss: entry does not exist")
		return nil, nil
	}

	var entry Entry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		return nil, fmt.Errorf("malformed cache entry: could not unmarshal JSON; unable to parse: %w", err)
	}

	entry.LastHitTimestamp = metav1.Time{Time: hitTime}
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		c.logError(ctx, err, logging.Fields{"key": key}, "Unable to marshal cache entry with last hit timestamp")
		return nil, fmt.Errorf("unable to marshal cache entry with last hit timestamp: %w", err)
	}
	cm.Data[key] = string(entryJSON)

	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *configMapCache) Save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs) error {
	err := retry.OnError(kwait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    5,
		Cap:      30 * time.Second,
	}, func(err error) bool {
		return argoerr.IsTransientErr(ctx, err) || (apierr.IsConflict(err) && strings.Contains(err.Error(), "Operation cannot be fulfilled on configmaps"))
	}, func() error {
		var innerErr = c.save(ctx, key, nodeID, value)
		return innerErr
	})
	return err
}

func (c *configMapCache) save(ctx context.Context, key string, nodeID string, value *wfv1.Outputs) error {
	if !cacheKeyRegex.MatchString(key) {
		errString := fmt.Sprintf("invalid cache key: %s", key)
		err := errors.New(errString)
		c.logError(ctx, err, logging.Fields{"key": key}, errString)
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.logInfo(ctx, logging.Fields{"key": key, "nodeID": nodeID}, "Saving ConfigMap cache entry")

	cache, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if apierr.IsNotFound(err) || cache == nil {
		cache, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(ctx, &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.name,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			c.logError(ctx, err, logging.Fields{"key": key, "nodeID": nodeID}, "Error saving to ConfigMap cache")
			return fmt.Errorf("could not save to config map cache: %w", err)
		}
	} else {
		err := c.validateConfigmap(ctx, cache)
		if err != nil {
			return err
		}
	}

	creationTime := time.Now()
	cache.SetLabels(map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache})

	newEntry := Entry{
		NodeID:            nodeID,
		Outputs:           value,
		CreationTimestamp: metav1.Time{Time: creationTime},
		LastHitTimestamp:  metav1.Time{Time: creationTime},
	}

	entryJSON, err := json.Marshal(newEntry)
	if err != nil {
		c.logError(ctx, err, logging.Fields{"key": key, "nodeID": nodeID}, "Unable to marshal cache entry")
		return fmt.Errorf("unable to marshal cache entry: %w", err)
	}

	if cache.Data == nil {
		cache.Data = make(map[string]string)
	}
	cache.Data[key] = string(entryJSON)

	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(ctx, cache, metav1.UpdateOptions{})
	if err != nil {
		c.logError(ctx, err, logging.Fields{"key": key, "nodeID": nodeID}, "Kubernetes error creating new cache entry")
		return fmt.Errorf("error creating cache entry: %w. Please check out this page for help: https://argo-workflows.readthedocs.io/en/latest/memoization/#faqs", err)
	}
	return nil
}
