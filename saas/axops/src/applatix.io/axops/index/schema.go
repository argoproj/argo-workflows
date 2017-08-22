package index

import "applatix.io/axdb"

const (
	SearchIndexTable = "search_index"
	SearchIndexType  = "type"
	SearchIndexKey   = "key"
	SearchIndexValue = "value"
	SearchIndexCtime = "ctime"
	SearchIndexMtime = "mtime"
)

var SearchIndexSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    SearchIndexTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		SearchIndexType:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		SearchIndexKey:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		SearchIndexValue: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		SearchIndexCtime: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		SearchIndexMtime: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(60 * axdb.OneDay),
	},
	UseSearch: true,
}
