package volume

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

type Volume struct {
	ID                string                 `json:"id,omitempty" description:"uuid"`
	Name              string                 `json:"name,omitempty" description:"name"`
	Ctime             int64                  `json:"ctime,omitempty" description:"creation time in seconds since epoch"`
	Mtime             int64                  `json:"mtime,omitempty" description:"modification time in seconds since epoch"`
	Atime             int64                  `json:"atime,omitempty" description:"access time in seconds since epoch"`
	Anonymous         *bool                  `json:"anonymous,omitempty"  description:"anonymous"`
	StorageProvider   string                 `json:"storage_provider,omitempty"  description:"storage provider name"`
	StorageProviderID string                 `json:"storage_provider_id,omitempty"  description:"storage provider id"`
	StorageClass      string                 `json:"storage_class,omitempty"  description:"storage class name"`
	StorageClassID    string                 `json:"storage_class_id,omitempty"  description:"storage class id"`
	Enabled           *bool                  `json:"enabled,omitempty"  description:"enabled"`
	AXRN              string                 `json:"axrn,omitempty"  description:"applatix resource name"`
	Owner             string                 `json:"owner,omitempty"  description:"owner username"`
	Creator           string                 `json:"creator,omitempty"  description:"creator username"`
	Status            string                 `json:"status,omitempty"  description:"volume status (init, creating, active, error, deleting)"`
	StatusDetail      map[string]interface{} `json:"status_detail,omitempty"  description:"status detail"`
	Concurrency       int                    `json:"concurrency,omitempty" description:"concurrency"`
	Referrers         []interface{}          `json:"referrers,omitempty"  description:"list of deployments referring to this volume"`
	ResourceID        string                 `json:"resource_id,omitempty"  description:"storage resource id"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"  description:"volume attributes"`
}

// volumeDB is an internal structure used for serializing/deserializing volumes to axdb
type volumeDB struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name,omitempty"`
	Ctime             int64  `json:"ctime,omitempty"`
	Mtime             int64  `json:"mtime,omitempty"`
	Atime             int64  `json:"atime,omitempty"`
	Anonymous         *bool  `json:"anonymous,omitempty"`
	StorageProvider   string `json:"storage_provider,omitempty" `
	StorageProviderID string `json:"storage_provider_id,omitempty"`
	StorageClass      string `json:"storage_class,omitempty"`
	StorageClassID    string `json:"storage_class_id,omitempty"`
	Enabled           *bool  `json:"enabled,omitempty"`
	AXRN              string `json:"axrn,omitempty"`
	Owner             string `json:"owner,omitempty"`
	Creator           string `json:"creator,omitempty"`
	Status            string `json:"status,omitempty"`
	StatusDetail      string `json:"status_detail,omitempty"`
	Concurrency       int    `json:"concurrency,omitempty"`
	Referrers         string `json:"referrers,omitempty"`
	ResourceID        string `json:"resource_id,omitempty"`
	Attributes        string `json:"attributes,omitempty"`
}

// Possible volume statuses. This should match fixturemanager's VolumeStatus
const (
	VolumeStatusInit     = "init"
	VolumeStatusCreating = "creating"
	VolumeStatusActive   = "active"
	VolumeStatusDeleting = "deleting"
)

func (vdb *volumeDB) Volume() *Volume {
	var statusDetail map[string]interface{}
	if vdb.StatusDetail != "" {
		if err := json.Unmarshal([]byte(vdb.StatusDetail), &statusDetail); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to unmarshal status_detail string for volume %s: %v", vdb.ID, err))
			return nil
		}
	}
	var referrers []interface{}
	if vdb.Referrers != "" {
		if err := json.Unmarshal([]byte(vdb.Referrers), &referrers); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to unmarshal referrers string for volume %s: %v", vdb.ID, err))
			return nil
		}
	}
	var attributes map[string]interface{}
	if vdb.Attributes != "" {
		if err := json.Unmarshal([]byte(vdb.Attributes), &attributes); err != nil {
			utils.ErrorLog.Printf(fmt.Sprintf("Failed to unmarshal attributes string for volume %s: %v", vdb.ID, err))
			return nil
		}
	}
	v := &Volume{
		ID:                vdb.ID,
		Name:              vdb.Name,
		Ctime:             vdb.Ctime / 1e6,
		Mtime:             vdb.Mtime / 1e6,
		Atime:             vdb.Atime / 1e6,
		Anonymous:         vdb.Anonymous,
		StorageProvider:   vdb.StorageProvider,
		StorageProviderID: vdb.StorageProviderID,
		StorageClass:      vdb.StorageClass,
		StorageClassID:    vdb.StorageClassID,
		Enabled:           vdb.Enabled,
		AXRN:              vdb.AXRN,
		Owner:             vdb.Owner,
		Creator:           vdb.Creator,
		Status:            vdb.Status,
		StatusDetail:      statusDetail,
		Concurrency:       vdb.Concurrency,
		Referrers:         referrers,
		ResourceID:        vdb.ResourceID,
		Attributes:        attributes,
	}
	return v
}

// GetVolumes returns a slice of volumes filtered by the given parameters
func GetVolumes(params map[string]interface{}) ([]Volume, *axerror.AXError) {
	volumeDBs := []volumeDB{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, VolumeTableName, params, &volumeDBs)
	if axErr != nil {
		return nil, axErr
	}
	volumes := make([]Volume, len(volumeDBs))
	for i := range volumeDBs {
		v := volumeDBs[i].Volume()
		volumes[i] = *v
	}
	return volumes, nil
}

// GetVolumeByID returns a single volume instance by its ID, or nil if it does not exist
func GetVolumeByID(id string) (*Volume, *axerror.AXError) {
	volumes, axErr := GetVolumes(map[string]interface{}{
		VolumeID: id,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(volumes) == 0 {
		return nil, nil
	}
	if len(volumes) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Found multiple volumes with ID: %s", id)
	}
	v := volumes[0]
	return &v, nil
}

// GetVolumeByAXRN returns a single volume instance by its AXRN, or nil if it does not exist
func GetVolumeByAXRN(axrn string) (*Volume, *axerror.AXError) {
	volumes, axErr := GetVolumes(map[string]interface{}{
		VolumeAXRN: axrn,
	})
	if axErr != nil {
		return nil, axErr
	}
	if len(volumes) == 0 {
		return nil, nil
	}
	if len(volumes) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Found multiple volumes with AXRN: %s", axrn)
	}
	v := volumes[0]
	return &v, nil
}
