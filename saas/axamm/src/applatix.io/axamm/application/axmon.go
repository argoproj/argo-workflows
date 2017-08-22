package application

import (
	"applatix.io/axamm/adc"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
)

func CreateApp(name, id string) (*axerror.AXError, int) {
	utils.DebugLog.Printf("[AXMON] Creating application: %s (ID: %s)", name, id)
	detail := map[string]string{
		"name": name,
	}

	if axErr, code := adc.Reserve(id, "application", GetAppMonitorCpuCores(), GetAppMonitorMemMiB(), adc.AdcDefaultTtl, detail); axErr != nil {
		return axErr, code
	}

	return utils.AxmonCl.PostWithTimeRetry("axmon/application", nil, map[string]string{"name": name}, nil, retryConfig)
}

func DeleteApp(name, id string) (*axerror.AXError, int) {
	utils.DebugLog.Printf("[AXMON] Deleting application: %s (ID: %s)", name, id)
	if axErr, code := utils.AxmonCl.DeleteWithTimeRetry("axmon/application/"+name, nil, nil, nil, retryConfig); axErr != nil {
		return axErr, code
	}

	return adc.Release(id)
}

type AppSummary struct {
	Result Summary `json:"result"`
}

type Summary struct {
	Monitor   bool `json:"monitor"`
	Namespace bool `json:"namespace"`
	Registry  bool `json:"registry"`
}

func GetSystemAppStatus(name, id string) (*AppSummary, *axerror.AXError) {
	s := AppSummary{}
	if axErr, _ := utils.AxmonCl.GetWithTimeRetry("axmon/application/"+name, nil, &s, retryConfig); axErr != nil {
		return nil, axErr
	}
	return &s, nil
}
