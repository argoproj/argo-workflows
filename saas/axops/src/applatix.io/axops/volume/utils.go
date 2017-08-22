package volume

import (
	"time"

	"applatix.io/axerror"
	"applatix.io/common"
)

const (
	EBSStorageProviderName = "ebs"
	SSDStorageClassName    = "ssd"
	//SSDIOStorageClassName = "ssd-io"
)

// PopulateStorageProviderClasses populates Applatix's built-in storage providers/classes
func PopulateStorageProviderClasses() *axerror.AXError {
	now := time.Now().UTC().Unix()
	ebsStorageProvider := StorageProvider{
		ID:    common.GenerateUUIDv5(EBSStorageProviderName),
		Name:  EBSStorageProviderName,
		Ctime: now,
		Mtime: now,
		SPSchema: map[string]interface{}{
			"volume_type": []string{"gp2", "io1", "st1", "sc1"},
			"filesystem":  []string{"ext4"},
		},
	}
	axErr := ebsStorageProvider.upsert()
	if axErr != nil {
		return axErr
	}

	attachedGPStorageClass := StorageClass{
		ID:          common.GenerateUUIDv5(SSDStorageClassName),
		Name:        SSDStorageClassName,
		Description: "General purpose SSD volume that balances price and performance for a wide variety of transactional workloads",
		Ctime:       now,
		Mtime:       now,
		Parameters: map[string]interface{}{
			"aws": map[string]interface{}{
				"storage_provider_id":   ebsStorageProvider.ID,
				"storage_provider_name": ebsStorageProvider.Name,
				"volume_type":           "gp2",
				"filesystem":            "ext4",
			},
		},
	}
	axErr = attachedGPStorageClass.upsert()
	if axErr != nil {
		return axErr
	}

	// attachedIOStorageClass := StorageClass{
	// 	ID:          common.GenerateUUIDv5(SSDStorageClassName),
	// 	Name:        SSDStorageClassName,
	// 	Description: "Highest-performance SSD volume designed for mission-critical applications",
	// 	Ctime:       now,
	// 	Mtime:       now,
	// 	Parameters: map[string]interface{}{
	// 		"aws": map[string]interface{}{
	// 			"storage_provider_id":   ebsStorageProvider.ID,
	// 			"storage_provider_name": ebsStorageProvider.Name,
	// 			"volume_type":           "io1",
	// 			"filesystem":            "ext4",
	// 		},
	// 	},
	// }
	// axErr = attachedIOStorageClass.upsert()
	// if axErr != nil {
	// 	return axErr
	// }

	return nil
}
