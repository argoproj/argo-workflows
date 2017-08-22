// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Cluster API [/cluster]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/cluster"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

type ClusterSettingData struct {
	Data []*cluster.ClusterSetting `json:"data"`
}

// @Title ListClusterSettings
// @Description List cluster settings
// @Accept  json
// @Param   key	 	query   string     false       "key."
// @Param   value	query   string     false       "value."
// @Success 200 {object} ClusterSettingData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /cluster
// @Router /cluster/settings [GET]
func ListClusterSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		params, axErr := GetContextParams(c,
			[]string{
				cluster.ClusterKey,
				cluster.ClusterValue,
			},
			[]string{},
			[]string{},
			[]string{})

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		settings, dbErr := cluster.GetClusterSettings(params)
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
			return
		}

		resultMap := ClusterSettingData{
			Data: settings,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetClusterSettingByKey
// @Description Get cluster setting by key
// @Accept  json
// @Param   key	         path    string     true        "key."
// @Success 200 {object} cluster.ClusterSetting
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /cluster
// @Router /cluster/settings/{key} [GET]
func GetClusterSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		setting, axErr := cluster.GetClusterSetting(key)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if setting == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.JSON(axerror.REST_STATUS_OK, setting)
	}
}

// @Title CreateClusterSetting
// @Description Create cluster setting
// @Accept  json
// @Param   setting   	 body    cluster.ClusterSetting     true        "System setting."
// @Success 201 {object} cluster.ClusterSetting
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /cluster
// @Router /cluster/settings [POST]
func CreateClusterSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		setting := &cluster.ClusterSetting{}
		if err := utils.GetUnmarshalledBody(c, setting); err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		if view, axErr, code := setting.Create(); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			c.JSON(code, view)
			return
		}
	}
}

// @Title UpdateClusterSetting
// @Description Update cluster setting
// @Accept  json
// @Param   key     	 path    string     	      true        "Key of System setting."
// @Param   setting     body    cluster.ClusterSetting     true        "System setting."
// @Success 201 {object} cluster.ClusterSetting
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /cluster
// @Router /cluster/settings/{key} [PUT]
func UpdateClusterSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		old, axErr := cluster.GetClusterSetting(key)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if old == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		new := &cluster.ClusterSetting{}
		if err := utils.GetUnmarshalledBody(c, new); err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		new.Ctime = old.Ctime
		new.Mtime = old.Mtime
		new.Key = old.Key

		if new, axErr, code := new.Update(); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			c.JSON(code, new)
			return
		}
	}
}

// @Title DeleteClusterSettingByKey
// @Description Delete cluster setting by key
// @Accept  json
// @Param   key     	 path    string     true        "Key of System setting."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /cluster
// @Router /cluster/settings/{key} [DELETE]
func DeleteClusterSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		view, axErr := cluster.GetClusterSetting(key)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if view == nil {
			c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
			return
		}

		if axErr, code := view.Delete(); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			c.JSON(code, utils.NullMap)
			return
		}
	}
}
