// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package usage

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

const (
	HostUsageTable = "host_usage"

	HostUsageHostId         = "host_id"
	HostUsageHostName       = "host_name"
	HostUsageCPU            = "cpu"
	HostUsageCPUUsed        = "cpu_used"
	HostUsageCPUTotal       = "cpu_total"
	HostUsageCPUPercent     = "cpu_percent"
	HostUsageMem            = "mem"
	HostUsageMemPercent     = "mem_percent"
	HostUsageCPURequest     = "cpu_request"
	HostUsageCPURequestUsed = "cpu_request_used"
	HostUsageMemRequest     = "mem_request"

//HostUsageNetwork    = "network"
//HostUsageDisk       = "disk"
//HostUsageIO         = "io"
)

var HostUsageSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    HostUsageTable,
	Type:    axdb.TableTypeTimeSeries,
	Columns: map[string]axdb.Column{
		HostUsageHostId:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		HostUsageHostName:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		HostUsageCPU:            axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageCPUUsed:        axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageCPUTotal:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageCPUPercent:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageMem:            axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageMemPercent:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageCPURequest:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageCPURequestUsed: axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		HostUsageMemRequest:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	},
	Stats: map[string]int{
		HostUsageCPUUsed:        axdb.ColumnStatSum,
		HostUsageMem:            axdb.ColumnStatSum,
		HostUsageCPURequest:     axdb.ColumnStatSum,
		HostUsageCPURequestUsed: axdb.ColumnStatSum,
		HostUsageMemRequest:     axdb.ColumnStatSum,
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(3 * axdb.OneYear),
	},
}

type HostUsage struct {
	HostId         string  `json:"host_id,omitempty"`
	HostName       string  `json:"host_name,omitempty"`
	CPU            float64 `json:"cpu,omitempty"`
	CPUUsed        float64 `json:"cpu_used,omitempty"`
	CPUTotal       float64 `json:"cpu_total,omitempty"`
	CPUPercent     float64 `json:"cpu_percent,omitempty"`
	Mem            float64 `json:"mem,omitempty"`
	MemPercent     float64 `json:"mem_percent,omitempty"`
	CPURequest     float64 `json:"cpu_request,omitempty"`
	CPURequestUsed float64 `json:"cpu_request_used,omitempty"`
	MemRequest     float64 `json:"mem_request,omitempty"`
}

const (
	ContainerUsageTable = "container_usage"

	ContainerUsageCostId        = "cost_id"
	ContainerUsageHostId        = "host_id"
	ContainerUsageServiceId     = "service_id"
	ContainerUsageContainerId   = "container_id"
	ContainerUsageContainerName = "container_name"
	ContainerUsageCPU           = "cpu"
	ContainerUsageCPUUsed       = "cpu_used"
	ContainerUsageCPUTotal      = "cpu_total"
	ContainerUsageCPUPercent    = "cpu_percent"
	ContainerUsageMem           = "mem"
	ContainerUsageMemPercent    = "mem_percent"

	ContainerUsageCPURequest     = "cpu_request"
	ContainerUsageCPURequestUsed = "cpu_request_used"
	ContainerUsageMemRequest     = "mem_request"
)

var ContainerUsageSchema = axdb.Table{
	AppName: axdb.AXDBAppAXOPS,
	Name:    ContainerUsageTable,
	Type:    axdb.TableTypeTimeSeries,
	Columns: map[string]axdb.Column{
		ContainerUsageCostId:         axdb.Column{Type: axdb.ColumnTypeOrderedMap, Index: axdb.ColumnIndexPartition},
		ContainerUsageHostId:         axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerUsageServiceId:      axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerUsageContainerId:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerUsageContainerName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		ContainerUsageCPU:            axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageCPUUsed:        axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageCPUTotal:       axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageCPUPercent:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageMem:            axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageMemPercent:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageCPURequest:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageCPURequestUsed: axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		ContainerUsageMemRequest:     axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	},
	Stats: map[string]int{
		ContainerUsageCPUUsed:        axdb.ColumnStatSum,
		ContainerUsageMem:            axdb.ColumnStatSum,
		ContainerUsageCPURequest:     axdb.ColumnStatSum,
		ContainerUsageCPURequestUsed: axdb.ColumnStatSum,
		ContainerUsageMemRequest:     axdb.ColumnStatSum,
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(3 * axdb.OneYear),
	},
}

type ContainerUsage struct {
	CostId         map[string]string `json:"cost_id,omitempty"`
	HostId         string            `json:"host_id,omitempty"`
	ServiceId      string            `json:"service_id,omitempty"`
	ContainerId    string            `json:"container_id,omitempty"`
	ContainerName  string            `json:"container_name,omitempty"`
	CPU            float64           `json:"cpu,omitempty"`
	CPUUsed        float64           `json:"cpu_used,omitempty"`
	CPUTotal       float64           `json:"cpu_total,omitempty"`
	CPUPercent     float64           `json:"cpu_percent,omitempty"`
	Mem            float64           `json:"mem,omitempty"`
	MemPercent     float64           `json:"mem_percent,omitempty"`
	CPURequest     float64           `json:"cpu_request,omitempty"`
	CPURequestUsed float64           `json:"cpu_request_used,omitempty"`
	MemRequest     float64           `json:"mem_request,omitempty"`
}

func GetHostUsageById(hostId string) (*HostUsage, *axerror.AXError) {
	params := map[string]interface{}{
		HostUsageHostId:          hostId,
		axdb.AXDBQueryMaxEntries: 1,
	}

	resultArray := []HostUsage{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHostUsage, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s, table failed, err: %v", axdb.AXDBTableHostUsage, axErr)
		return nil, axErr

	}
	if len(resultArray) >= 1 {
		return &resultArray[0], nil
	} else {
		return nil, nil
	}
}
