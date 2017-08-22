// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Fixture API [/fixture]
package axops

import (
	"net/http"

	"applatix.io/axerror"
	"applatix.io/axops/fixture"
	"applatix.io/template"
	"github.com/gin-gonic/gin"
)

type FixtureTemplatesData struct {
	Data []template.FixtureTemplate `json:"data"`
}

type FixtureClassesData struct {
	Data []fixture.Class `json:"data"`
}

type FixtureClassCreateUpdatePayload struct {
	TemplateID string `json:"template_id"`
}

type FixtureInstancesData struct {
	Data []fixture.Instance `json:"data"`
}

func fixturesNotModified(c *gin.Context) bool {
	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && fixture.GetETag() == etag {
		return true
	}
	return false

}

// @Title GetFixtureTemplates
// @Description List fixture templates
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   repo	 query   string     false       "Repo."
// @Param   branch	 query   string     false       "Branch."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   search	 query   string     false       "Search."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,repo,branch,description"
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,repo which is sorting by name DESC and repo ASC"
// @ Success 200 {object} FixtureTemplatesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/templates [GET]
func ListFixtureTemplates() gin.HandlerFunc {
	return func(c *gin.Context) {

		if fixturesNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := GetContextParams(c,
			[]string{
				fixture.TemplateName,
				fixture.TemplateDescription,
				fixture.TemplateRepo,
				fixture.TemplateBranch,
				fixture.TemplateRepoBranch,
			}, []string{}, []string{}, []string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		templates, axErr := fixture.GetFixtureTemplates(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.Header("ETag", fixture.GetETag())
		resultMap := FixtureTemplatesData{
			Data: templates,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetFixtureTemplateByID
// @Description Get a fixture template
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture template."
// @ Success 200 {object} template.FixtureTemplate
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/templates/{id} [GET]
func GetFixtureTemplateByID() gin.HandlerFunc {
	return func(c *gin.Context) {

		if fixturesNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}

		templateID := c.Param("id")
		template, axErr := fixture.GetFixtureTemplateByID(templateID)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		if template != nil {
			c.Header("ETag", fixture.GetETag())
			c.JSON(axerror.REST_STATUS_OK, template)
		} else {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND)
		}
	}
}

// @Title GetFixtureClasses
// @Description List fixture classes
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   search	 query   string     false       "Search."
// @ Success 200 {object} FixtureClassesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/classes [GET]
func ListFixtureClasses() gin.HandlerFunc {
	return func(c *gin.Context) {
		params, axErr := GetContextParams(c,
			[]string{
				fixture.ClassName,
				fixture.ClassDescription,
				fixture.ClassRepo,
				fixture.ClassBranch,
				fixture.ClassStatus,
			}, []string{}, []string{}, []string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		classes, axErr := fixture.GetClasses(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		resultMap := FixtureClassesData{
			Data: classes,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

// @Title GetFixtureClassByID
// @Description Get fixture class by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture class."
// @ Success 200 {object} fixture.Class
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/classes/{id} [GET]
func GetFixtureClassByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		classID := c.Param("id")
		class, axErr := fixture.GetClassByID(classID)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		if class != nil {
			c.JSON(axerror.REST_STATUS_OK, class)
		} else {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND)
		}
	}
}

// @Title CreateFixtureClass
// @Description Create fixture class
// @Accept  json
// @Param   template   	 body    FixtureClassCreateUpdatePayload     true        "Fixture template ID."
// @ Success 200 {object} fixture.Class
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/classes [POST]
func CreateFixtureClass() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title UpdateFixtureClass
// @Description Update fixture class to use a differnt template id
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture class."
// @Param   template   	 body    FixtureClassCreateUpdatePayload     true        "Fixture template ID."
// @ Success 200 {object} fixture.Class
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/classes/{id} [PUT]
func UpdateFixtureClass() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title DeleteFixtureClassByID
// @Description Delete fixture class by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture class."
// @Param   template   	 body    FixtureClassCreateUpdatePayload     true        "Fixture template ID."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/classes/{id} [DELETE]
func DeleteFixtureClass() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title GetFixtureInstances
// @Description List fixture instances
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   class_name	 query   string     false       "Class Name."
// @Param   class_id	 query   string     false       "Class ID."
// @Param   enabled	 query   bool       false       "Enabled, indicate if the fixture instance is enabled for reservation."
// @Param   search	 query   string     false       "Search."
// @Success 200 {object} FixtureInstancesData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances [GET]
func ListFixtureInstances() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title GetFixtureInstanceByID
// @Description Get fixture instance by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture instance."
// @Success 200 {object} fixture.Instance
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances/{id} [GET]
func GetFixtureInstanceByID() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}

// @Title CreateFixtureInstance
// @Description Create fixture instance
// @Accept  json
// @Param   instance   	 body    fixture.Instance     true        "Fixture instance."
// @Success 201 {object} fixture.Instance
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances [POST]
func CreateFixtureInstance() gin.HandlerFunc {
	return FixtureManagerProxy(true)
}

// @Title DeleteFixtureInstanceByID
// @Description Delete fixture instance by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of fixture instance."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances/{id} [DELETE]
func DeleteFixtureInstance() gin.HandlerFunc {
	return FixtureManagerProxy(true)
}

// @Title UpdateFixtureInstance
// @Description Update fixture instance
// @Accept  json
// @Param   id     	 path    string     	      true        "ID of fixture instance."
// @Param   instance   	 body    fixture.Instance     true        "Fixture instance."
// @Success 201 {object} fixture.Instance
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances/{id} [PUT]
func UpdateFixtureInstance() gin.HandlerFunc {
	return FixtureManagerProxy(true)
}

type InstanceActionPayload struct {
	Action     string  `json:"action"`
	Parameters MapType `json:"parameters"`
}

// @Title PerformFixtureInstanceAction
// @Description Perform an action against a fixture instance
// @Accept  json
// @Param   id     	 path    string     	      true        "ID of fixture instance."
// @Param   instance   	 body    InstanceActionPayload     true        "Fixture action."
// @Success 201 {object} fixture.Instance
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/instances/{id}/action [POST]
func PerformFixtureInstanceAction() gin.HandlerFunc {
	return FixtureManagerProxy(true)
}

type FixtureSummaryTotal struct {
	Available int64
	Total     int64
}

type FixtureSummaryData map[string]FixtureSummaryTotal

// @Title GetFixtureSummary
// @Description Gets counts of instance availability and total by a query filter and grouping
// @Accept  json
// @Param   group_by	 query   string     false       "Group by."
// @Param   filters	 query   string     false       "Key value filters."
// @Success 200 {object} FixtureSummaryData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /fixture
// @Router /fixture/summary [GET]
func GetFixtureSummary() gin.HandlerFunc {
	return FixtureManagerProxy(false)
}
