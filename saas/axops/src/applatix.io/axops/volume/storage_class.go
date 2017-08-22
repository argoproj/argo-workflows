package volume

import (
	"encoding/json"
	"fmt"
	"strings"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

// StorageClass data structure
type StorageClass struct {
	ID          string                 `json:"id,omitempty" description:"uuid"`
	Name        string                 `json:"name,omitempty" description:"name"`
	Ctime       int64                  `json:"ctime,omitempty" description:"creation time in seconds since epoch"`
	Mtime       int64                  `json:"mtime,omitempty" description:"modification time in seconds since epoch"`
	Description string                 `json:"description,omitempty" description:"storage class description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" description:"storage class parameters"`
	Axrn        string                 `json:"axrn,omitempty"`
}

// storageClassDB is an internal structure used for serializing storage classes to axdb
type storageClassDB struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Ctime       int64  `json:"ctime,omitempty"`
	Mtime       int64  `json:"mtime,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  string `json:"parameters,omitempty"`
}

// GetStorageClasses returns a slice of storage classes filtered by the given parameters
func GetStorageClasses(params map[string]interface{}) ([]StorageClass, *axerror.AXError) {
	storageClassDbs := []storageClassDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, StorageClassTableName, params, &storageClassDbs)
	if axErr != nil {
		return nil, axErr
	}
	storageClasses := make([]StorageClass, len(storageClassDbs))
	for i := range storageClassDbs {
		sc := storageClassDbs[i].StorageClass()
		storageClasses[i] = *sc
	}
	return storageClasses, nil
}

// GetStorageClassByID returns a single storage class instance by its ID, or nil if it does not exist
func GetStorageClassByID(id string) (*StorageClass, *axerror.AXError) {
	storageClasses, axErr := GetStorageClasses(map[string]interface{}{
		StorageClassID: id,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(storageClasses) == 0 {
		return nil, nil
	}
	if len(storageClasses) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Found multiple storage classes with ID: %s", id))
	}
	sc := storageClasses[0]
	return &sc, nil
}

// GetStorageClassByName returns a single storage class instance by its name, or nil if it does not exist
func GetStorageClassByName(name string) (*StorageClass, *axerror.AXError) {
	storageClasses, axErr := GetStorageClasses(map[string]interface{}{
		StorageClassName: strings.ToLower(name),
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(storageClasses) == 0 {
		return nil, nil
	}
	if len(storageClasses) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Found multiple storage classes with name: %s", name))
	}
	sc := storageClasses[0]
	return &sc, nil
}

// create inserts a new a storage class to the database
func (sc *StorageClass) create() *axerror.AXError {
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, StorageClassTableName, sc.storageClassDB())
	if axErr != nil {
		return axErr
	}
	return nil
}

// update updates a storage class to the database
func (sc *StorageClass) update() *axerror.AXError {
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, StorageClassTableName, sc.storageClassDB())
	if axErr != nil {
		return axErr
	}
	return nil
}

// Creates or updates a storage class
// This is only intended to be used by axops_initializer to insert our pre-created providers/classes)
// This is not thread safe.
func (sc *StorageClass) upsert() *axerror.AXError {
	axErr := sc.create()
	if axErr != nil {
		if axErr.Code == axerror.ERR_AXDB_INSERT_DUPLICATE.Code {
			dupAxErr := axErr
			existingSP, axErr := GetStorageClassByID(sc.ID)
			if axErr != nil {
				return axErr
			}
			if existingSP == nil {
				// If we had ERR_AXDB_INSERT_DUPLICATE during create and subsequently failed to get
				// the storage class by the this ID, it indicates the caller is attempting to upsert
				// a new entry with the same name as the previous. Raise original duplicate error.
				return dupAxErr
			}
			existingSP.Name = sc.Name
			existingSP.Description = sc.Description
			existingSP.Mtime = sc.Mtime
			existingSP.Parameters = sc.Parameters
			return existingSP.update()
		}
		return axErr
	}
	return nil
}

// storageClassDB returns a storage class db instance suitable for storing into axdb
func (sc *StorageClass) storageClassDB() *storageClassDB {
	parameters := ""
	if sc.Parameters != nil {
		paramBytes, err := json.Marshal(sc.Parameters)
		if err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to marshal the storage class schema: %v", err))
			return nil
		}
		parameters = string(paramBytes)
	}
	spDB := &storageClassDB{
		ID:          sc.ID,
		Name:        strings.ToLower(sc.Name),
		Description: sc.Description,
		Ctime:       sc.Ctime * 1e6,
		Mtime:       sc.Mtime * 1e6,
		Parameters:  parameters,
	}
	return spDB
}

func (scdb *storageClassDB) StorageClass() *StorageClass {
	var parameters map[string]interface{}
	if scdb.Parameters != "" {
		if err := json.Unmarshal([]byte(scdb.Parameters), &parameters); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to unmarshal the parameters string in storage class:%v", err))
			return nil
		}
	}
	sc := &StorageClass{
		ID:          scdb.ID,
		Name:        strings.ToLower(scdb.Name),
		Description: scdb.Description,
		Ctime:       scdb.Ctime / 1e6,
		Mtime:       scdb.Mtime / 1e6,
		Parameters:  parameters,
	}
	return sc
}
