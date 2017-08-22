// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

// profile table schema
const ProfileTableName = "profile"
const AuditTrailTableName = "audit_trail"

// the table record
const (
	APIName          = "api_name"
	APIMethod        = "api_method"
	APIParameters    = "api_parameters"
	APICallerSession = "api_session_id"
	APICallerLogin   = "api_user_login"
	APIExecTime      = "api_exec_time"
	APIRetCode       = "api_return_code"
	APIRemoteAddr    = "api_remote_addr"
	APIRequestURI    = "api_request_uri"
)

var ProfileSchema = axdb.Table{AppName: axopsPerfApp, Name: ProfileTableName, Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
	APIName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
	APIMethod:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
	APIRetCode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
	APIParameters:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	APICallerSession: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	APIExecTime:      axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
	APIRemoteAddr:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	APIRequestURI:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
},

	Stats: map[string]int{
		APIExecTime: axdb.ColumnStatPercent,
	},
	UseSearch: false,
	Configs: map[string]interface{}{
		"default_time_to_live": int64(30 * axdb.OneDay),
	},
}

var AuditTrailSchema = axdb.Table{
	AppName: axopsPerfApp, Name: AuditTrailTableName, Type: axdb.TableTypeTimeSeries, Columns: map[string]axdb.Column{
		APICallerSession: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		APICallerLogin:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		APIName:          axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		APIMethod:        axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		APIRetCode:       axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		APIParameters:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		APIExecTime:      axdb.Column{Type: axdb.ColumnTypeDouble, Index: axdb.ColumnIndexNone},
		APIRemoteAddr:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		APIRequestURI:    axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},

	Stats: map[string]int{
		APIExecTime: axdb.ColumnStatPercent,
	},
	UseSearch: false,
	Configs: map[string]interface{}{
		"default_time_to_live": int64(30 * axdb.OneDay),
	},
}

type Profile struct {
	ApiName        string  `json:"api_name"`
	ApiMethod      string  `json:"api_method,omitempty"`
	ApiParameters  string  `json:"api_parameters,omitempty"`
	ApiCallSession string  `json:"api_session_id"`
	ApiCallerLogin string  `json:"api_user_login,omitempty"`
	ApiExecTime    float64 `json:"api_exec_time"`
	ApiRetCode     string  `json:"api_return_code"`
	APIRemoteAddr  string  `json:"api_remote_addr"`
	APIRequestURI  string  `json:"api_request_uri"`
}

func getApiName(c *gin.Context, basePath string) string {
	var pathStr string
	var resStr string = ""
	if strings.HasPrefix(c.Request.URL.Path, basePath) {
		pathStr = c.Request.URL.Path[len(basePath):]
	} else {
		pathStr = c.Request.URL.Path
	}
	if strings.HasPrefix(pathStr, "/") {
		pathStr = pathStr[1:]
	}
	segments := strings.Split(pathStr, "/")
	first := true
	for _, seg := range segments {
		//if seg is a parameter name
		if len(c.Param(seg)) > 0 {
			continue
		}

		// if seg is a parameter value
		isValue := false
		for _, param := range c.Params {
			if seg == param.Value {
				isValue = true
				break
			}
		}
		if isValue {
			continue
		}

		if first {
			resStr = resStr + seg
			first = false
		} else {
			resStr = resStr + "/" + seg
		}
	}

	return resStr
}

// get the parameters
func getParameters(c *gin.Context) string {
	var paramStr string = ""
	isFirst := true

	for _, v := range c.Params {
		if isFirst {
			paramStr = paramStr + v.Key + "=" + v.Value
			isFirst = false
		} else {
			paramStr = paramStr + "," + v.Key + "=" + v.Value
		}
	}

	// get parameters from query string
	values := c.Request.URL.Query()
	for name, v := range values {
		vStr := "[" + strings.Join(v, ",") + "]"
		if isFirst {
			paramStr = paramStr + name + "=" + vStr
			isFirst = false
		} else {
			paramStr = paramStr + "," + name + "=" + vStr
		}
	}
	return paramStr
}

func AddProfiler(basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// get the API request information from gin.Context

		// the params passed to axdb query
		var apiData Profile
		apiData.ApiName = getApiName(c, basePath)

		session := GetContextSession(c)
		apiData.ApiCallSession = session.ID

		// API method: GET/PUT/POST
		apiData.ApiMethod = c.Request.Method
		apiData.ApiParameters = getParameters(c)

		// API RemoteAddr and RequestURI
		apiData.APIRemoteAddr = c.Request.RemoteAddr
		apiData.APIRequestURI = c.Request.RequestURI

		//get the start timestamp
		beginTicket := time.Now().UnixNano()
		c.Next()

		endTicket := time.Now().UnixNano()
		apiData.ApiExecTime = float64((endTicket - beginTicket) / 1e6)
		apiData.ApiRetCode = strconv.Itoa(c.Writer.Status())

		apiData.ApiCallerLogin = session.Username

		SendToProfileChan(&apiData)
	}
}

func getProfileArray(c *gin.Context, intervalStr string, tableName string) ([]map[string]interface{}, *axerror.AXError) {
	//interval int64, minTime int64, maxTime int64
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		utils.ErrorLog.Printf("expecting interval to be int64 got %s", intervalStr)
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessage("invalid URL request")
	}

	minTime := queryParameterInt(c, QueryMinTime)
	maxTime := queryParameterInt(c, QueryMaxTime)
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

	// We don't break down by parameter of URL request for now.
	profileArray := []map[string]interface{}{}
	axErr := Dbcl.Get(axopsPerfApp, tableName, params, &profileArray)
	if axErr != nil {
		return nil, axErr
	}
	for _, profile := range profileArray {
		if profile[axdb.AXDBTimeColumnName] != nil {
			profile[axdb.AXDBTimeColumnName] = int64(profile[axdb.AXDBTimeColumnName].(float64)) / 1e9
		}
	}

	return profileArray, nil
}

func ProfileHandler(c *gin.Context, intervalStr string, format string) {
	var tableName string
	var sessionId = c.Request.URL.Query().Get(axdb.AXDBQuerySessionID)
	if sessionId == "" {
		tableName = ProfileTableName
	} else {
		tableName = AuditTrailTableName
	}

	dataArray, axErr := getProfileArray(c, intervalStr, tableName)
	if axErr != nil {
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}
	if sessionId != "" {
		result := make([]map[string]interface{}, len(dataArray))
		var count = 0
		for i, _ := range dataArray {
			if dataArray[i][APICallerSession] == sessionId {
				result[count] = dataArray[i]
				count++
			}
		}
		if format == "json" {
			c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: result[0:count]})
		} else if format == "html" {
			c.HTML(axdb.RestStatusOK, "audit_trial.html", gin.H{"dataArray": result[0:count]})
		}
	} else {
		if format == "json" {
			c.JSON(axdb.RestStatusOK, map[string]interface{}{RestData: dataArray})
		} else if format == "html" {
			c.HTML(axdb.RestStatusOK, "profile.html", gin.H{"dataArray": dataArray})
		}
	}
}
