package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/argoproj/argo-workflows/v4/pkg/storage/models"
	storageutil "github.com/argoproj/argo-workflows/v4/pkg/storage"
	watchutil "github.com/argoproj/argo-workflows/v4/pkg/storage/watch"
)

// StatusStore implements rest.Updater for /status subresources.
// It loads the current object, applies the update to .Status only, and saves.
type StatusStore struct {
	db           *gorm.DB
	watchManager *watchutil.Manager
	scheme       *runtime.Scheme
	gvr          schema.GroupVersionResource
	kind         string
	namespaced   bool
	newFunc      func() runtime.Object
}

var _ rest.Updater = &StatusStore{}
var _ rest.Getter = &StatusStore{}
var _ rest.Scoper = &StatusStore{}

// StatusStoreConfig holds configuration for creating a StatusStore.
type StatusStoreConfig struct {
	DB           *gorm.DB
	WatchManager *watchutil.Manager
	Scheme       *runtime.Scheme
	GVR          schema.GroupVersionResource
	Kind         string
	Namespaced   bool
	NewFunc      func() runtime.Object
}

// NewStatusStore creates a new StatusStore.
func NewStatusStore(cfg StatusStoreConfig) *StatusStore {
	return &StatusStore{
		db:           cfg.DB,
		watchManager: cfg.WatchManager,
		scheme:       cfg.Scheme,
		gvr:          cfg.GVR,
		kind:         cfg.Kind,
		namespaced:   cfg.Namespaced,
		newFunc:      cfg.NewFunc,
	}
}

func (s *StatusStore) New() runtime.Object {
	return s.newFunc()
}

func (s *StatusStore) Destroy() {}

func (s *StatusStore) NamespaceScoped() bool {
	return s.namespaced
}

func (s *StatusStore) GetSingularName() string {
	return s.gvr.Resource
}

// Get retrieves the current object (same as main store Get).
func (s *StatusStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace, _ := genericapirequest.NamespaceFrom(ctx)

	var record models.ResourceRecord
	q := s.db.Where("kind = ? AND name = ?", s.kind, name)
	if s.namespaced {
		q = q.Where("namespace = ?", namespace)
	}
	if err := q.First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound(s.gvr.GroupResource(), name)
		}
		return nil, errors.NewInternalError(err)
	}

	obj := s.newFunc()
	if err := json.Unmarshal([]byte(record.Data), obj); err != nil {
		return nil, errors.NewInternalError(err)
	}
	return obj, nil
}

// Update updates only the status subresource of the object.
func (s *StatusStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	namespace, _ := genericapirequest.NamespaceFrom(ctx)

	var result runtime.Object

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var record models.ResourceRecord
		q := tx.Where("kind = ? AND name = ?", s.kind, name)
		if s.namespaced {
			q = q.Where("namespace = ?", namespace)
		}
		if err := q.First(&record).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFound(s.gvr.GroupResource(), name)
			}
			return err
		}

		existing := s.newFunc()
		if err := json.Unmarshal([]byte(record.Data), existing); err != nil {
			return errors.NewInternalError(err)
		}

		updated, err := objInfo.UpdatedObject(ctx, existing)
		if err != nil {
			return err
		}

		updatedAccessor, err := meta.Accessor(updated)
		if err != nil {
			return err
		}

		// Optimistic concurrency check.
		if updatedAccessor.GetResourceVersion() != "" {
			clientRV, err := strconv.ParseInt(updatedAccessor.GetResourceVersion(), 10, 64)
			if err != nil {
				return errors.NewBadRequest("invalid resourceVersion")
			}
			if clientRV != record.ResourceVersion {
				return errors.NewConflict(s.gvr.GroupResource(), name, fmt.Errorf("the object has been modified"))
			}
		}

		if updateValidation != nil {
			if err := updateValidation(ctx, updated, existing); err != nil {
				return err
			}
		}

		rv, err := storageutil.NextResourceVersion(tx)
		if err != nil {
			return err
		}
		updatedAccessor.SetResourceVersion(strconv.FormatInt(rv, 10))

		data, err := json.Marshal(updated)
		if err != nil {
			return err
		}

		record.ResourceVersion = rv
		record.Data = string(data)
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		// Insert watch event.
		watchEvt := models.WatchEvent{
			Kind:            s.kind,
			Namespace:       namespace,
			Name:            name,
			UID:             record.UID,
			ResourceVersion: rv,
			EventType:       string(watch.Modified),
			Data:            string(data),
		}
		if err := tx.Create(&watchEvt).Error; err != nil {
			return err
		}

		result = updated
		return nil
	})
	if txErr != nil {
		return nil, false, txErr
	}

	accessor, _ := meta.Accessor(result)
	s.watchManager.Notify(s.kind, accessor.GetNamespace(), watch.Modified, result, 0)

	return result, false, nil
}

func (s *StatusStore) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(s.gvr.GroupResource()).ConvertToTable(ctx, obj, tableOptions)
}
