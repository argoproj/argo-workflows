// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Label API [/labels]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/label"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

var labelSearchFields = []string{label.LabelType, label.LabelKey, label.LabelValue}

type LabelsData struct {
	Data []label.Label `json:"data"`
}

// @Title GetLabels
// @Description List labels
// @Accept  json
// @Produce json
// @Param	type		query	string	false	"Type. user, service"
// @Param	key		query	string	false	"Key."
// @Param	value		query	string	false	"Value."
// @Param   	reserved	query   bool    false   "Reserved."
// @Param	search		query	string	false	"Search.\nExample:search=~auth"
// @Success 200 {object} LabelsData
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /labels
// @Router /labels [GET]
func ListLabels() gin.HandlerFunc {
	return func(c *gin.Context) {

		labels := []label.Label{}

		params, axErr := GetContextParams(c, labelSearchFields, []string{label.LabelReserved}, []string{}, []string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		labels, dbErr := label.GetLabels(params)
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
			return
		}

		// Convert to second
		for i, _ := range labels {
			labels[i].Ctime = labels[i].Ctime / 1e6
		}

		resultMap := LabelsData{
			Data: labels,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetLabelByID
// @Description Get label by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of label"
// @Success 200 {object} label.Label
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /labels
// @Router /labels/{id} [GET]
func GetLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		l, axErr := label.GetLabelByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if l == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.JSON(axerror.REST_STATUS_OK, l)
		return
	}
}

// @Title CreateLabel
// @Description Create label
// @Accept	json
// @Param	label	body    label.Label	true        "Label object"
// @Success 201 {object} label.Label
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /labels
// @Router /labels [POST]
func CreateLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		l := &label.Label{}
		err := utils.GetUnmarshalledBody(c, l)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		l, axErr := l.Create()
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		c.JSON(axerror.REST_CREATE_OK, l)
		return
	}
}

// @Title DeleteLabel
// @Description Delete label by ID
// @Accept  json
// @Produce json
// @Param   id     path    string     true       "Label ID."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /labels
// @Router /labels/{id} [DELETE]
func DeleteLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		l, axErr := label.GetLabelByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if l == nil {
			c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
			return
		}

		if l.Type == label.LabelTypeUser {
			users, axErr := user.GetUsersByLabel(l.Key)
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}

			for _, u := range users {
				labels := []string{}
				for _, label := range u.Labels {
					if label != l.Key {
						labels = append(labels, label)
					}
				}
				u.Labels = labels

				axErr = u.Update()
				if axErr != nil {
					c.JSON(axerror.REST_INTERNAL_ERR, axErr)
					return
				}
			}
		}

		axErr = l.Delete()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}
