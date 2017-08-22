// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package container

type Container struct {
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name,omitempty"`
	ServiceId string            `json:"service_id,omitempty"`
	HostId    string            `json:"host_id,omitempty"`
	HostName  string            `json:"host_name,omitempty"`
	CostId    map[string]string `json:"cost_id,omitempty"`
	Mem       float64           `json:"mem,omitempty"`
	CPU       float64           `json:"cpu,omitempty"`
}

// Given the event data, return the container object that's suitable to be put into the container table.
func EventDataToContainer(data map[string]interface{}) map[string]interface{} {
	cont := make(map[string]interface{})
	for k, v := range data {
		if _, ok := ContainerSchema.Columns[k]; ok {
			cont[k] = v
		}
	}
	return cont
}
