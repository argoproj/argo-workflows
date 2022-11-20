package sync

import (
	"context"
	"encoding/hex"
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
	lock       sync.Mutex
	pending    map[string](map[string]bool)
}

type SyncMetadataEntry struct {
	Key     string                       `json:"key"`
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
	InvalidMutexHolders     SyncManagerStorageError = fmt.Errorf("A Mutex may not have more than 1 holder")
)

func newSyncManagerStorage(ns string, ki kubernetes.Interface, name string) *syncManagerStorage {
	log.Infof("Creating new sync manager storage on namespace %s with name %s", ns, name)
	return &syncManagerStorage{
		namespace:  ns,
		name:       name,
		kubeClient: ki,
		lock:       sync.Mutex{},
		pending:    make(map[string]map[string]bool),
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
	hexKey := hex.EncodeToString([]byte(key))

	cm, err := c.getDB(ctx)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Error loading sync storage")
		return nil, false, ConfigMapNotFound
	}

	c.logInfo(log.Fields{"key": key}, "config map loaded")

	rawEntry, ok := cm.Data[hexKey]
	if !ok {
		c.logInfo(log.Fields{"key": key, "hexKey": hexKey}, "sync storage key not found")
		return nil, false, KeyNotFound
	}

	var entry SyncMetadataEntry
	err = json.Unmarshal([]byte(rawEntry), &entry)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Failed to unmarshal")
		return nil, true, FailedtoUnMarshal
	}

	c.logInfo(log.Fields{"key": key, "hexKey": hexKey}, "Loaded sync metadata")

	return &entry, true, nil
}

func (c *syncManagerStorage) Store(ctx context.Context, key string, holders []string, syncTy v1alpha1.SynchronizationType) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.store(ctx, key, holders, syncTy)
}

func (c *syncManagerStorage) store(ctx context.Context, key string, holders []string, syncTy v1alpha1.SynchronizationType) error {
	hexKey := hex.EncodeToString([]byte(key))
	db, err := c.getDB(ctx)
	if err != nil {
		return err
	}
	if syncTy == v1alpha1.SynchronizationTypeMutex && len(holders) > 1 {
		return InvalidMutexHolders
	}

	newEntry := SyncMetadataEntry{Key: key, Holders: holders, LockTy: syncTy}

	entryJson, err := json.Marshal(newEntry)
	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Unable to marshal sync entry")
	}
	if db.Data == nil {
		db.Data = make(map[string]string)
	}

	db.Data[hexKey] = string(entryJson)
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

func (c *syncManagerStorage) getDB(ctx context.Context) (*apiv1.ConfigMap, error) {
	db, err := c.kubeClient.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if err != nil || db == nil {
		if apierr.IsNotFound(err) || db == nil {
			db, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Create(ctx, &apiv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: c.name,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				log.Warnf("Failed to create config map for due to %s", err.Error())
				c.logError(err, log.Fields{}, "Failed to create config map")
				return nil, FailedtoCreateConfigMap
			}
		}
	}
	db.SetLabels(map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapSyncManager})
	if db.Data == nil {
		db.Data = make(map[string]string)
	}
	return db, nil
}

func (c *syncManagerStorage) DeleteLock(ctx context.Context, key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.deleteLock(ctx, key)
}

func (c *syncManagerStorage) deleteLock(ctx context.Context, key string) error {
	hexKey := hex.EncodeToString([]byte(key))
	db, err := c.getDB(ctx)
	delete(db.Data, hexKey)

	_, err = c.kubeClient.CoreV1().ConfigMaps(c.namespace).Update(ctx, db, metav1.UpdateOptions{})

	if err != nil {
		c.logError(err, log.Fields{"key": key}, "Kubernetes error creating new db entry")
		return FailedtoUpdateMap
	}

	return nil
}

func (c *syncManagerStorage) deleteLockHolders(ctx context.Context, key string, holders []string) error {
	c.logInfo(log.Fields{"key": key}, "Deleting holders")
	entry, _, err := c.load(ctx, key)
	if err != nil && err != KeyNotFound {
		return err
	}
	if err == KeyNotFound {
		return nil
	}
	newHolders := filterHolders(holders, entry.Holders)
	if len(newHolders) == 0 {
		return c.deleteLock(ctx, key)
	}
	return c.store(ctx, key, newHolders, entry.LockTy)
}

func (c *syncManagerStorage) AddToQueue(ctx context.Context, key string, holders []string) error {
	return nil
}

func (c *syncManagerStorage) RemoveFromQueue(ctx context.Context, key string) (string, error) {
	return "", nil
}
