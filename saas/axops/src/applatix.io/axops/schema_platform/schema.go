package schema_platform

import "applatix.io/axdb"

const (
	ArtifactsTable    = "artifacts"
	ArtifactsID       = "artifact_id"
	SourceArtifactsID = "source_artifact_id"
	ServiceInstanceID = "service_instance_id"
	FullPath          = "full_path"
	Name              = "name"
	IsAlias           = "is_alias"
	Description       = "description"
	SrcPath           = "src_path"
	SrcName           = "src_name"
	Excludes          = "excludes"
	StorageMethod     = "storage_method"
	StoragePath       = "storage_path"
	InlineStorage     = "inline_storage"
	CompressionMode   = "compression_mode"
	SymlinkMode       = "symlink_mode"
	ArchiveMode       = "archive_mode"
	NumByte           = "num_byte"
	NumFile           = "num_file"
	NumDir            = "num_dir"
	NumSymlink        = "num_symlink"
	NumOther          = "num_other"
	NumSkipByte       = "num_skip_byte"
	NumSkip           = "num_skip"
	NumStoredByte     = "stored_byte"
	Meta              = "meta"
	Timestamp         = "timestamp"
	WorkflowID        = "workflow_id"
	PodName           = "pod_name"
	ContainerName     = "container_name"
	Checksum          = "checksum"
	Tags              = "tags"
	RetentionTags     = "retention_tags"
	ArtifactType      = "artifact_type"
	Deleted           = "deleted"
	DeletedDate       = "deleted_date"
	DeletedBy         = "deleted_by"
	ThirdPartyLinks   = "third_party"
	RelativePath      = "relative_path"
	StructurePath     = "structure_path"

	RetentionTable       = "artifact_retention"
	RetentionName        = "name"
	RetentionPolicy      = "policy"
	RetentionDescription = "description"
	RetentionNumberOfArt = "total_number"
	RetentionTotalSize   = "total_size"
	RetentionRealSize    = "total_real_size"

	ArtifactMetaTable     = "artifact_meta"
	ArtifactMetaAttribute = "attribute"
	ArtifactMetaValue     = "value"

	NodePortTable = "nodeport_table"
	ElbName       = "elb_name"
	ListenerPort  = "listener_port"
	NodePort      = "node_port"
	ElbAddr       = "elb_addr"
)

var ArtifactsSchema = axdb.Table{
	AppName: axdb.AXDBAppAXSYS,
	Name:    ArtifactsTable,
	Type:    axdb.TableTypeTimeSeries,
	Columns: map[string]axdb.Column{
		ArtifactsID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ServiceInstanceID: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		SourceArtifactsID: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		FullPath:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Name:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		IsAlias:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		Description:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SrcPath:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SrcName:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Excludes:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StorageMethod:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StoragePath:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		InlineStorage:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CompressionMode:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		SymlinkMode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ArchiveMode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		NumByte:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumFile:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumDir:            axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSymlink:        axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumOther:          axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSkipByte:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumSkip:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		NumStoredByte:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		Meta:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Timestamp:         axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		WorkflowID:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		PodName:           axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ContainerName:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Checksum:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Tags:              axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RetentionTags:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ArtifactType:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Deleted:           axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexStrong},
		DeletedDate:       axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		DeletedBy:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ThirdPartyLinks:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RelativePath:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		StructurePath:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},

	UseSearch: true,
}

var ArtifactRetentionSchema = axdb.Table{
	AppName: axdb.AXDBAppAXSYS,
	Name:    RetentionTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		RetentionName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		RetentionPolicy:      axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		RetentionDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		RetentionNumberOfArt: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		RetentionTotalSize:   axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		RetentionRealSize:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var ArtifactMetaSchema = axdb.Table{
	AppName: axdb.AXDBAppAXSYS,
	Name:    ArtifactMetaTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ArtifactMetaAttribute: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ArtifactMetaValue:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

var NodePortManagerSchema = axdb.Table{
	AppName: axdb.AXDBAppAXSYS,
	Name:    NodePortTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ElbName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ListenerPort: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexClustering},
		NodePort:     axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ElbAddr:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}
