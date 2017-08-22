// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/host"
	"applatix.io/axops/service"
	"applatix.io/axops/usage"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

func GetHostList() gin.HandlerFunc {
	return func(c *gin.Context) {
		hosts, dbErr := GetAllHostsWithExtendedInfo()
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
		} else {
			resultMap := map[string]interface{}{RestData: hosts}
			c.JSON(axerror.REST_STATUS_OK, resultMap)
		}
	}
}

type ExtendedHostInfo struct {
	// Host basic information
	host.Host
	// Host usage information
	Services []service.ServiceSummary `json:"services,omitempty"`
	Usage    usage.HostUsage          `json:"usage,omitempty"`
}

func GetAllHostsWithExtendedInfo() ([]ExtendedHostInfo, *axerror.AXError) {
	hosts, dbErr := host.GetAllHosts()
	if dbErr != nil {
		return nil, dbErr
	}

	hostsExt := []ExtendedHostInfo{}

	// retrieve the running service list
	servsList, err := service.GetServicesSummaryFromTable(service.RunningServiceTable)
	// the service list on the host will be empty, we don't return error.
	if err != nil {
		utils.ErrorLog.Printf("Request to %s table returns empty list. err: %v", service.RunningServiceTable, err)
	}

	// for each host, find the running services on it and its usage
	for _, h := range hosts {
		extHostInfo := ExtendedHostInfo{Host: h}
		if servsList != nil {
			// for each host find the running services on it
			servs := []service.ServiceSummary{}
			for _, serv := range servsList {
				if serv.HostId == h.ID {
					servs = append(servs, serv)
				}
			}

			if len(servs) != 0 {
				extHostInfo.Services = servs
			}

		}

		usage, err1 := usage.GetHostUsageById(h.ID)
		if err1 == nil {
			if usage != nil {
				usage.CPUPercent = usage.CPURequest / h.CPU
				usage.MemPercent = usage.MemRequest / h.Mem
				extHostInfo.Usage = *usage
			}
		}
		hostsExt = append(hostsExt, extHostInfo)
	}
	return hostsExt, nil
}
