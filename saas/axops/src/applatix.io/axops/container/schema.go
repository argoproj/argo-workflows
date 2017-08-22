// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package container

import "applatix.io/axdb"

const (
	ContainerId        = "id"
	ContainerName      = "name"
	ContainerServiceId = "service_id"
	ContainerHostId    = "host_id"
	ContainerHostName  = "host_name"
	ContainerCostId    = "cost_id"
	ContainerMem       = "mem"
	ContainerCPU       = "cpu"
	ContainerLogDone   = "url_done"
	ContainerLogLive   = "url_run"
	ContainerEndpoint  = "endpoint"
)

var ContainerSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    axdb.AXDBTableContainer,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		ContainerId:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		ContainerName:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerServiceId: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		ContainerHostId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		ContainerHostName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerCostId:    axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexNone},
		ContainerMem:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerCPU:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(30 * axdb.OneMinute),
	},
}
