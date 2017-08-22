package core

import (
	"bytes"
	"encoding/json"

	"applatix.io/axdb"
	"applatix.io/axerror"
	//"fmt"
	"fmt"
	"strings"
	"time"

	"applatix.io/axdb/axdbcl"
	"github.com/gin-gonic/gin"
)

func getTableFromContext(c *gin.Context) (table TableInterface) {
	app := theDB.getApp(c.Param("dbapp"))
	return app.getTable(c.Param("dbtable"))
}

func getClusterStatus(c *gin.Context) string {
	if ok := theDB.clusterIsReady() && theDB.backendIsRunning(); ok {
		infoLog.Printf("*** CLUSTER STATUS: OK")
		return "OK"
	} else {
		infoLog.Printf("*** CLUSTER STATUS: NOK")
		return "NOK"
	}
}

func switchProfileStatus(c *gin.Context) string {
	status := strings.ToLower(c.Request.URL.Query().Get("switch"))
	if status != "on" && status != "off" {
		return "Unknown switch status"
	}

	theDB.SwitchProfileStatus(status, true)
	return status
}

func getBody(c *gin.Context) (map[string]interface{}, error) {
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(c.Request.Body)
	str := buffer.String()
	// infoLog.Printf("%s %s/%s/%s %s", c.Request.Method, c.Param("dbversion"), c.Param("dbapp"), c.Param("dbtable"), str)

	if strings.Contains(str, "'") {
		str = strings.Replace(str, "'", "''", -1)
		buffer = bytes.NewBufferString(str)
	}

	data := make(map[string]interface{})
	decoder := json.NewDecoder(buffer)
	decoder.UseNumber()
	err := decoder.Decode(&data)
	if err != nil {
		errorLog.Printf("Can't decode data %s into json, error: %s", buffer.String(), err)
		return nil, err
	}

	return data, nil
}

func AddResponseHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		ResponseEchoRequestUUID(c)

		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Cache-Control", "no-cache")

		c.Next()
	}
}

func ResponseEchoRequestUUID(c *gin.Context) {
	var requestUUID = c.Request.Header.Get("X-Request-UUID")
	if len(requestUUID) > 0 {
		c.Writer.Header().Set("X-Request-UUID-Echo", requestUUID)
	}
}

// Init http router and start REST server
func StartRouter(internal bool) {
	router := gin.New()
	router.Use(gin.Recovery())
	nullResult := make(map[string]interface{})

	//router.GET("ping", func(c *gin.Context) {
	//	c.JSON(axdb.RestStatusOK, "pong")
	//})

	//router.GET("/v1/ping", func(c *gin.Context) {
	//	c.JSON(axdb.RestStatusOK, "pong")
	//})

	router.Use(AddResponseHeaders())
	v1 := router.Group(axdb.AXDBVersion)
	{

		v1.Use(AddProfiler(v1.BasePath()))
		v1.GET("/:dbapp/:dbtable", func(c *gin.Context) {
			if c.Param("dbapp") == "axdb" {
				// move this out to a separate function once we implement more axdb functions
				if c.Param("dbtable") == "version" {
					c.JSON(axdb.RestStatusOK, []map[string]interface{}{{"version": axdb.AXDBVersion}})

				} else if c.Param("dbtable") == "status" {
					if clusterStatus := getClusterStatus(c); clusterStatus == "OK" {
						c.JSON(axdb.RestStatusOK, []map[string]interface{}{{"status": "OK"}})
					} else {
						c.JSON(axdb.RestStatusInvalid, []map[string]interface{}{{"status": "NOK"}})
					}

				} else if c.Param("dbtable") == "profile" {
					status := switchProfileStatus(c)
					c.JSON(axdb.RestStatusOK, []map[string]interface{}{{"switch": status}})
				} else {
					c.JSON(axdb.RestStatusNotFound, axerror.ERR_AXDB_TABLE_NOT_FOUND)
				}
				return
			} else if c.Param("dbapp") == "histogram" {
				infoLog.Printf("In histogram request")
				// for access API histogram
				//ProfileHandler(c, c.Param("dbtable"), "json")
				ProfileHandler(c, c.Param("dbtable"))
				return
			} //else if c.Param("dbapp") == "profile_html" {
			//	ProfileHandler(c, c.Param("interval"), "html")
			//}

			params := make(map[string]interface{})
			for k, v := range c.Request.URL.Query() {
				// debugLog.Printf("key %v value %v", k, v)
				if k == axdb.AXDBSelectColumns {
					for _, vstr := range v {
						vstr = axdb.EscapedString(vstr)
					}
					params[k] = v
				} else {
					v[0] = axdb.EscapedString(v[0])
					params[k] = v[0]
				}
			}

			table := getTableFromContext(c)
			if table == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_AXDB_TABLE_NOT_FOUND)
				return
			}

			resultArray, err := table.get(params)
			if err == nil {
				c.JSON(axdb.RestStatusOK, resultArray)
				debugLog.Printf("returning %d entries", len(resultArray))
			} else {
				c.JSON(err.RestStatus, err.ToAXError())
			}
		})

		v1.POST("/:dbapp/:dbtable", func(c *gin.Context) {
			data, err := getBody(c)
			if err != nil {
				c.JSON(axdb.RestStatusInvalid, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("Post request has no body"))
				return
			}

			if c.Param("dbapp") == "axdb" {
				if c.Params.ByName("dbtable") == "create_table" {
					jsonData, err := json.Marshal(data)
					if err != nil {
						// This is not expected to happen, we just unmarshaled the data
						panic("Unexpected json marshal error")
					}
					if !isLeaderNode() {
						payload, err := getAppTableDefFromJsonByte(jsonData)
						if err == nil {
							loadLeaderNodeIp()
							leaderAddr := fmt.Sprintf("http://%s:%s/v1", leaderIp, "8080")
							client := axdbcl.NewAXDBClientWithTimeout(leaderAddr, time.Second*60)
							_, err := client.Post("axdb", "create_table", payload)
							if err == nil {
								c.JSON(axdb.RestStatusOK, nullResult)
							} else {
								infoLog.Printf("forward failed, ERR: %v", err)
								c.JSON(axdb.RestStatusInternalError, err)
							}
						} else {
							c.JSON(axdb.RestStatusInvalid, err.ToAXError())
						}
						return
					}

					axErr := addAppTableFromJsonByte(jsonData, true, true)
					if axErr == nil {
						c.JSON(axdb.RestStatusOK, nullResult)
					} else {
						c.JSON(axErr.RestStatus, axErr.ToAXError())
					}
					return
				}
			}
			table := getTableFromContext(c)
			if table == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_AXDB_TABLE_NOT_FOUND)
				return
			}

			resMap, axErr := table.save(data, true)
			if resMap == nil {
				resMap = nullResult
			}
			if axErr == nil {
				c.JSON(axdb.RestStatusOK, resMap)
			} else {
				c.JSON(axErr.RestStatus, axErr.ToAXError())
			}
		})

		v1.PUT("/:dbapp/:dbtable", func(c *gin.Context) {
			data, err := getBody(c)
			if err != nil {
				c.JSON(axdb.RestStatusInvalid, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("Put request has no body"))
				return
			}

			if c.Param("dbapp") == "axdb" {
				if c.Params.ByName("dbtable") == "update_table" {
					jsonData, err := json.Marshal(data)
					if err != nil {
						// This is not expected to happen, we just marshal the data
						panic("Unexpected json marshal error")
					}

					if !isLeaderNode() {
						payload, err := getAppTableDefFromJsonByte(jsonData)
						if err == nil {
							infoLog.Printf("payload: %v", payload)
							//load axdb-0 ip address before each forwarding. It's not performance optimal, but could tolerate axdb-0 pod restart
							loadLeaderNodeIp()
							leaderAddr := fmt.Sprintf("http://%s:%s/v1", leaderIp, "8080")
							client := axdbcl.NewAXDBClientWithTimeout(leaderAddr, time.Second*60)
							_, err := client.Put("axdb", "update_table", payload)
							if err == nil {
								c.JSON(axdb.RestStatusOK, nullResult)
							} else {
								infoLog.Printf("forward failed, ERR: %v", err)
								c.JSON(axdb.RestStatusInternalError, err)
							}
						} else {
							c.JSON(axdb.RestStatusInvalid, err.ToAXError())
						}
						return
					}
					axErr := updateAppTableFromJsonByte(jsonData)
					if axErr == nil {
						c.JSON(axdb.RestStatusOK, nullResult)
					} else {
						c.JSON(axErr.RestStatus, axErr.ToAXError())
					}
					return
				}
			}

			table := getTableFromContext(c)
			if table == nil {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_AXDB_TABLE_NOT_FOUND)
				return
			}

			resMap, axErr := table.save(data, false)
			if resMap == nil {
				resMap = nullResult
			}
			if axErr == nil {
				c.JSON(axdb.RestStatusOK, resMap)
			} else {
				c.JSON(axErr.RestStatus, axErr.ToAXError())
			}
		})

		v1.DELETE("/:dbapp/:dbtable", func(c *gin.Context) {
			if c.Param("dbapp") == "axdb" {
				c.JSON(axdb.RestStatusInvalid, axerror.ERR_AXDB_INVALID_PARAM)
				return
			}

			buffer := new(bytes.Buffer)
			buffer.ReadFrom(c.Request.Body)
			infoLog.Printf("DELETE %s/%s/%s %s", axdb.AXDBVersion, c.Param("dbapp"), c.Param("dbtable"), buffer.String())

			var resMap map[string]interface{} = nil
			var err *axdb.AXDBError
			if buffer.Len() == 0 {
				// No body, drop the table.
				resMap, err = theDB.getApp(c.Param("dbapp")).deleteTable(c.Param("dbtable"))
			} else {
				var data []map[string]interface{}
				decoder := json.NewDecoder(buffer)
				decoder.UseNumber()
				decoder.Decode(&data)

				if len(data) == 0 {
					errStr := "Expecting http body to be an array of keys to delete, but didn't find an array"
					infoLog.Println(errStr)
					err = axdb.NewAXDBError(axdb.RestStatusInvalid, nil, errStr)
				} else {
					table := getTableFromContext(c)
					if table == nil {
						err = axdb.NewAXDBError(axdb.RestStatusNotFound, nil, "table not found")
					} else {
						resMap, err = table.delete(data)
					}
				}
			}

			if resMap == nil {
				resMap = nullResult
			}
			if err == nil {
				c.JSON(axdb.RestStatusOK, resMap)
			} else {
				c.JSON(err.RestStatus, err.ToAXError())
			}
		})
	}

	if !internal {
		router.LoadHTMLGlob("../ax/axdb/html/*")
	}

	// Listen and Server in 0.0.0.0:8080, use for debugging
	router.Run(":8080")

	// switch to tls later
	// router.RunTLS(":8081", cert, key)
}
