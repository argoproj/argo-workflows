// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package volume

import "applatix.io/axdb"

// Storage Provider table constants
const (
	StorageProviderTableName = "storage_providers"
	StorageProviderID        = "id"
	StorageProviderName      = "name"
	StorageProviderCtime     = "ctime"
	StorageProviderMtime     = "mtime"
	StorageProviderSPSchema  = "sp_schema"
)

var StorageProviderSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    StorageProviderTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		StorageProviderID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		StorageProviderName:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		StorageProviderCtime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		StorageProviderMtime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		StorageProviderSPSchema: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

// Storage Class table constants
const (
	StorageClassTableName   = "storage_classes"
	StorageClassID          = "id"
	StorageClassName        = "name"
	StorageClassDescription = "description"
	StorageClassCtime       = "ctime"
	StorageClassMtime       = "mtime"
	StorageClassParameters  = "parameters"
)

var StorageClassSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    StorageClassTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		StorageClassID:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		StorageClassName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		StorageClassDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StorageClassCtime:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		StorageClassMtime:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		StorageClassParameters:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}

// Volumes table constants
const (
	VolumeTableName         = "volumes"
	VolumeID                = "id"
	VolumeName              = "name"
	VolumeCtime             = "ctime"
	VolumeMtime             = "mtime"
	VolumeAtime             = "atime"
	VolumeAnonymous         = "anonymous"
	VolumeStorageProvider   = "storage_provider"
	VolumeStorageProviderID = "storage_provider_id"
	VolumeStorageClass      = "storage_class"
	VolumeStorageClassID    = "storage_class_id"
	VolumeEnabled           = "enabled"
	VolumeAXRN              = "axrn"
	VolumeOwner             = "owner"
	VolumeCreator           = "creator"
	VolumeStatus            = "status"
	VolumeStatusDetail      = "status_detail"
	VolumeConcurrency       = "concurrency"
	VolumeReferrers         = "referrers"
	VolumeResourceID        = "resource_id"
	VolumeAttributes        = "attributes"
)

var VolumeSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    VolumeTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		VolumeID:                axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		VolumeName:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeCtime:             axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		VolumeMtime:             axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		VolumeAtime:             axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		VolumeAnonymous:         axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		VolumeStorageProvider:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeStorageProviderID: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeStorageClass:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeStorageClassID:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeEnabled:           axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		VolumeAXRN:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		VolumeOwner:             axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeCreator:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeStatus:            axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeStatusDetail:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeConcurrency:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		VolumeReferrers:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeResourceID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		VolumeAttributes:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}
