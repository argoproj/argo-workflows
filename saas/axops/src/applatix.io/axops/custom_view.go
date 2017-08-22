// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Custom View API [/custom_views]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/custom_view"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

type CustomViewData struct {
	Data []custom_view.CustomView `json:"data"`
}

// @Title ListCustomViews
// @Description List custom views
// @Accept  json
// @Param   id	 	 query   string     false       "ID."
// @Param   name	 query   string     false       "Name."
// @Param   type	 query   string     false       "Type."
// @Param   user_id	 query   string     false       "User ID."
// @Param   username	 query   string     false       "User name."
// @Param   limit	 query   int 	    false       "Limit."
// @Param   search	 query   string     false       "Search."
// @Success 200 {object} CustomViewData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /custom_views
// @Router /custom_views [GET]
func ListCustomViews() gin.HandlerFunc {
	return func(c *gin.Context) {
		params, axErr := GetContextParams(c,
			[]string{
				custom_view.CustomViewID,
				custom_view.CustomViewName,
				custom_view.CustomViewType,
				custom_view.CustomViewUserID,
				custom_view.CustomViewUserName,
			},
			[]string{},
			[]string{},
			[]string{})

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		views, dbErr := custom_view.GetCustomViews(params)
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
			return
		}

		resultMap := CustomViewData{
			Data: views,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetCustomViewByID
// @Description Get custom view by ID
// @Accept  json
// @Param   id	         path    string     true        "ID"
// @Success 200 {object} custom_view.CustomView
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /custom_views
// @Router /custom_views/{id} [GET]
func GetCustomView() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		view, axErr := custom_view.GetCustomViewById(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if view == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.JSON(axerror.REST_STATUS_OK, view)
	}
}

// @Title CreateCustomView
// @Description Create custom view
// @Accept  json
// @Param   custom_view   	 body    custom_view.CustomView     true        "Custom view."
// @Success 201 {object} custom_view.CustomView
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /custom_views
// @Router /custom_views [POST]
func CreateCustomView() gin.HandlerFunc {
	return func(c *gin.Context) {
		view := &custom_view.CustomView{}
		if err := utils.GetUnmarshalledBody(c, view); err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		ssnU := GetContextUser(c)
		view.Username = ssnU.Username
		view.UserID = ssnU.ID

		if view, axErr, code := view.Create(); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			c.JSON(code, view)
			return
		}
	}
}

// @Title UpdateCustomView
// @Description Update custom view
// @Accept  json
// @Param   id     	 path    string     	      true        "ID of custom view."
// @Param   custom_view     body    custom_view.CustomView     true        "Custom view."
// @Success 201 {object} custom_view.CustomView
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /custom_views
// @Router /custom_views/{id} [PUT]
func UpdateCustomView() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		old, axErr := custom_view.GetCustomViewById(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if old == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		new := &custom_view.CustomView{}
		if err := utils.GetUnmarshalledBody(c, new); err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("%v", err))
			return
		}

		new.Ctime = old.Ctime
		new.Mtime = old.Mtime
		new.ID = old.ID
		new.UserID = old.UserID
		new.Username = old.Username

		if new, axErr, code := new.Update(); axErr != nil {
			c.JSON(code, axErr)
			return
		} else {
			c.JSON(code, new)
			return
		}
	}
}

// @Title DeleteCustomViewByID
// @Description Delete custom view by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of custom view."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /custom_views
// @Router /custom_views/{id} [DELETE]
func DeleteCustomView() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		view, axErr := custom_view.GetCustomViewById(id)
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
