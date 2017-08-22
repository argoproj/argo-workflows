package volume

import (
	"encoding/json"
	"fmt"
	"strings"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

// StorageProvider data structure
type StorageProvider struct {
	ID       string                 `json:"id,omitempty" description:"uuid"`
	Name     string                 `json:"name,omitempty" description:"name"`
	Ctime    int64                  `json:"ctime,omitempty" description:"creation time in seconds since epoch"`
	Mtime    int64                  `json:"mtime,omitempty" description:"modification time in seconds since epoch"`
	SPSchema map[string]interface{} `json:"sp_schema,omitempty"  description:"storage provider schema"`
}

// storageProviderDB is a internal structure used for serializing storage providers to axdb
type storageProviderDB struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Ctime    int64  `json:"ctime,omitempty"`
	Mtime    int64  `json:"mtime,omitempty"`
	SPSchema string `json:"sp_schema,omitempty"`
}

// GetStorageProviders returns a slice of storage providers filtered by the given parameters
func GetStorageProviders(params map[string]interface{}) ([]StorageProvider, *axerror.AXError) {
	storageProviderDBs := []storageProviderDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, StorageProviderTableName, params, &storageProviderDBs)
	if axErr != nil {
		return nil, axErr
	}
	storageProviders := make([]StorageProvider, len(storageProviderDBs))
	for i := range storageProviderDBs {
		sp := storageProviderDBs[i].StorageProvider()
		storageProviders[i] = *sp
	}
	return storageProviders, nil
}

// GetStorageProviderByID returns a single storage provider instance by its ID, or nil if it does not exist
func GetStorageProviderByID(id string) (*StorageProvider, *axerror.AXError) {
	storageProviders, axErr := GetStorageProviders(map[string]interface{}{
		StorageProviderID: id,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(storageProviders) == 0 {
		return nil, nil
	}
	if len(storageProviders) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Found multiple storage providers with ID: %s", id))
	}
	sp := storageProviders[0]
	return &sp, nil
}

// create inserts a new a storage provider to the database
func (sp *StorageProvider) create() *axerror.AXError {
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, StorageProviderTableName, sp.storageProviderDB())
	if axErr != nil {
		return axErr
	}
	return nil
}

// update updates a storage provider to the database
func (sp *StorageProvider) update() *axerror.AXError {
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, StorageProviderTableName, sp.storageProviderDB())
	if axErr != nil {
		return axErr
	}
	return nil
}

// Creates or updates a storage provider
// This is only intended to be used by axops_initializer to insert our pre-created providers/classes)
// This is not thread safe.
func (sp *StorageProvider) upsert() *axerror.AXError {
	axErr := sp.create()
	if axErr != nil {
		if axErr.Code == axerror.ERR_AXDB_INSERT_DUPLICATE.Code {
			dupAxErr := axErr
			existingSP, axErr := GetStorageProviderByID(sp.ID)
			if axErr != nil {
				return axErr
			}
			if existingSP == nil {
				// If we had ERR_AXDB_INSERT_DUPLICATE during create and subsequently failed to get
				// the storage provider by the this ID, it indicates the caller is attempting to upsert
				// a new entry with the same name as the previous. Raise original duplicate error.
				return dupAxErr
			}
			existingSP.SPSchema = sp.SPSchema
			existingSP.Name = sp.Name
			existingSP.Mtime = sp.Mtime
			return existingSP.update()
		}
		return axErr
	}
	return nil
}

// storageProviderDB returns a storage class db instance suitable for storing into axdb
func (sp *StorageProvider) storageProviderDB() *storageProviderDB {
	spSchema := ""
	if sp.SPSchema != nil {
		schemaBytes, err := json.Marshal(sp.SPSchema)
		if err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to marshal the storage provider schema: %v", err))
			return nil
		}
		spSchema = string(schemaBytes)
	}
	spDB := &storageProviderDB{
		ID:       sp.ID,
		Name:     strings.ToLower(sp.Name),
		Ctime:    sp.Ctime * 1e6,
		Mtime:    sp.Mtime * 1e6,
		SPSchema: spSchema,
	}
	return spDB
}

func (spdb *storageProviderDB) StorageProvider() *StorageProvider {
	var spSchema map[string]interface{}
	if spdb.SPSchema != "" {
		if err := json.Unmarshal([]byte(spdb.SPSchema), &spSchema); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to unmarshal the schema string in storage provider:%v", err))
			return nil
		}
	}
	sp := &StorageProvider{
		ID:       spdb.ID,
		Name:     strings.ToLower(spdb.Name),
		Ctime:    spdb.Ctime / 1e6,
		Mtime:    spdb.Mtime / 1e6,
		SPSchema: spSchema,
	}
	return sp
}
