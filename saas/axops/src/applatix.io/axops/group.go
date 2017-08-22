// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Group API [/groups]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/user"
	"github.com/gin-gonic/gin"
)

type GroupsData struct {
	Data []user.Group `json:"data"`
}

// @Title ListGroups
// @Description List groups
// @Accept  json
// @Success 200 {object} GroupsData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /groups
// @Router /groups [GET]
func ListGroups() gin.HandlerFunc {
	return func(c *gin.Context) {
		if groups, axErr := user.GetGroups(map[string]interface{}{}); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			newGroups := []user.Group{}
			for i, _ := range groups {
				//if groups[i].Name != user.GroupSuperAdmin {
				newGroups = append(newGroups, groups[i])
				//}
			}

			resultMap := GroupsData{
				Data: newGroups,
			}

			c.JSON(axerror.REST_STATUS_OK, resultMap)
		}
	}
}
