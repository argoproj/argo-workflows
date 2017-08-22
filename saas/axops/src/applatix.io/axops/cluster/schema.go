package cluster

import "applatix.io/axdb"

const (
	ClusterSettingTable = "cluster_settings"
	ClusterKey          = "key"
	ClusterValue        = "value"
	ClusterCtime        = "ctime"
	ClusterMtime        = "mtime"
)

var ClusterSettingSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ClusterSettingTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ClusterKey:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ClusterValue: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ClusterCtime: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ClusterMtime: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}
