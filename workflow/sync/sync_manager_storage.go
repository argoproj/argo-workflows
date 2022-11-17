package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	LockTy  v1alpha1.SynchronizationType `json:"lockTy"`
	Holders []string                     `json:"holders"`
}

type SyncManagerStorageError error

var (
	ConfigMapNotFound       SyncManagerStorageError = fmt.Errorf("Config map could not be found")
	KeyNotFound             SyncManagerStorageError = fmt.Errorf("Could not find key")
	FailedtoUpdateMap       SyncManagerStorageError = fmt.Errorf("Failed to create entry")
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

func (c *syncManagerStorage) Load(ctx context.Context, key string) (*SyncMetadataEntry, bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.load(ctx, key)
}

func (c *syncManagerStorage) load(ctx context.Context, key string) (*SyncMetadataEntry, bool, error) {
	cm, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			c.logError(err, log.Fields{}, "config map does not exist")
			return nil, false, ConfigMapNotFound
		}
		c.logError(err, log.Fields{"key": key}, "Error loading sync storage")
		return nil, false, ConfigMapNotFound
	}

	c.logInfo(log.Fields{"key": key}, "config map loaded")

	rawEntry, ok := cm.Data[key]
	if !ok {
		c.logInfo(log.Fields{"key": key}, "sync storage key not found")
		return nil, false, KeyNotFound
	}

	var entry SyncMetadataEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Failed to unmarshal")
		return nil, true, FailedtoUnMarshal
	}

	return &entry, true, nil
}

func (c *syncManagerStorage) Store(ctx context.Context, key string, holders []string, syncTy v1alpha1.SynchronizationType) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.store(ctx, key, holders, syncTy)
}

func (c *syncManagerStorage) store(ctx context.Context, key string, holders []string, syncTy v1alpha1.SynchronizationType) error {
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
	newEntry := SyncMetadataEntry{Holders: holders, LockTy: syncTy}

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
		return FailedtoUpdateMap
	}

	return nil
}

func (c *syncManagerStorage) DeleteLockHolders(ctx context.Context, key string, holders []string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.deleteLockHolders(ctx, key, holders)
}

func filterHolders(toFilter []string, fullSet []string) []string {
	var newHolders []string
	set := make(map[string]bool)
	for _, holder := range toFilter {
		set[holder] = true
	}

	for _, holder := range fullSet {
		_, ok := set[holder]
		if !ok {
			newHolders = append(newHolders, holder)
		}
	}
	return newHolders
}

func (c *syncManagerStorage) deleteLockHolders(ctx context.Context, key string, holders []string) error {
	entry, _, err := c.load(ctx, key)
	if err != nil {
		return err
	}
	newHolders := filterHolders(holders, entry.Holders)
	return c.store(ctx, key, newHolders, entry.LockTy)
}
