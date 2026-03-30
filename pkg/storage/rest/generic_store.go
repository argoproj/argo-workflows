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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apimachinery/pkg/util/uuid"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"

	"github.com/argoproj/argo-workflows/v4/pkg/storage/models"
	"github.com/argoproj/argo-workflows/v4/pkg/storage/query"
	storageutil "github.com/argoproj/argo-workflows/v4/pkg/storage"
	watchutil "github.com/argoproj/argo-workflows/v4/pkg/storage/watch"
)

// GenericStore implements the k8s apiserver registry REST interfaces backed by SQL via GORM.
type GenericStore struct {
	db           *gorm.DB
	watchManager *watchutil.Manager
	scheme       *runtime.Scheme
	gvr          schema.GroupVersionResource
	gvk          schema.GroupVersionKind
	listGVK      schema.GroupVersionKind
	newFunc      func() runtime.Object
	newListFunc  func() runtime.Object
	kind         string
	namespaced   bool
}

// StoreConfig holds configuration for creating a GenericStore.
type StoreConfig struct {
	DB           *gorm.DB
	WatchManager *watchutil.Manager
	Scheme       *runtime.Scheme
	GVR          schema.GroupVersionResource
	GVK          schema.GroupVersionKind
	ListGVK      schema.GroupVersionKind
	NewFunc      func() runtime.Object
	NewListFunc  func() runtime.Object
	Kind         string
	Namespaced   bool
}

// NewGenericStore creates a new GenericStore.
func NewGenericStore(cfg StoreConfig) *GenericStore {
	return &GenericStore{
		db:           cfg.DB,
		watchManager: cfg.WatchManager,
		scheme:       cfg.Scheme,
		gvr:          cfg.GVR,
		gvk:          cfg.GVK,
		listGVK:      cfg.ListGVK,
		newFunc:      cfg.NewFunc,
		newListFunc:  cfg.NewListFunc,
		kind:         cfg.Kind,
		namespaced:   cfg.Namespaced,
	}
}

var _ rest.Getter = &GenericStore{}
var _ rest.Lister = &GenericStore{}
var _ rest.Creater = &GenericStore{}
var _ rest.Updater = &GenericStore{}
var _ rest.GracefulDeleter = &GenericStore{}
var _ rest.Watcher = &GenericStore{}
var _ rest.Scoper = &GenericStore{}

func (s *GenericStore) New() runtime.Object {
	return s.newFunc()
}

func (s *GenericStore) Destroy() {}

func (s *GenericStore) NewList() runtime.Object {
	return s.newListFunc()
}

func (s *GenericStore) NamespaceScoped() bool {
	return s.namespaced
}

func (s *GenericStore) GetSingularName() string {
	return s.gvr.Resource
}

func (s *GenericStore) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(s.gvr.GroupResource()).ConvertToTable(ctx, obj, tableOptions)
}

// Get retrieves a single resource by namespace and name.
func (s *GenericStore) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericNamespace(ctx)

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

	return s.deserialize(record.Data)
}

// List returns a list of resources matching the given options.
func (s *GenericStore) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	namespace := genericNamespace(ctx)

	q := s.db.Model(&models.ResourceRecord{}).Where("kind = ?", s.kind)
	if s.namespaced && namespace != "" {
		q = q.Where("namespace = ?", namespace)
	}

	// Apply label selector.
	if options != nil && options.LabelSelector != nil {
		q = query.ApplyLabelSelector(q, options.LabelSelector)
	}

	// Apply field selector.
	if options != nil && options.FieldSelector != nil {
		q = query.ApplyFieldSelector(q, options.FieldSelector)
	}

	// Apply limit/continue (offset-based).
	if options != nil && options.Limit > 0 {
		q = q.Limit(int(options.Limit))
	}
	if options != nil && options.Continue != "" {
		offset, err := strconv.Atoi(options.Continue)
		if err != nil {
			return nil, errors.NewBadRequest("invalid continue token")
		}
		q = q.Offset(offset)
	}

	q = q.Order("resource_version ASC")

	var records []models.ResourceRecord
	if err := q.Find(&records).Error; err != nil {
		return nil, errors.NewInternalError(err)
	}

	return s.buildList(records)
}

// Create stores a new resource.
func (s *GenericStore) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	namespace := accessor.GetNamespace()
	name := accessor.GetName()
	if name == "" {
		if accessor.GetGenerateName() != "" {
			name = accessor.GetGenerateName() + string(uuid.NewUUID())[:5]
			accessor.SetName(name)
		} else {
			return nil, errors.NewBadRequest("name is required")
		}
	}

	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	// Set metadata.
	uid := string(uuid.NewUUID())
	accessor.SetUID(types.UID(uid))
	now := metav1.Now()
	accessor.SetCreationTimestamp(now)
	accessor.SetGeneration(1)

	var result runtime.Object
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		rv, err := storageutil.NextResourceVersion(tx)
		if err != nil {
			return err
		}
		accessor.SetResourceVersion(strconv.FormatInt(rv, 10))

		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		record := models.ResourceRecord{
			Kind:            s.kind,
			Namespace:       namespace,
			Name:            name,
			UID:             uid,
			ResourceVersion: rv,
			Generation:      1,
			Data:            string(data),
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// Insert labels.
		for k, v := range accessor.GetLabels() {
			label := models.ResourceLabel{
				ResourceID: record.ID,
				Key:        k,
				Value:      v,
			}
			if err := tx.Create(&label).Error; err != nil {
				return err
			}
		}

		// Insert watch event.
		watchEvt := models.WatchEvent{
			Kind:            s.kind,
			Namespace:       namespace,
			Name:            name,
			UID:             uid,
			ResourceVersion: rv,
			EventType:       string(watch.Added),
			Data:            string(data),
		}
		if err := tx.Create(&watchEvt).Error; err != nil {
			return err
		}

		result = obj
		return nil
	})
	if txErr != nil {
		return nil, errors.NewInternalError(txErr)
	}

	// Notify watchers.
	s.watchManager.Notify(s.kind, namespace, watch.Added, result, 0)

	return result, nil
}

// Update modifies an existing resource with optimistic concurrency.
func (s *GenericStore) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	namespace := genericNamespace(ctx)

	var result runtime.Object
	var created bool

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// Load existing record.
		var record models.ResourceRecord
		q := tx.Where("kind = ? AND name = ?", s.kind, name)
		if s.namespaced {
			q = q.Where("namespace = ?", namespace)
		}
		err := q.First(&record).Error

		if err == gorm.ErrRecordNotFound {
			if !forceAllowCreate {
				return errors.NewNotFound(s.gvr.GroupResource(), name)
			}
			// Create via update (PUT-create).
			existing := s.newFunc()
			updated, err := objInfo.UpdatedObject(ctx, existing)
			if err != nil {
				return err
			}
			if createValidation != nil {
				if err := createValidation(ctx, updated); err != nil {
					return err
				}
			}
			result, err = s.Create(ctx, updated, nil, nil)
			created = true
			return err
		}
		if err != nil {
			return err
		}

		existing, err := s.deserialize(record.Data)
		if err != nil {
			return err
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
				return errors.NewConflict(s.gvr.GroupResource(), name, fmt.Errorf("the object has been modified; please apply your changes to the latest version"))
			}
		}

		if updateValidation != nil {
			if err := updateValidation(ctx, updated, existing); err != nil {
				return err
			}
		}

		// Increment resource version.
		rv, err := storageutil.NextResourceVersion(tx)
		if err != nil {
			return err
		}
		updatedAccessor.SetResourceVersion(strconv.FormatInt(rv, 10))
		updatedAccessor.SetGeneration(record.Generation + 1)

		data, err := json.Marshal(updated)
		if err != nil {
			return err
		}

		record.ResourceVersion = rv
		record.Generation = record.Generation + 1
		record.Data = string(data)
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		// Sync labels: delete old, insert new.
		if err := tx.Where("resource_id = ?", record.ID).Delete(&models.ResourceLabel{}).Error; err != nil {
			return err
		}
		for k, v := range updatedAccessor.GetLabels() {
			label := models.ResourceLabel{
				ResourceID: record.ID,
				Key:        k,
				Value:      v,
			}
			if err := tx.Create(&label).Error; err != nil {
				return err
			}
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

	if !created {
		accessor, _ := meta.Accessor(result)
		s.watchManager.Notify(s.kind, accessor.GetNamespace(), watch.Modified, result, 0)
	}

	return result, created, nil
}

// Delete removes a resource.
func (s *GenericStore) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	namespace := genericNamespace(ctx)

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

		existing, err := s.deserialize(record.Data)
		if err != nil {
			return err
		}

		if deleteValidation != nil {
			if err := deleteValidation(ctx, existing); err != nil {
				return err
			}
		}

		rv, err := storageutil.NextResourceVersion(tx)
		if err != nil {
			return err
		}

		// Delete labels (cascade should handle this, but be explicit).
		if err := tx.Where("resource_id = ?", record.ID).Delete(&models.ResourceLabel{}).Error; err != nil {
			return err
		}

		// Delete the record (hard delete, not soft delete for now).
		if err := tx.Unscoped().Delete(&record).Error; err != nil {
			return err
		}

		// Insert DELETED watch event.
		watchEvt := models.WatchEvent{
			Kind:            s.kind,
			Namespace:       namespace,
			Name:            name,
			UID:             record.UID,
			ResourceVersion: rv,
			EventType:       string(watch.Deleted),
			Data:            record.Data,
		}
		if err := tx.Create(&watchEvt).Error; err != nil {
			return err
		}

		result = existing
		return nil
	})
	if txErr != nil {
		return nil, false, txErr
	}

	accessor, _ := meta.Accessor(result)
	s.watchManager.Notify(s.kind, accessor.GetNamespace(), watch.Deleted, result, 0)

	return result, true, nil
}

// Watch returns a watch.Interface that watches for changes.
func (s *GenericStore) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	namespace := genericNamespace(ctx)

	var rv int64
	if options != nil && options.ResourceVersion != "" {
		var err error
		rv, err = strconv.ParseInt(options.ResourceVersion, 10, 64)
		if err != nil {
			return nil, errors.NewBadRequest("invalid resourceVersion")
		}
	}

	return s.watchManager.Watch(s.kind, namespace, rv, s.scheme)
}

func (s *GenericStore) deserialize(data string) (runtime.Object, error) {
	obj := s.newFunc()
	if err := json.Unmarshal([]byte(data), obj); err != nil {
		return nil, fmt.Errorf("failed to deserialize resource: %w", err)
	}
	// Ensure TypeMeta is always populated so kubectl and API consumers get kind/apiVersion.
	s.setTypeMeta(obj)
	return obj, nil
}

// setTypeMeta sets the Kind and APIVersion on obj using the store's known GVK.
func (s *GenericStore) setTypeMeta(obj runtime.Object) {
	type typeSetter interface {
		SetGroupVersionKind(schema.GroupVersionKind)
	}
	if ts, ok := obj.(typeSetter); ok {
		ts.SetGroupVersionKind(s.gvk)
	}
}

func (s *GenericStore) buildList(records []models.ResourceRecord) (runtime.Object, error) {
	list := s.newListFunc()

	// Set TypeMeta on the list itself.
	type typeSetter interface {
		SetGroupVersionKind(schema.GroupVersionKind)
	}
	if ts, ok := list.(typeSetter); ok {
		ts.SetGroupVersionKind(s.listGVK)
	}

	items := make([]runtime.Object, 0, len(records))
	for _, r := range records {
		obj, err := s.deserialize(r.Data)
		if err != nil {
			return nil, err
		}
		items = append(items, obj)
	}

	if err := meta.SetList(list, items); err != nil {
		return nil, fmt.Errorf("failed to set list items: %w", err)
	}

	return list, nil
}

// genericNamespace extracts the namespace from the apiserver request context.
func genericNamespace(ctx context.Context) string {
	ns, _ := genericapirequest.NamespaceFrom(ctx)
	return ns
}
