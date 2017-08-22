// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Search API [/search]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"github.com/gin-gonic/gin"
)

type SearchIndexData struct {
	Data []*index.SearchIndex `json:"data"`
}

// @Title ListSearchIndexes
// @Description List search indexes
// @Accept  json
// @Param   type	  query   string     false       "type. eg. services, templates, policies, projects, applications, deployments"
// @Param   key		  query   string     false       "key. eg. name, description, username, status_string, status"
// @Param   search	  query   string     false       "Search."
// @Param   search_fields query   string     false       "Search fields."
// @Success 200 {object} SearchIndexData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /search
// @Router /search/indexes [GET]
func ListSearchIndexes() gin.HandlerFunc {
	return func(c *gin.Context) {
		params, axErr := GetContextParams(c,
			[]string{
				index.SearchIndexKey,
				index.SearchIndexValue,
				index.SearchIndexType,
			},
			[]string{},
			[]string{},
			[]string{})

		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		indexes, dbErr := index.GetSearchIndexes(params)
		if dbErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, dbErr)
			return
		}

		resultMap := SearchIndexData{
			Data: indexes,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}
