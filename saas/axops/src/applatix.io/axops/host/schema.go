package host

import "applatix.io/axdb"

const (
	HostId        = "id"
	HostName      = "name"
	HostStatus    = "status"
	HostPrivateIP = "private_ip"
	HostPublicIP  = "public_ip"
	HostMem       = "mem"
	HostCPU       = "cpu"
	HostECU       = "ecu"
	HostDisk      = "disk"
	HostModel     = "model"
	HostNetwork   = "network"
)

var HostSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    axdb.AXDBTableHost,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		HostId:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		HostName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		HostStatus:    axdb.Column{Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		HostPrivateIP: axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		HostPublicIP:  axdb.Column{Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		HostMem:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostCPU:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostECU:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostDisk:      axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostModel:     axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		HostNetwork:   axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(1 * axdb.OneHour),
	},
}
