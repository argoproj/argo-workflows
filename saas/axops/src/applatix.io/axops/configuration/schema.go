package configuration

import "applatix.io/axdb"

const (
	ConfigurationTableName   = "configuration"
	ConfigurationUserName    = "user"
	ConfigurationName        = "name"
	ConfigurationDescription = "description"
	ConfigurationValue       = "value"
	ConfigurationDateCreated = "ctime"
	ConfigurationLastUpdated = "mtime"
)

var ConfigurationSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ConfigurationTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ConfigurationUserName:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ConfigurationName:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		ConfigurationDescription: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ConfigurationValue:       axdb.Column{Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		ConfigurationDateCreated: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		ConfigurationLastUpdated: axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}
