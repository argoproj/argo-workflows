// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Policy API [/spendings]
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/host"
	"applatix.io/axops/sandbox"
	"applatix.io/axops/usage"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	WeekInSeconds      = (7 * 24 * 3600)
	WeekInMicroSeconds = (7 * 24 * 3600 * 1e6)

	usageCpuRequestUsedSum = usage.HostUsageCPURequestUsed + axdb.AXDBSumColumnSuffix
	usageMemRequestSum     = usage.HostUsageMemRequest + axdb.AXDBSumColumnSuffix
	usageMemRequestCount   = usage.HostUsageMemRequest + axdb.AXDBCountColumnSuffix
)

func getSpendingArray(interval int64, minTime int64, maxTime int64) ([]PerfData, *axerror.AXError) {
	//minTime = minTime / interval * interval
	//maxTime = maxTime / interval * interval

	params := map[string]interface{}{
		axdb.AXDBQueryMinTime:       minTime * 1e6,
		axdb.AXDBQueryMaxTime:       maxTime * 1e6,
		axdb.AXDBIntervalColumnName: interval,
	}

	// We don't break down by app for now.
	hostUsageArray := make([]map[string]interface{}, 100)
	axErr := Dbcl.Get(axdb.AXDBAppAXOPS, usage.HostUsageTable, params, &hostUsageArray)
	if axErr != nil {
		return nil, axErr
	}

	maxLen := (maxTime-minTime)/interval + 1
	dataArray := make([]PerfData, maxLen)
	if len(hostUsageArray) == 0 {
		realMaxTime := maxTime / interval * interval
		for i := range dataArray {
			dataArray[i].Time = realMaxTime - int64(i)*interval
		}
		return dataArray, nil
	}

	// Use average price for now. We are currently not tracking the host histories.
	modelPrice, axErr := host.GetAveragePrice(false)
	if modelPrice == nil {
		return nil, axErr
	}

	chargeSum := 0.0
	spentSum := 0.0
	i := 0

	sumUpCost := func() {
		if spentSum > chargeSum {
			InfoLog.Printf("wrong data, spent %v is bigger than total aws charge %v", spentSum, chargeSum)
			spentSum = chargeSum
		}
		dataArray[i].Data = chargeSum
		dataArray[i].Max = spentSum
		spentSum = 0
		chargeSum = 0
	}

	dataArray[0].Time = int64(hostUsageArray[0][axdb.AXDBTimeColumnName].(float64)/1e6) / interval * interval
	for _, hostUsage := range hostUsageArray {
		t := int64(hostUsage[axdb.AXDBTimeColumnName].(float64)) / 1e6 / interval * interval
		hostId := hostUsage[usageHostId]
		if hostId == nil || len(hostId.(string)) == 0 {
			continue
		}

		if dataArray[i].Time != int64(t) {
			sumUpCost()
			i++
			dataArray[i].Time = t
		}

		//cpuUsedSum := hostUsage[usageCpuRequestUsedSum].(float64)
		memUsedSum := hostUsage[usageMemRequestSum].(float64)
		count := hostUsage[usageMemRequestCount].(float64)

		//cpuCost := cpuUsedSum * modelPrice.CoreCost
		memCost := memUsedSum * modelPrice.MemCost * utils.SpendingInterval
		//if cpuCost > memCost {
		//	spentSum += cpuCost
		//} else {
		//	spentSum += memCost
		//}
		spentSum += memCost

		// The price data are collected at exactly SpendingInterval. This means we can use the count of SpendingInterval to estimate how long the
		// host has been up.
		chargeSum += modelPrice.Cost * count * utils.SpendingInterval
	}

	sumUpCost()

	return dataArray[0 : i+1], nil
}

func getSpendingsBreakDownArray(interval int64, minTime int64, maxTime int64, breakDownBy string, filterBy string, filterByValue string, filterMap map[string]map[string]bool) ([]PerfDataBreakDown, *axerror.AXError) {
	//minTime = minTime / interval * interval
	//maxTime = maxTime / interval * interval

	params := map[string]interface{}{
		axdb.AXDBQueryMinTime:       minTime * 1e6,
		axdb.AXDBQueryMaxTime:       maxTime * 1e6,
		axdb.AXDBIntervalColumnName: interval,
	}
	var containerUsageArray []map[string]interface{}
	axErr := Dbcl.Get(AxOpsApp, usage.ContainerUsageTable, params, &containerUsageArray)
	if axErr != nil {
		return nil, axErr
	}

	if len(containerUsageArray) == 0 {
		return []PerfDataBreakDown{}, nil
	}

	modelPrice, axErr := host.GetAveragePrice(false)
	if modelPrice == nil {
		return nil, axErr
	}

	maxLen := (maxTime - minTime) / interval
	intervalBreakdowns := make([]map[string]*PerfDataBreakDown, maxLen)
	for _, containerUsage := range containerUsageArray {
		_, hasId := containerUsage[COSTID]
		if !hasId {
			continue
		}

		costID := containerUsage[COSTID].(map[string]interface{})
		if len(costID) == 0 {
			continue
		}

		if costID[breakDownBy] == nil {
			continue
		}

		if filterMap != nil && len(filterMap) != 0 {
			match := true
			for k, v := range costID {
				if keyFilter, ok := filterMap[k]; ok {
					value := v.(string)
					if k == "user" && sandbox.IsSandboxEnabled() {
						value = sandbox.GetUserIdForEmail(v.(string))
					}

					if _, ok := keyFilter[value]; !ok {
						match = false
						break
					}
				}
			}

			if !match {
				continue
			}
		} else {
			// filter data by the passed in filter parameters
			if filterBy != "" && filterByValue != "" {
				if v, exist := costID[filterBy]; exist {
					value := v.(string)
					if filterBy == "user" && sandbox.IsSandboxEnabled() {
						value = sandbox.GetUserIdForEmail(value)
					}
					if value != filterByValue {
						continue
					}
				} else {
					continue
				}
			}
		}

		costItemName := costID[breakDownBy].(string)

		t := int64(containerUsage[axdb.AXDBTimeColumnName].(float64)) / 1e6 / interval * interval
		// round the UTC time t to the next larger UI time boundary
		intervalIndex := maxLen - (maxTime+1-t-interval)/interval
		intervalStartTime := minTime + intervalIndex*interval
		if intervalIndex < 0 || intervalIndex >= maxLen {
			continue
		}

		intervalPerfs := intervalBreakdowns[intervalIndex]
		if intervalPerfs == nil {
			intervalPerfs = make(map[string]*PerfDataBreakDown)
			intervalBreakdowns[intervalIndex] = intervalPerfs
		}

		perf := intervalPerfs[costItemName]
		if perf == nil {
			if breakDownBy == "user" && sandbox.IsSandboxEnabled() {
				perf = &PerfDataBreakDown{Time: intervalStartTime, Data: 0, Id: sandbox.GetUserIdForEmail(costItemName), Name: sandbox.ReplaceEmailInSandbox(costItemName)}
			} else {
				perf = &PerfDataBreakDown{Time: intervalStartTime, Data: 0, Id: costItemName, Name: costItemName}
			}

			if costID["user"] != nil {
				if costID["user"].(string) == "axsys" || costID["user"].(string) == "k8s" {
					perf.IsSystem = true
				}
			}

			intervalPerfs[costItemName] = perf
		}
		//cpuSum := containerUsage[usageCpuRequestUsedSum].(float64)
		memSum := containerUsage[usageMemRequestSum].(float64)
		//cpuSpent := cpuSum * modelPrice.CoreCost
		memSpent := memSum * modelPrice.MemCost * utils.SpendingInterval

		//if cpuSpent > memSpent {
		//	perf.Data += cpuSpent
		//} else {
		//	perf.Data += memSpent
		//}

		perf.Data += memSpent

	}
	perfData := []PerfDataBreakDown{}

	length := len(intervalBreakdowns)
	for i := range intervalBreakdowns {
		for _, perf := range intervalBreakdowns[length-i-1] {
			perfData = append(perfData, *perf)
		}
	}

	return perfData, nil
}

func SpendingPerfHandler(c *gin.Context, intervalStr string) {
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting interval to be int64 got %s", intervalStr)
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}

	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && GetSpendingETag() == etag && interval == IntervalHour {
		c.Status(http.StatusNotModified)
		return
	}

	minTime := queryParameterInt(c, QueryMinTime)
	maxTime := queryParameterInt(c, QueryMaxTime)
	if maxTime == 0 {
		maxTime = time.Now().Unix()
	}
	if minTime == 0 {
		// by default returns 100 data points
		minTime = maxTime - 99*interval
	}

	_ = "breakpoint"
	dataArray, axErr := getSpendingArray(interval, minTime, maxTime)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	c.Header("ETag", GetSpendingETag())
	c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: dataArray})
}

type SpendingBreakDownData struct {
	Data []PerfDataBreakDown `json:"data"`
}

// @Title GetSpendingBreakDown
// @Description Get spending break down by some costid key with optional filter
// @Accept  json
// @Param   interval  	 	path    string     true        "Interval in seconds."
// @Param   by  	 	query   string     true        "Group by attribute, eg. user, app, service. This indicates how the spending will be grouped."
// @Param   min_time	 	query   int        false       "Min time. Default will be 100 intervals ago."
// @Param   max_time	 	query   int        false       "Max time. Default will be the now."
// @Param   filterBy	 	query   string     false       "Filter by attribute, eg. user, app, service."
// @Param   filterByValue	query   string     false       "Filter by attribute's value. We support only one value for now."
// @Param   filter	        query   string     false       "Filter, if used, filterBy and filterByValue will be ignored, eg:filter=app:system;service:axdb[Encode needed]"
// @Success 200 {object} SpendingBreakDownData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /spendings
// @Router /spendings/breakdown/{interval} [GET]
func SpendingPerfBreakDownHandler(c *gin.Context, intervalStr string) {
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting interval to be int64 got %s", intervalStr)
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(`"interval" should be valid integer.`))
		return
	}

	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && GetSpendingETag() == etag && interval == IntervalHour {
		c.Status(http.StatusNotModified)
		return
	}

	minTime := queryParameterInt(c, QueryMinTime)
	maxTime := queryParameterInt(c, QueryMaxTime)
	if maxTime == 0 {
		maxTime = time.Now().Unix()
	}
	if minTime == 0 {
		// by default returns 100 data points
		minTime = maxTime - 99*interval
	}

	byParam := c.Request.URL.Query()["by"]
	if byParam == nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(`"by" is the required field, the option is user, app, service.`))
		return
	}

	var filterMap map[string]map[string]bool
	filter := c.Request.URL.Query().Get("filter")
	if filter != "" {
		DebugLog.Println("Spending filtering:", filter)
		DebugLog.Println("Spending filtering(full):", c.Request.URL.Query()["filter"])
		filterMap = make(map[string]map[string]bool)
		kvs := strings.Split(filter, ";")
		for _, kv := range kvs {
			if strings.Count(kv, ":") != 1 {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The query string is invalid for field %v : Expecting one ':' as seperator of key and value in %v.", "filter", kv))
				return
			}

			kvl := strings.Split(kv, ":")
			key := kvl[0]
			vals := strings.Split(kvl[1], ",")

			if valMap, ok := filterMap[key]; !ok {
				valMap = map[string]bool{}
				filterMap[key] = valMap
			}

			valMap := filterMap[key]
			for _, val := range vals {
				valMap[val] = true
			}

			filterMap[key] = valMap
		}
		DebugLog.Println("Spending filter map:", filterMap)
	}

	var dataArray []PerfDataBreakDown
	var axErr *axerror.AXError
	dataArray, axErr = getSpendingsBreakDownArray(interval, minTime, maxTime, byParam[0], queryParameter(c, QueryFilterBy), queryParameter(c, QueryFilterByValue), filterMap)

	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	c.Header("ETag", GetSpendingETag())
	c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: dataArray})
}

func SpendingDetailHandler(c *gin.Context, startStr string, endStr string) {

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting int64 got %s", startStr)
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting int64 got %s", endStr)
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}

	interval := (end - start)
	_ = "breakpoint"
	if interval < 0 {
		ErrorLog.Printf("start %d bigger than end %d", start, end)
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}

	if interval > IntervalDay*5 {
		interval = IntervalDay
	} else if interval > IntervalHour*5 {
		interval = IntervalHour

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && GetSpendingETag() == etag && interval == IntervalHour {
			c.Status(http.StatusNotModified)
			return
		}

	} else {
		interval = IntervalMinute
	}
	InfoLog.Printf("picked interval %d", interval)

	params := map[string]interface{}{
		axdb.AXDBQueryMinTime:       start * 1e6,
		axdb.AXDBQueryMaxTime:       end * 1e6,
		axdb.AXDBIntervalColumnName: interval,
	}
	containerUsageArray := make([]map[string]interface{}, 100)
	axErr := Dbcl.Get(AxOpsApp, usage.ContainerUsageTable, params, &containerUsageArray)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	dataArray := []Usage{}
	if len(containerUsageArray) == 0 {
		c.Header("ETag", GetSpendingETag())
		c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: dataArray})
		return
	}

	modelPrice, axErr := host.GetAveragePrice(false)
	if modelPrice == nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	usageMap := map[string]*Usage{}
	for _, containerUsage := range containerUsageArray {
		_, hasId := containerUsage[COSTID]
		if hasId {
			costId := containerUsage[COSTID].(map[string]interface{})
			if len(costId) != 0 {
				usage := usageMap[axdb.SerializeOrderedMap(costId)]
				if usage == nil {
					usage = &Usage{}
					usage.CostID = costId
					usageMap[axdb.SerializeOrderedMap(costId)] = usage
				}

				//cpuSum := containerUsage[usageCpuRequestUsedSum].(float64)
				memSum := containerUsage[usageMemRequestSum].(float64)
				//cpuSpent := cpuSum * modelPrice.CoreCost
				memSpent := memSum * modelPrice.MemCost * utils.SpendingInterval
				//debugLog.Printf("%v cpu spent %v mem spent %v", costId, cpuSpent, memSpent)

				//if cpuSpent > memSpent {
				//	usage.Spent += cpuSpent
				//} else {
				//	usage.Spent += memSpent
				//}

				usage.Spent += memSpent
			}

		}
	}

	i := 0
	usageArray := make([]*Usage, len(usageMap))
	for _, v := range usageMap {
		usageArray[i] = v
		i++
	}
	if sandbox.IsSandboxEnabled() {
		for _, u := range usageArray {
			if usr, ok := u.CostID["user"]; ok {
				// replace email with name and add user id
				u.CostID["user"] = sandbox.ReplaceEmailInSandbox(usr.(string))
				u.CostID["id"] = sandbox.GetUserIdForEmail(usr.(string))
			}
		}
	}

	c.Header("ETag", GetSpendingETag())
	c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: usageArray})
}

var eTag string = "spendings-interval-3600-" + time.Now().String()

func GetSpendingETag() string {
	return eTag
}

func UpdateSpendingETag() {
	eTag = "spendings-interval-3600-" + time.Now().String()
}
