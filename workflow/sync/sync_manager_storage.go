package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "k8s.io/api/core/v1"
)

type syncManagerStorage struct {
	namespace  string
	name       string
	kubeClient kubernetes.Interface
	lock       sync.RWMutex
}

type SyncMetadataEntry struct {
	SyncTy SyncType `json:"syncType"`
}

type SyncManagerStorageError error

var (
	ConfigMapNotFound       SyncManagerStorageError = fmt.Errorf("Config map could not be found")
	KeyNotFound             SyncManagerStorageError = fmt.Errorf("Could not find key")
	FailedtoCreateEntry     SyncManagerStorageError = fmt.Errorf("Failed to create entry")
	FailedtoMarshal         SyncManagerStorageError = fmt.Errorf("Could not marshal")
	FailedtoUnMarshal       SyncManagerStorageError = fmt.Errorf("Could not unmarshal")
	FailedtoCreateConfigMap SyncManagerStorageError = fmt.Errorf("Failed to create config map")
)

func newSyncManagerStorage(ns string, ki kubernetes.Interface, name string) *syncManagerStorage {
	return &syncManagerStorage{
		namespace:  ns,
		name:       name,
		kubeClient: ki,
		lock:       sync.RWMutex{},
	}
}

func (c *syncManagerStorage) logError(err error, fields log.Fields, message string) {
	log.WithFields(log.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).WithError(err).Debug(message)
}

func (c *syncManagerStorage) logInfo(fields log.Fields, message string) {
	log.WithFields(log.Fields{"namespace": c.namespace, "name": c.name}).WithFields(fields).Info(message)
}

func (c *syncManagerStorage) Load(ctx context.Context, key string) (*SyncMetadataEntry, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			c.logError(err, log.Fields{}, "config map does not exist")
			return nil, ConfigMapNotFound
		}
		c.logError(err, log.Fields{"key": key}, "Error loading sync storage")
		return nil, ConfigMapNotFound
	}

	c.logInfo(log.Fields{"key": key}, "config map loaded")

	rawEntry, ok := cm.Data[key]
	if !ok {
		c.logInfo(log.Fields{"key": key}, "sync storage key not found")
		return nil, KeyNotFound
	}

	var entry SyncMetadataEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Failed to unmarshal")
		return nil, FailedtoUnMarshal
	}

	return &entry, nil
}

func (c *syncManagerStorage) Store(ctx context.Context, key string, syncType SyncType) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	db, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if err != nil || db == nil {
		if apierr.IsNotFound(err) || db == nil {
			db, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(ctx, &apiv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: c.name,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				c.logError(err, log.Fields{"key": key}, "Failed to create config map")
				return FailedtoCreateConfigMap
			}
		}
	}

	db.SetLabels(map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapSyncManager})
	newEntry := SyncMetadataEntry{SyncTy: syncType}

	entryJson, err := json.Marshal(newEntry)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Unable to marshal sync entry")
	}
	if db.Data == nil {
		db.Data = make(map[string]string)
	}

	db.Data[key] = string(entryJson)
	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(ctx, db, metav1.UpdateOptions{})

	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Kubernetes error creating new db entry")
		return FailedtoCreateEntry
	}

	return nil
}

func (c *syncManagerStorage) GetSyncType(ctx context.Context, key string) (*SyncType, error) {
	meta, err := c.Load(ctx, key)
	if err != nil {
		return nil, err
	}
	switch meta.SyncTy {
	case WorkflowLevel:
		ty := WorkflowLevel
		return &ty, nil
	case TemplateLevel:
		ty := TemplateLevel
		return &ty, nil
	default:
		return nil, fmt.Errorf("Invalid integer received of %d", meta.SyncTy)
	}
}
