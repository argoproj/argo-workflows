package deployment

import (
	"applatix.io/axerror"
	"applatix.io/axops/service"
)

type PerfDataBreakDown struct {
	Time     int64   `json:"time"`
	Data     float64 `json:"data"`
	Name     string  `json:"name"`
	Id       string  `json:"id"`
	IsSystem bool    `json:"is_system"`
}

type SpendingResult struct {
	Data []*PerfDataBreakDown `json:"data"`
}

func GetDeploymentSpending(d *Deployment) (*axerror.AXError, int) {

	if TEST_ENABLED {
		return nil, 200
	}

	//params := map[string]interface{}{
	//	"by":       "service",
	//	"filter":   fmt.Sprintf("app:%v;service:%v", d.ApplicationName, d.Name),
	//	"min_time": d.CreateTime / 1e6,
	//}
	//
	//interval := 60 * 60
	//if time.Now().Unix()-d.LaunchTime > 3*24*60*60 {
	//	interval = 24 * 60 * 60
	//}
	////} else if time.Now().Unix()-d.LaunchTime > 24*60*60 {
	////	interval = 60 * 60
	////} else if time.Now().Unix()-d.LaunchTime > 60*60 {
	////	interval = 10 * 60
	////}
	//
	//data := SpendingResult{}
	//axErr := utils.AxopsCl.Get(fmt.Sprintf("spendings/breakdown/%v", interval), params, &data)
	//if axErr != nil {
	//	common.ErrorLog.Printf("[AXOPS]Failed to get deployment %v/%v spending: %v.\n", d.ApplicationName, d.Name, axErr)
	//	return axErr, axerror.REST_INTERNAL_ERR
	//}
	//
	//var cost float64
	//for _, spending := range data.Data {
	//	cost += spending.Data
	//}

	d.Cost = service.GetSpendingCents(d.CPU, d.Mem, float64(d.RunTime))

	return nil, 200
}
