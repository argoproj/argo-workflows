// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Storage API [/storage]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/volume"
	"applatix.io/promcl"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

type StorageClassesData struct {
	Data []volume.StorageClass `json:"data"`
}

type VolumesData struct {
	Data []volume.Volume `json:"data"`
}

// @Title GetStorageClasses
// @Description List storage classes
// @Accept  json
// @Success 200 {object} StorageClassesData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/classes [GET]
func ListStorageClasses() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: ask hong how to use this
		params, axErr := GetContextParams(c, []string{"name", "description"}, []string{}, []string{}, []string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		storageClasses, axErr := volume.GetStorageClasses(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		resultMap := StorageClassesData{
			Data: storageClasses,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title CreateVolume
// @Description Create a new volume
// @Accept  json
// @Param   volume   	 body    volume.Volume     true        "Volume."
// @Success 200 {object} volume.Volume
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes [POST]
func CreateVolume() gin.HandlerFunc {
	// Need to supply user context in reverse proxy, so that creator/owner is stored along with the volume.
	return FixtureManagerProxy(true)
}

// @Title GetVolumes
// @Description List storage volumes
// @Accept  json
// @Param   anonymous  	 query   bool     false       "Anonymous, volume which are anonymous or not"
// @Param   deployment_id  	 query   string     false       "Deployment ID, volumes which are in use by a deployment"
// @Success 200 {object} VolumesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes [GET]
func ListVolumes() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title GetVolumeByID
// @Description Get a volume by ID
// @Accept  json
// @Param   id     	 path    string     	      true        "Volume ID."
// @Success 200 {object} volume.Volume
// @Failure 404 {object} axerror.AXError "Volume does not exist"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes/{id} [GET]
func GetVolume() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title GetVolumeStats
// @Description Get stats for a volume
// @Accept  json
// @Param   id     	 path    string     true        "ID of the volume"
// @Param   interval  	 	query   string     true        "Interval in seconds."
// @Param   type                query   string     true        "Type of the volume stats. Valid values: [readops, writeops, readtot, writetot, readsizetot, writesizetot, readsizeavg, writesizeavg]"
// @Param   min_time	 	query   int        false       "Min time. Default will be 100 intervals ago."
// @Param   max_time	 	query   int        false       "Max time. Default will be the now."
// @Success 200 {array} promcl.VolStatResult
// @Failure 404 {object} axerror.AXError "Volume does not exist"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes/{id}/stats [GET]
func GetVolumeStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		volumeObj, axErr := volume.GetVolumeByID(idStr)
		if axErr != nil {
			ErrorLog.Printf("Failed to get resource ID from Volume ID %s, detail: %v", idStr, *axErr)
			c.JSON(axerror.REST_BAD_REQ, axErr)
		}

		volId := volumeObj.ResourceID
		typeStr := queryParameter(c, "type")
		intervalStr := queryParameter(c, "interval")
		if !promcl.IsValidType(typeStr) {
			ErrorLog.Printf("No metric type matching %s", typeStr)
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(`No metric type matching.`))
			return
		}

		interval, err := strconv.ParseInt(intervalStr, 10, 64)
		if err != nil {
			ErrorLog.Printf("expecting interval to be int64 got %s", intervalStr)
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(`"interval" should be valid integer.`))
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

		// Get device name and instance name
		device_name, instance_name, axErr := promcl.GetDeviceAndInstance(volId)
		if axErr != nil {
			ErrorLog.Printf("Failed to get prometheus device name from Volume ID %s, detail: %v", volId, *axErr)
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		}

		InfoLog.Printf("Volstats information: device name: %s, instance_name: %s", device_name, instance_name)

		var metric_type string
		var metric_type2 string
		if typeStr == "readops" || typeStr == "writeops" {
			if typeStr == "readops" {
				metric_type = promcl.READ_OPS_METRIC
			} else {
				metric_type = promcl.WRITE_OPS_METRIC
			}

			vol_result, axErr := promcl.GetVolumeMetric(metric_type, device_name, instance_name, minTime, maxTime, interval)
			if axErr != nil {
				ErrorLog.Printf("Failed to get prometheus %s for %s, device_name %s, instance_name %s, detail: %v", volId, metric_type, device_name, instance_name, *axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}

			// Data points from Prometheus
			result := (*vol_result).Values
			InfoLog.Print("Volstats information: ", result)

			var ret_result [][2]float64
			for i := 0; i < len(result)-1; i++ {
				first_number, err1 := strconv.ParseFloat(result[i+1][1].(string), 64)
				second_number, err2 := strconv.ParseFloat(result[i][1].(string), 64)
				if err1 != nil || err2 != nil {
					ErrorLog.Printf("expecting to be float64 in result, %v, %v", err1, err2)
					c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessagef(`expecting to be float64 in result Error: %v, %v`, err1, err2))
					return
				}
				// Finding operations per second (ops_at_t2 - ops_at_t1)/(t2 - t1)
				ret_result = append(ret_result, [2]float64{result[i][0].(float64), (first_number - second_number) / float64(interval)})
			}

			c.JSON(axerror.REST_STATUS_OK, &ret_result)
			return
		} else if typeStr == "readtot" || typeStr == "writetot" || typeStr == "readsizetot" || typeStr == "writesizetot" {
			if typeStr == "readtot" {
				metric_type = promcl.READ_OPS_METRIC
			} else if typeStr == "writetot" {
				metric_type = promcl.WRITE_OPS_METRIC
			} else if typeStr == "readsizetot" {
				metric_type = promcl.READ_SECTOR_METRIC
			} else {
				metric_type = promcl.WRITE_SECTOR_METRIC
			}

			vol_result, axErr := promcl.GetVolumeMetric(metric_type, device_name, instance_name, minTime, maxTime, interval)
			if axErr != nil {
				ErrorLog.Printf("Failed to get prometheus %s for %s, device_name %s, instance_name %s, detail: %v", volId, metric_type, device_name, instance_name, *axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}

			// Data points from Prometheus
			result := (*vol_result).Values
			InfoLog.Print("Volstats information: ", result)

			var ret_result [][2]float64
			for i := 0; i < len(result); i++ {
				number, err := strconv.ParseFloat(result[i][1].(string), 64)
				if err != nil {
					ErrorLog.Printf("expecting to be float64 in result, %v", err)
					c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessagef(`expecting to be float64 in result Error: %v`, err))
					return
				}
				ret_result = append(ret_result, [2]float64{result[i][0].(float64), number})
			}
			c.JSON(axerror.REST_STATUS_OK, &ret_result)
			return

		} else if typeStr == "readsizeavg" || typeStr == "writesizeavg" {
			if typeStr == "readsizeavg" {
				metric_type = promcl.READ_SECTOR_METRIC
				metric_type2 = promcl.READ_OPS_METRIC
			} else {
				metric_type = promcl.WRITE_SECTOR_METRIC
				metric_type2 = promcl.WRITE_OPS_METRIC
			}

			vol_result, axErr := promcl.GetVolumeMetric(metric_type, device_name, instance_name, minTime, maxTime, interval)
			if axErr != nil {
				ErrorLog.Printf("Failed to get prometheus %s for %s, device_name %s, instance_name %s, detail: %v", volId, metric_type, device_name, instance_name, *axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}

			vol_result2, axErr := promcl.GetVolumeMetric(metric_type2, device_name, instance_name, minTime, maxTime, interval)
			if axErr != nil {
				ErrorLog.Printf("Failed to get prometheus %s for %s, device_name %s, instance_name %s, detail: %v", volId, metric_type, device_name, instance_name, *axErr)
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}

			// Data points from Prometheus
			result := (*vol_result).Values
			result2 := (*vol_result2).Values
			InfoLog.Printf("Volstats information: result1: %v, result2: %v", result, result2)
			var length int
			if len(result) > len(result2) {
				length = len(result2)
			} else {
				length = len(result)
			}

			var ret_result [][2]float64
			for i := 0; i < length-1; i++ {
				first_number, err1 := strconv.ParseFloat(result[i+1][1].(string), 64)
				second_number, err2 := strconv.ParseFloat(result[i][1].(string), 64)
				thrid_number, err3 := strconv.ParseFloat(result2[i+1][1].(string), 64)
				fourth_number, err4 := strconv.ParseFloat(result2[i][1].(string), 64)
				if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
					ErrorLog.Printf("expecting to be float64 in result, %v, %v, %v, %v", err1, err2, err3, err4)
					c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessagef(`expecting to be float64 in result Error: %v, %v, %v, %v`, err1, err2, err3, err4))
					return
				}
				// Finding sectors per operation (sectors_at_t2 - sectors_at_t1)/(ops_at_t2 - ops_at_t1)
				ret_result = append(ret_result, [2]float64{result[i][0].(float64), (first_number - second_number) / (thrid_number - fourth_number + 1.0)})
			}
			c.JSON(axerror.REST_STATUS_OK, &ret_result)
			return
		} else {
			ErrorLog.Printf("No matching metric types for %s", typeStr)
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessagef("No matching metric types for %s", typeStr))
			return
		}
	}
}

// @Title UpdateVolumeByID
// @Description Update a volume
// @Accept  json
// @Param   id     	 path    string     	      true        "Volume ID."
// @Param   volume   	 body    volume.Volume     true        "Volume."
// @Success 200 {object} volume.Volume
// @Failure 404 {object} axerror.AXError "Volume does not exist"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes/{id} [PUT]
func UpdateVolume() gin.HandlerFunc {
	// NOTE: if and when volume ACLs are enforced (e.g. only the volume owner can make changes to his volumes),
	// we will need to enable user context to verify he has permissions. Until then, user context is not needed.
	return FixtureManagerProxy(false)
}

// @Title DeleteVolume
// @Description Delete volume by ID
// @Accept  json
// @Param   id     	 path    string     true        "Volume ID."
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Volume is in use"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /storage
// @Router /storage/volumes/{id} [DELETE]
func DeleteVolume() gin.HandlerFunc {
	// See note in UpdateVolume() about user context. May need to supply user context to enforce delete permissions.
	return FixtureManagerProxy(false)
}
