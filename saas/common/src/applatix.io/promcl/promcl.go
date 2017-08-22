// Copyright 2015-2017 Applatix, Inc. All rights reserved.

package promcl

import (
	"applatix.io/axerror"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const READ_OPS_METRIC = "node_disk_reads_completed"
const WRITE_OPS_METRIC = "node_disk_writes_completed"
const READ_SECTOR_METRIC = "node_disk_sectors_read"
const WRITE_SECTOR_METRIC = "node_disk_sectors_written"

type FilesystemFreeResponseMetric struct {
	Device   string `json:"device"`
	Instance string `json:"Instance"`
}

type FilesystemFreeResult struct {
	Value  [2]interface{}               `json:"value"`
	Metric FilesystemFreeResponseMetric `json:"metric"`
}

type FilesystemFreeData struct {
	ResultType string                 `json:"resultType"`
	Result     []FilesystemFreeResult `json:"result"`
}

type FilesystemFreeResponse struct {
	Status string             `json:"status"`
	Data   FilesystemFreeData `json:"data"`
}

type VolStatResult struct {
	Values [][2]interface{} `json:"values"`
}

type VolStatData struct {
	Result []VolStatResult `json:"result"`
}

type VolStatResponse struct {
	Data VolStatData `json:"data"`
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) *axerror.AXError {
	r, err := myClient.Get(url)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Error: %v", err)
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(target)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Error: %v", err)
	}

	return nil
}

func getPromURL() string {
	return "http://prometheus.axsys:9090/api/v1"
}

func GetDeviceAndInstance(volId string) (string, string, *axerror.AXError) {
	// Call prometheus
	// Get device name and instance name from filesystem free metric using volume id (getting metadata)
	metadata := FilesystemFreeResponse{}
	axErr := getJson(fmt.Sprintf("%s/query?query=node_filesystem_free{mountpoint=~'.*%s.*'}", getPromURL(), volId), &metadata)

	if axErr != nil {
		return "", "", axErr
	}

	if len(metadata.Data.Result) < 1 {
		return "", "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Error: No device name for this volId %v", volId)
	}

	device := strings.Split(metadata.Data.Result[0].Metric.Device, "/dev/")

	if len(device) != 2 {
		return "", "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Error: Failed to get prometheus device name.")
	}
	device_name := device[1]
	instance_name := metadata.Data.Result[0].Metric.Instance

	if device_name == "" || instance_name == "" {
		return "", "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Error: Device name %s or Instance name %s cannot be empty", device_name, instance_name)
	}
	return device_name, instance_name, nil
}

func GetVolumeMetric(metricName string, deviceName string, instanceName string, minTime int64, maxTime int64, interval int64) (*VolStatResult, *axerror.AXError) {
	stats_response := VolStatResponse{}
	url := fmt.Sprintf("%s/query_range?query=%s{device='%s',instance='%s'}&start=%d&end=%d&step=%ds", getPromURL(), metricName, deviceName, instanceName, minTime, maxTime, interval)
	axErr := getJson(url, &stats_response)

	if axErr != nil {
		return nil, axErr
	}
	return &(stats_response.Data.Result[0]), nil
}

func IsValidType(typeStr string) bool {
	switch typeStr {
	case
		"readops",
		"writeops",
		"readtot",
		"writetot",
		"readsizeavg",
		"writesizeavg",
		"readsizetot",
		"writesizetot":
		return true
	}
	return false
}

func DeleteVolumeMetric(workflowId string) (*http.Response, *axerror.AXError) {
	url := fmt.Sprintf("%s/series?match[]={pod_name=~'.*%s.*'}", getPromURL(), workflowId)
	req, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to create a delete request root ID: %v", workflowId)
	}

	r, err := myClient.Do(req)

	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to delete workflow metrics: %v, error: %v", workflowId, err)
	}

	return r, nil
}
