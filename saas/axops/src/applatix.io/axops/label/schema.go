package label

import "applatix.io/axdb"

const (
	LabelTableName = "label"
	LabelID        = "id"
	LabelType      = "type"
	LabelKey       = "key"
	LabelValue     = "value"
	LabelReserved  = "reserved"
	LabelCTime     = "ctime"
)

var LabelSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    LabelTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		LabelID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		LabelType:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		LabelKey:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		LabelValue:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		LabelReserved: axdb.Column{Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		LabelCTime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}
