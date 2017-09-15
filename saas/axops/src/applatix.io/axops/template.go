// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Template API [/templates]
package axops

import (
	"net/http"
	"sort"
	"strings"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"github.com/gin-gonic/gin"
)

// Query parameters
const (
	TemplateDedup = "dedup"
)

// Fake object for swagger
type TemplatesData struct {
	Data []template.BaseTemplate `json:"data"`
}

// @Title ListTemplates
// @Description List templates
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   repo	 query   string     false       "Repo."
// @Param   branch	 query   string     false       "Branch."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,repo which is sorting by name DESC and repo ASC"
// @Param   labels	 query   string     false       "Labels, eg:labels=k1:v1,v2;k2:v1"
// @Param   search	 query   string     false       "Search."
// @Param   search_fields query   string     false       "Search fields."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,repo,branch,description,subtype,cost"
// @Param   dedup	 query   string     false       "Deduplicates result set if parameter is set to true"
// @Success 200 {object} TemplatesData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /templates
// @Router /templates [GET]
func TemplateListHandler(c *gin.Context) {

	dedup := strings.ToLower(c.Request.URL.Query().Get(TemplateDedup)) == "true"
	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && service.GetTemplateETag() == etag {
		c.Status(http.StatusNotModified)
		return
	}

	params, axErr := GetContextParams(c,
		[]string{
			service.TemplateId,
			service.TemplateName,
			service.TemplateDescription,
			service.TemplateType,
			service.TemplateRepo,
			service.TemplateBranch,
			service.TemplateRepoBranch,
		},
		[]string{},
		[]string{},
		[]string{service.TemplateLabels, service.TemplateAnnotations})

	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	for k := range c.Request.URL.Query() {
		if k == QueryCommit {
			params[service.TemplateEnvParameters] = utils.EnvKeyCommit
		}
	}

	tempArray, axErr := service.GetTemplates(params)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	if dedup {
		m := make(map[string]service.EmbeddedTemplateIf)

		for _, item := range tempArray {
			str := item.String()
			m[str] = item
		}

		var new []service.EmbeddedTemplateIf

		for _, t := range m {
			new = append(new, t)
		}

		tempArray = new
	}

	// This piece of code is mainly for UI query optimization. The templates loads quite
	// slow when using the lucene index sorting. The idea is to sort here instead of doing
	// it from axdb. Once we have a faster approach for general sorting, the code can be
	// deleted
	sorts := c.Request.URL.Query().Get("sort")
	if sorts == "" {
		// If no sort, sort by name
		sort.Sort(NameSorter(tempArray))
	}
	resultMap := map[string]interface{}{RestData: tempArray}
	c.Header("ETag", service.GetTemplateETag())
	c.JSON(axdb.RestStatusOK, resultMap)
}

type NameSorter []service.EmbeddedTemplateIf

func (a NameSorter) Len() int           { return len(a) }
func (a NameSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NameSorter) Less(i, j int) bool { return a[i].GetName() < a[j].GetName() }

// @Title GetTemplate
// @Description Get template by id
// @Accept  json
// @Param   templateId   path   string     false       "UUID of the template"
// @ Success 200 {object} template.BaseTemplate
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /templates
// @Router /templates/{templateId} [GET]
func TemplateGetHandler(c *gin.Context, templateid string) {

	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && service.GetTemplateETag() == etag {
		c.Status(http.StatusNotModified)
		return
	}

	if !utils.IsUUID(templateid) {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'%s' is not a valid UUID", templateid))
		return
	}

	template, axErr := service.GetTemplateById(templateid)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	if template != nil {
		c.Header("ETag", service.GetTemplateETag())
		c.JSON(axerror.REST_STATUS_OK, template)
	} else {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND)
	}
}

func TemplateDeleteHandler(c *gin.Context, templateId string) {
	if len(templateId) == 0 {
		c.JSON(axdb.RestStatusInvalid, axerror.ERR_API_INVALID_REQ.New())
		return
	}
	axErr := service.DeleteTemplateById(templateId)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
	} else {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
	}
}
