package core

import (
	"applatix.io/axdb"
	"fmt"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"strings"
	"time"
)

var profileChan = make(chan *Profile, 200)

type Profile map[string]interface{}

// get the parameter names, we don't care parameter values
func getParametersFromQuery(c *gin.Context) string {
	var names []string
	// get parameters from query string
	values := c.Request.URL.Query()
	for name, _ := range values {
		names = append(names, fmt.Sprintf("%s=?", name))
	}
	sort.Strings(names)
	return strings.Join(names, ";")
}

// returns 0 on error. 0 is not a valid parameter. 0 also indicates that parameter is not set
func queryParameterInt(c *gin.Context, name string) int64 {
	valueArray := c.Request.URL.Query()[name]
	if valueArray == nil {
		return 0
	}

	value, err := strconv.ParseInt(valueArray[0], 10, 64)
	if err != nil {
		errorLog.Printf("expecting int64 got %v", valueArray[0])
		c.JSON(axdb.RestStatusInvalid, axdb.AXMap{})
		return 0
	}
	return value
}

func getMetaDataFromContext(c *gin.Context) (Profile, *axdb.AXDBError) {
	profile := make(map[string]interface{})
	app := c.Param("dbapp")
	if app == "" {
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, "app is empty.")
	}

	table := c.Param("dbtable")
	if table == "" {
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, "table name is empty.")
	}

	var op string = ""
	var parameters string = ""
	restMethod := c.Request.Method
	if restMethod == "GET" {
		// for "select" query, we will also record the parameter names
		parameters = getParametersFromQuery(c)
		op = "SELECT"
	} else if restMethod == "POST" {
		if c.Param("dbapp") == "axdb" && c.Params.ByName("dbtable") == "create_table" {
			op = "DDL"
		} else {
			op = "INSERT"
		}
	} else if restMethod == "PUT" {
		if c.Param("dbapp") == "axdb" && c.Params.ByName("dbtable") == "create_table" {
			op = "DDL"
		} else {
			op = "UPDATE"
		}
	} else if restMethod == "DELETE" {
		// drop table
		if c.Request.Body == nil {
			op = "DDL"
		} else {
			op = "DELETE"
		}

	} else {
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, "Invalid Rest method.")
	}

	tableName := fmt.Sprintf("%s_%s", app, table)
	profile[AXDBPerfTableName] = tableName
	profile[AXDBPerfOperation] = op
	if len(parameters) == 0 {
		parameters = "none"
	}
	profile[AXDBPerfParameters] = parameters
	return profile, nil

}

func AddProfiler(basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// if profiling isn't turned on, return immediately
		if profileSwitchStatus == false {
			c.Next()
		} else {
			// get the AXDB request information from gin.Context
			profileData, err := getMetaDataFromContext(c)
			if err == nil {
				//get the start timestamp
				beginTicket := time.Now().UnixNano()
				c.Next()
				retCode := c.Writer.Status()
				// we only record the successful queries
				if retCode == axdb.RestStatusOK {
					endTicket := time.Now().UnixNano()
					profileData[AXDBPerfExecTime] = float64((endTicket - beginTicket) / 1e6)
					//apiData.ApiCallerLogin = session.Username
					SendToProfileChan(&profileData)
				}
			}
		}
	}
}

func getProfileArray(c *gin.Context, intervalStr string) ([]map[string]interface{}, *axdb.AXDBError) {
	//interval int64, minTime int64, maxTime int64
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		errorLog.Printf("expecting interval to be int64 got %s", intervalStr)
		return nil, axdb.NewAXDBError(axdb.RestStatusInvalid, nil, "invalid URL request")
	}

	minTime := queryParameterInt(c, axdb.AXDBQueryMinTime)
	maxTime := queryParameterInt(c, axdb.AXDBQueryMaxTime)
	if maxTime == 0 {
		maxTime = time.Now().Unix()
	}
	if minTime == 0 {
		// by default returns 50 data points
		minTime = maxTime - 50*interval
	}

	minTime = minTime / interval * interval
	maxTime = maxTime / interval * interval
	params := map[string]interface{}{
		axdb.AXDBQueryMinTime:       minTime * 1e6,
		axdb.AXDBQueryMaxTime:       maxTime * 1e6,
		axdb.AXDBIntervalColumnName: interval,
	}

	infoLog.Printf("Preparing to query profile raw data....")

	profileArray, axErr := profileTable.get(params)
	if axErr != nil {
		return nil, axErr
	}
	for _, profile := range profileArray {
		if profile[axdb.AXDBTimeColumnName] != nil {
			// convert the unit of ax_time to second, which is only for UI display
			profile[axdb.AXDBTimeColumnName] = profile[axdb.AXDBTimeColumnName].(int64) / 1e6
		}
	}
	return profileArray, nil
}

func ProfileHandler(c *gin.Context, intervalStr string) {
	dataArray, axErr := getProfileArray(c, intervalStr)
	if axErr != nil {
		c.JSON(axErr.RestStatus, axdb.AXMap{})
		return
	}

	var format string = "json"
	valueArray := c.Request.URL.Query()["format"]
	if valueArray == nil {
		format = "json"
	} else if valueArray[0] == "html" {
		format = "html"
	} else {
		format = "json"
	}

	if format == "json" {
		c.JSON(axdb.RestStatusOK, map[string]interface{}{"data": dataArray})
	} else if format == "html" {
		c.HTML(axdb.RestStatusOK, "profile.html", gin.H{"dataArray": dataArray})
	}
}

func SendToProfileChan(p *Profile) {
	if p == nil {
		return
	}

	select {
	case profileChan <- p:
	default:
		debugLog.Printf("Profile channel is full, data: %v\n", *p)
	}
}

func ProfilerWorker() {
	for profile := range profileChan {
		debugLog.Printf("Profile buffer channel status: %v/%v\n", len(profileChan), cap(profileChan))
		_, err := profileTable.save(*profile, true)
		if err != nil {
			infoLog.Printf("*** TEST: insert failure with error %v", err)
		}
	}
}
