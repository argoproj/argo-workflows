package adc

import (
	"strconv"
	"time"

	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/restcl"
)

var TEST_ENABLED bool = false

func EnableTest() {
	TEST_ENABLED = true
}

type Request struct {
	Category   string            `json:"category"`
	ResourceId string            `json:"resource_id"`
	TTL        string            `json:"ttl"`
	CpuCores   float64           `json:"cpu_cores"`
	MemMib     float64           `json:"mem_mib"`
	Detail     map[string]string `json:"detail,omitempty"`
}

// 2 hours default TTL for ADC
var AdcDefaultTtl int64 = 60 * 60 * 2
var MaxRetryDuration time.Duration = 10 * time.Minute

var retryConfig *restcl.RetryConfig = &restcl.RetryConfig{
	Timeout: MaxRetryDuration,
}

func Reserve(id, category string, cpuCores, memMib float64, ttl int64, detail map[string]string) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, axerror.REST_STATUS_OK
	}

	r := Request{
		Category:   category,
		ResourceId: id,
		CpuCores:   cpuCores,
		MemMib:     memMib,
		TTL:        strconv.Itoa(int(ttl)),
		Detail:     detail,
	}
	utils.DebugLog.Printf("[ADC] Reserving resources for %v: cpu_cores: %f, mem_mib: %f, ttl: %s\n", id, r.CpuCores, r.MemMib, r.TTL)
	axErr, code := utils.AdcCl.PutWithTimeRetry("adc/resource", nil, r, nil, retryConfig)
	if axErr != nil {
		utils.ErrorLog.Printf("[ADC] Resource reservation %v failed: %v.\n", r, axErr)
	} else {
		utils.DebugLog.Printf("[ADC] Resource reservation %v succeeded.\n", r)
	}
	return axErr, code
}

func Release(id string) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, axerror.REST_STATUS_OK
	}
	utils.DebugLog.Printf("[ADC] Releasing resources for %v\n", id)
	axErr, code := utils.AdcCl.DeleteWithTimeRetry("adc/resource/"+id, nil, nil, nil, retryConfig)
	if axErr != nil {
		utils.ErrorLog.Printf("[ADC] Resource release %v failed: %v.\n", id, axErr)
	} else {
		utils.DebugLog.Printf("[ADC] Resource release %v succeeded.\n", id)
	}
	return axErr, code
}
