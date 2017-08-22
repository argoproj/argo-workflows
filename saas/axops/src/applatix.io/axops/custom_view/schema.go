package custom_view

import "applatix.io/axdb"

const (
	CustomViewTable    = "custom_view"
	CustomViewID       = "id"
	CustomViewName     = "name"
	CustomViewType     = "type"
	CustomViewUserName = "username"
	CustomViewUserID   = "user_id"
	CustomViewInfo     = "info"
	CustomViewCtime    = "ctime"
	CustomViewMtime    = "mtime"
)

var CustomViewSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    CustomViewTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		CustomViewID:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		CustomViewName:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CustomViewType:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CustomViewUserName: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CustomViewUserID:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CustomViewInfo:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		CustomViewCtime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		CustomViewMtime:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
	UseSearch: true,
}
