// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Service API [/services]
package axops

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"applatix.io/axamm/deployment"
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/event"
	"applatix.io/axops/sandbox"
	"applatix.io/axops/service"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/template"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/yhat/wsutil"
)

// Query parameters
const (
	ServiceQueryIsActive = "is_active"
	ServiceQueryCommitID = "commit_id"
	ServiceQueryTemplate = "template"
	IncludeDetails       = "include_details"
)

type LogEntry struct {
	Log string `json:"log,omitempty"`
}

type ServicesData struct {
	Data []*service.Service `json:"data"`
}

var serviceSearchFields = []string{service.ServiceTaskId, service.ServiceName, service.ServiceDescription, service.ServiceUserName}

// @Title ListServices
// @Description List services
// @Accept  json
// @Param   name  	 query   string     false       "Name."
// @Param   template_name	 query   string     false       "Template name."
// @Param   template_id	 query   string     false       "Template ID."
// @Param   description	 query   string     false       "Description."
// @Param   username	 query   string     false       "Username."
// @Param   user_id	 query   string     false       "User ID."
// @Param   min_time	 query   int 	    false       "Min time."
// @Param   max_time	 query   int 	    false       "Max time."
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,repo which is sorting by name DESC and repo ASC"
// @Param   is_active	 query   bool       false       "Is active."
// @Param   task_only	 query   bool       false       "Task only."
// @Param   include_details	 query   bool       false       "Include detail, all the children services will be included."
// @Param   repo	 query   string     false       "Repo."
// @Param   branch	 query   string     false       "Branch."
// @Param   revision	 query   string     false       "Revision."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   labels	 query   string     false       "Labels, eg:labels=k1:v1,v2;k2:v1"
// @Param   status	 query   int     false       "Status, 0:success, 1:waiting, 2:running, -1:failed, -2:canceled, 255: init"
// @Param   policy_id  	 query   string     false       "Policy ID."
// @Param   search  	 query   string     false       "The text to search for."
// @Param   search_fields query   string     false       "Search fields."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit,endpoint"
// @Param   tags          query   string     false       "Artifact tag associated with the service, for full text search purpose. eg: tag=~release"
// @Success 200 {object} ServicesData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /services
// @Router /services [GET]
func ServiceListHandler(c *gin.Context) {

	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && service.GetServiceETag() == etag {
		c.Status(http.StatusNotModified)
		return
	}

	params, axErr := GetServiceQueryParamsFromContext(c)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	serviceArray, restCode, axErr := GetServiceObjectsFromQuery(c, params)
	if axErr == nil {
		returnServiceResults(c, serviceArray)
	} else {
		c.JSON(restCode, axErr)
	}
}

func GetServiceObjectsFromQuery(c *gin.Context, params map[string]interface{}) ([]*service.Service, int, *axerror.AXError) {
	var axErr *axerror.AXError
	isActive := strings.ToLower(c.Request.URL.Query().Get(ServiceQueryIsActive))
	includeDetails := strings.ToLower(c.Request.URL.Query().Get(IncludeDetails)) == "true"

	//showSpace := strings.ToLower(c.Request.URL.Query().Get(ArtifactSpaceMgr)) == "true"
	limit := queryParameterInt(c, QueryLimit)
	offset := queryParameterInt(c, QueryOffset)

	if isActive == "true" {
		serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, includeDetails, params)
		if axErr != nil {
			return nil, axerror.REST_INTERNAL_ERR, axErr
		}
		return serviceArray, axerror.REST_STATUS_OK, nil

	} else if isActive == "false" {

		if limit != 0 {
			params[axdb.AXDBQueryMaxEntries] = limit
		}

		if offset != 0 {
			params[axdb.AXDBQueryOffsetEntries] = offset
		}

		params, axErr = GetContextTimeParams(c, params)

		if axErr != nil {
			return nil, axerror.REST_BAD_REQ, axErr
		}

		doneServiceArray, axErr := service.GetServicesFromTable(service.DoneServiceTable, includeDetails, params)
		if axErr != nil {
			ErrorLog.Printf(fmt.Sprintf("error retrieving from table %s, error %v", service.DoneServiceTable, axErr))
			return nil, axerror.REST_INTERNAL_ERR, axErr
		}

		return doneServiceArray, axerror.REST_STATUS_OK, nil
	}

	serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, includeDetails, params)
	if axErr != nil {
		return nil, axerror.REST_INTERNAL_ERR, axErr
	}

	if limit != 0 {
		params[axdb.AXDBQueryMaxEntries] = limit
	}

	if offset != 0 {
		params[axdb.AXDBQueryOffsetEntries] = offset
	}

	params, axErr = GetContextTimeParams(c, params)

	if axErr != nil {
		return nil, axerror.REST_BAD_REQ, axErr
	}

	doneServiceArray, axErr := service.GetServicesFromTable(service.DoneServiceTable, includeDetails, params)
	if axErr != nil {
		ErrorLog.Printf(fmt.Sprintf("error retrieving from table %s, error %v", service.DoneServiceTable, axErr))
		return nil, axerror.REST_INTERNAL_ERR, axErr
	}
	// no sorting, for now always show running ones first
	serviceArray = append(serviceArray, doneServiceArray...)

	return serviceArray, axerror.REST_STATUS_OK, nil
}

func returnServiceResults(c *gin.Context, serviceList []*service.Service) {

	if sandbox.IsSandboxEnabled() {
		for _, s := range serviceList {
			s.User = sandbox.ReplaceEmailInSandbox(s.User)
		}
	}

	c.Header("ETag", service.GetServiceETag())
	c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{RestData: serviceList})
}

func GetServiceQueryParamsFromContext(c *gin.Context) (param map[string]interface{}, err *axerror.AXError) {
	params, axErr := GetContextParams(c,
		[]string{
			service.ServiceTaskId,
			service.ServiceName,
			service.ServiceTemplateName,
			service.ServiceTemplateId,
			// no lucene index created on 'template' column due to the 32KB limitation
			//service.ServiceTemplateStr,
			service.ServiceCommit,
			service.ServiceDescription,
			service.ServiceUserName,
			service.ServiceRepo,
			service.ServiceBranch,
			service.ServiceRepoBranch,
			service.ServiceRevision,
			service.ServicePolicyId,
			service.ServiceStatusString,
		},
		[]string{},
		[]string{service.ServiceStatus,
			service.ServiceLaunchTime,
		},
		[]string{service.ServiceLabels, service.ServiceAnnotations})

	if axErr != nil {
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Bad request parameters, err: %v", axErr)
	}

	delete(params, axdb.AXDBQueryMaxEntries)
	delete(params, axdb.AXDBQueryOffsetEntries)

	commit_id := c.Request.URL.Query().Get(ServiceQueryCommitID)
	template_id := c.Request.URL.Query().Get(ServiceQueryTemplate)
	fixtures := c.Request.URL.Query().Get(service.ServiceFixtures)

	params[service.ServiceIsTask] = true
	if commit_id != "" {
		params[service.ServiceArguments] = commit_id
	}
	if template_id != "" {
		params[service.ServiceTemplateId] = template_id
	}
	if fixtures != "" {
		params[service.ServiceFixtures+axdb.AXDBMapColumnKeySuffix] = fixtures
	}

	// // if we use full text search and the artifact tags are used in the query, we will pick those with exact match.
	// artifactTags := c.Request.URL.Query().Get(service.ServiceArtifactTags)

	// if artifactTags != "" {
	// 	var luceneSearch *axdb.LuceneSearch
	// 	if params[axdb.AXDBQuerySearch] != nil {
	// 		luceneSearch = params[axdb.AXDBQuerySearch].(*axdb.LuceneSearch)
	// 	} else {
	// 		luceneSearch = axdb.NewLuceneSearch()
	// 	}
	// 	var tags []string
	// 	artifactTags = strings.ToLower(artifactTags)
	// 	if !strings.Contains(artifactTags, "~") {
	// 		tags = strings.Split(artifactTags, ",")
	// 		params[axdb.AXDBQueryExactSearch] = tags
	// 	} else {
	// 		tags = strings.Split(strings.TrimLeft(artifactTags, "~"), ",")
	// 	}
	// 	utils.InfoLog.Printf("[ARTIFACT]: tag list: %v", tags)

	// 	var values []string
	// 	for _, tag := range tags {
	// 		regExpValue := ".*" + tag + ".*"
	// 		values = append(values, regExpValue)
	// 	}

	// 	luceneSearch.AddQueryMust(axdb.NewLuceneRegexpFilterBase(service.ServiceArtifactTags, strings.Join(values, "|")))
	// 	params[axdb.AXDBQuerySearch] = luceneSearch
	// }
	return params, nil
}

// @Title GetService
// @Description Get service details
// @Accept  json
// @Param   serviceId   path   string     true       "UUID of the service"
// @Success 200 {object} service.Service
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /services
// @Router /services/{serviceId} [GET]
func ServiceDetailHandler(c *gin.Context, serviceId string) {

	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && service.GetServiceETag() == etag {
		c.Status(http.StatusNotModified)
		return
	}

	if !utils.IsUUID(serviceId) {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("serviceId '%s' is not a valid UUID", serviceId)))
		return
	}

	s, axErr := service.GetServiceDetail(serviceId, nil)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	if s == nil {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage(fmt.Sprintf("Cannot find service with id %v", serviceId)))
		return
	}

	if sandbox.IsSandboxEnabled() {
		s.User = sandbox.ReplaceEmailInSandbox(s.User)
	}

	c.Header("ETag", service.GetServiceETag())
	c.JSON(axerror.REST_STATUS_OK, s)
}

// @Title GetServiceOutput
// @Description Get service outputs (artifacts)
// @Accept  json
// @Param   serviceId    path   string     false       "UUID of the service"
// @Param   outputName   path   string     false       "The output artifact name"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "Output not found"
// @Resource /services
// @Router /services/{serviceId}/outputs/{outputName} [GET]
func ServiceOutputHandler(c *gin.Context, serviceId string, outputName string) {
	timeout := time.Duration(60 * time.Second)

	client := http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	url := fmt.Sprintf("http://axartifactmanager.axsys:9892/v1/artifacts?action=download&service_instance_id=%s&name=%s", serviceId, outputName)
	utils.InfoLog.Printf("forward request to %s", url)
	resp, err := client.Get(url)
	if err != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, err)
		return
	}
	c.Status(resp.StatusCode)
	for name, value := range resp.Header {
		c.Header(name, value[0])
	}
	defer resp.Body.Close()
	_, _ = io.Copy(c.Writer, resp.Body)
	return
}

// @Title GetServiceJira
// @Description Get Jira tickets associated with this Service object
// @Accept  json
// @Param   serviceId    path   string     false       "UUID of the service"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /services
// @Router /services/{serviceId}/issues [GET]
func ServiceJiraListHandler(c *gin.Context, serviceId string) {
	jiraIds, axErr := service.GetServiceJiraIDs(serviceId)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	c.JSON(axerror.REST_STATUS_OK, jiraIds)
}

//func ServiceJiraUpdateHandler(c *gin.Context, serviceId string, jiraId string) {
//	_, axErr := service.AttachJiraToService(serviceId, jiraId)
//	params := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceId}
//	resultArray, axErr := service.GetServiceMapsFromDB(params)
//	if axErr != nil || len(resultArray) < 1 {
//		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
//		return
//	}
//	serviceMap := resultArray[0]
//	jIds := serviceMap[service.ServiceJiraIssues].([]string)
//	for _, jId := range jIds {
//		if jiraId == jId {
//			utils.InfoLog.Printf("The jira (%s) has already been associated with job (%s)", jId, serviceId)
//			c.JSON(axerror.REST_STATUS_OK, nullMap)
//			return
//		}
//	}
//
//	jIds = append(jIds, jiraId)
//	service.Up
//	c.JSON(axerror.REST_STATUS_OK, nullMap)
//}

// @Title GetServiceLog
// @Description Get service console log
// @Accept  json
// @Param   serviceId    path   string     false       "UUID of the service"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "log not found"
// @Resource /services
// @Router /services/{serviceId}/logs [GET]
func ServiceLogHandler(c *gin.Context, serviceId string) {
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceId}

	resultArray, axErr := service.GetServiceMapsFromDB(params)
	if axErr != nil || len(resultArray) < 1 {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	header := c.Writer.Header()
	header["Content-Type"] = []string{"text/event-stream"}
	header["Transfer-Encoding"] = []string{"chunked"}
	header["X-Content-Type-Options"] = []string{"nosniff"}
	serviceMap := resultArray[0]
	status := int(serviceMap[service.ServiceStatus].(float64))
	if status <= 0 {
		// the task has completed, success or failure. Get the done log.
		timeout := time.Duration(60 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		url := fmt.Sprintf("http://axartifactmanager.axsys:9892/v1/artifacts?action=download&service_instance_id=%s&retention_tags=user-log", serviceId)
		utils.InfoLog.Printf("Forward request to %s", url)
		resp, err := client.Get(url)

		if err != nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("log not found on at: "+url))
			utils.InfoLog.Printf("Log not found on %s)", url)
			return
		}
		c.Status(axerror.REST_STATUS_OK)
		scanner := bufio.NewScanner(resp.Body)
		const maxCapacity = 1024 * 1024
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)
		for scanner.Scan() {
			c.Writer.WriteString("data:" + scanner.Text() + "\n\n")
			c.Writer.Flush()

			select {
			case <-c.Writer.CloseNotify():
				fmt.Println("client closed")
				return
			default:
			}
		}
		if scanner.Err() != nil {
			utils.ErrorLog.Println("Scanner error:", scanner.Err())
		}
	} else {
		// get the live log
		if logUrlData, ok := serviceMap[service.ServiceLogLive]; ok {
			logURLStr := "http://localhost:8001" + logUrlData.(string)
			logUrl, err := url.Parse(logURLStr)
			if err != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, err)
				return
			}
			query := logUrl.Query()
			query.Add("follow", "true")
			logUrl.RawQuery = query.Encode()
			resp, err := http.DefaultClient.Get(logUrl.String())
			if err != nil {
				c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("log not found at: "+logURLStr))
				return
			}
			c.Status(resp.StatusCode)
			scanner := bufio.NewScanner(resp.Body)
			const maxCapacity = 1024 * 1024
			buf := make([]byte, maxCapacity)
			scanner.Buffer(buf, maxCapacity)
			for scanner.Scan() {
				log, _ := json.Marshal(LogEntry{Log: scanner.Text() + "\n"})
				logStr := string(log)
				c.Writer.WriteString("data:" + logStr + "\n\n")
				c.Writer.Flush()

				select {
				case <-c.Writer.CloseNotify():
					fmt.Println("client closed")
					return
				default:
				}
			}
			if scanner.Err() != nil {
				utils.ErrorLog.Println("Scanner error:", scanner.Err())
			}
		}
	}
}

// @Title GetServiceEventsByServiceId
// @Description Get the events associated with the specified service id.
// @Accept  json
// @Param   serviceId    path   string     false       "UUID of the service"
// @Success 200 {object} service.ServiceStatusEvent
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "service not found"
// @Resource /services
// @Router /services/{serviceId}/events [GET]
func ServiceEventHandler(c *gin.Context, serviceId string) {
	ch, axErr := service.GetServiceStatusServiceIdChannel(c, serviceId)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}
	if ch == nil {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("service not found"))
		return
	}

	header := c.Writer.Header()
	header["Content-Type"] = []string{"text/event-stream"}
	header["Transfer-Encoding"] = []string{"chunked"}
	header["X-Content-Type-Options"] = []string{"nosniff"}

	for {
		select {
		case <-c.Writer.CloseNotify():
			fmt.Println("[STREAM] client closed")
			service.ClearServiceStatusIdChannel(c, serviceId)
			return
		case event := <-ch:
			eventStr, _ := json.Marshal(event)
			utils.DebugLog.Printf("[STREAM] writing to context %v event %s", c, string(eventStr))
			code, err := c.Writer.WriteString("data:" + string(eventStr) + "\n\n")
			utils.DebugLog.Printf("[STREAM] Writer error: %v %v", err, code)
			c.Writer.Flush()
		}
	}
}

type RepoBranch struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// @Title GetServiceEvents
// @Description Get all service events, optionally filtered by repo_branch.
// @Accept  json
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Success 200 {object} service.ServiceStatusEvent
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /service
// @Router /service/events [GET]
func ServiceEventsHandler(c *gin.Context) {
	branchStrs := []string{}
	repoBranch := c.Request.URL.Query().Get("repo_branch")
	if repoBranch != "" {
		if strings.HasPrefix(repoBranch, "[{") {
			branches := []RepoBranch{}
			err := json.Unmarshal([]byte(repoBranch), &branches)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Can not unmarshal repo_branch field: %v", err))
				return
			}
			for i := range branches {
				branchStrs = append(branchStrs, branches[i].Repo+"_"+branches[i].Branch)
			}
		} else {
			branchStrs = append(branchStrs, repoBranch)
		}
	}
	ch := service.GetServiceStatusServiceBranchChannel(c, branchStrs)
	header := c.Writer.Header()
	header["Content-Type"] = []string{"text/event-stream"}
	header["Transfer-Encoding"] = []string{"chunked"}
	header["X-Content-Type-Options"] = []string{"nosniff"}

	for {
		select {
		case <-c.Writer.CloseNotify():
			fmt.Println("[STREAM] client closed")
			service.ClearServiceStatusBranchChannel(c)
			return
		case event := <-ch:
			eventStr, _ := json.Marshal(event)
			utils.DebugLog.Printf("[STREAM] writing to context %v event %s", c, string(eventStr))
			code, err := c.Writer.WriteString("data:" + string(eventStr) + "\n\n")
			utils.DebugLog.Printf("[STREAM] Writer error: %v %v", err, code)
			c.Writer.Flush()
		}
	}
}

// @Title GetServiceConsole
// @Description Get service console (websocket)
// @Accept  json
// @Param   serviceId    path   string     true       "UUID of the service"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Failure 404 {object} axerror.AXError "log not found"
// @Resource /services
// @Router /services/{serviceId}/exec [GET]
func ServiceConsoleWebSocket() gin.HandlerFunc {
	consoleURL := &url.URL{
		Scheme: "ws://",
		Host:   "axconsole:81",
	}
	proxy := wsutil.NewSingleHostReverseProxy(consoleURL)
	return gin.WrapH(proxy)
}

// @Title LaunchService
// @Description Launch a service
// @Accept  json
// @Param   run_partial   query  bool      false       "Flag to indicate if only partial workflow is run."
// @Param   service         body    service.Service	true        "Service object"
// @Success 201 {object} service.Service
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /services
// @Router /services [POST]
func ServicePostHandler(c *gin.Context) {
	var isPartial bool = false
	dryRun := strings.ToLower(c.Request.URL.Query().Get("dry_run")) == "true"
	isPartialStr := strings.ToLower(c.Request.URL.Query().Get("run_partial"))
	if isPartialStr == "true" {
		utils.InfoLog.Printf("[PartialRun]: yes, partial_run!")
		isPartial = true
	}

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Failed to read request body, err: "+err.Error()))
		return
	}
	//utils.DebugLog.Printf("Service post payload:\n%s", string(body))
	var s service.Service
	err = json.Unmarshal(body, &s)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid service object, err: "+err.Error()))
		return
	}
	data, _ := json.Marshal(s)
	utils.DebugLog.Printf("Service posted:\n%s", string(data))

	if s.TemplateID != "" {
		t, axErr := service.GetTemplateById(s.TemplateID)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if t == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Cannot find a template with id: %v", s.TemplateID))
			return
		}
		if !isPartial {
			s.Template = t
		}
	}

	//utils.InfoLog.Printf("[PartialRun]: service obj: %v, commit: %v, oldcommit: %v", s, *(s.Commit), *(s.OldCommit))
	if s.Template == nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid service object - template or template_id was not specified"))
		return
	}

	if s.Name == "" {
		s.Name = s.Template.GetName()
		if s.Name == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Service name required"))
			return
		}
	}

	context := service.ServiceContext{User: GetContextUser(c), Root: &s, IsPartial: isPartial}
	context.Arguments = s.Arguments

	if sandbox.IsSandboxEnabled() && context.User != nil && context.User.IsDeveloper() && sandbox.MaxConcurrentJobLimitReached(context.User.ID) {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Max concurrent jobs by a user reached"))
		return
	}

	// Remember username before call to s.InitAll() clobbers it. This will enable us to set the services' user field later
	serviceUser := s.User

	// traverse the template document and generate additional services. This needs to be done top down, because
	// we need to top node's uuid first to populate the launch_path of the lower level services.
	axErr := s.InitAll(&context)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	utils.InfoLog.Printf("[PartialRun]: service obj after initAll: %v", s)
	if s.Commit != nil {
		if s.Description == "" {
			s.Description = s.Commit.Description
		}
	} else {
		commitParamName := service.FindParameterNameWithDefault(s.Template.GetInputs(), "%%"+utils.EnvNSSession+"."+utils.EnvKeyCommit+"%%")
		repoParamName := service.FindParameterNameWithDefault(s.Template.GetInputs(), "%%"+utils.EnvNSSession+"."+utils.EnvKeyRepo+"%%")
		if commitParamName != nil {
			if revision := s.Arguments[*commitParamName]; revision != nil {
				repo := ""
				if repoParamName != nil {
					if repoParam := s.Arguments[*repoParamName]; repoParam != nil {
						repo = *repoParam
					}
				}

				c, _ := commit.GetAPICommitByRevision(*revision, repo)
				if c != nil {
					if s.Description == "" {
						s.Description = c.Description
					}
					s.Commit = c
					s.Commit.Jobs = nil
				}
			}
		}
	}

	// Attempt to set context user from commit or user field when job is submitted by internal system
	if context.User != nil && context.User.Username == "system" {
		if s.Commit != nil {
			utils.DebugLog.Println("Job: Submitted by internal system. Determining user from commit")
			committer := s.Commit.Committer
			if committer != "" {
				utils.DebugLog.Println("Job: Submitted by internal system, find committer " + committer + ".")
				email := utils.GetUserEmail(committer)
				if email != "" {
					utils.DebugLog.Println("Job: Submitted by internal system, find committer email - " + email + ".")
					u, axErr := user.GetUserByName(email)
					if axErr != nil {
						c.JSON(axerror.REST_INTERNAL_ERR, axErr)
						return
					}
					if u != nil {
						utils.DebugLog.Println("Job: Submitted by internal system, find committer account - " + email + ".")
						context.User = u
					} else {
						utils.DebugLog.Println("Job: Submitted by internal system, can not find committer account - " + email + ".")
						context.User.Username = email
					}

					axErr = s.UpdateUserAll(&context)
					if axErr != nil {
						c.JSON(axerror.REST_BAD_REQ, axErr)
						return
					}
				}
			}
		} else if serviceUser != "" {
			utils.DebugLog.Printf("Job: Submitted by internal system. Determining user from 'user' field: %s", serviceUser)
			u, axErr := user.GetUserByName(serviceUser)
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
			if u != nil {
				utils.DebugLog.Printf("Job: user account located: %s", serviceUser)
				context.User = u
			} else {
				utils.DebugLog.Printf("Job: no user account found from: %s", serviceUser)
				context.User.Username = s.User
			}
			axErr = s.UpdateUserAll(&context)
			if axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		} else {
			utils.DebugLog.Println("Job: Submitted by internal system. Unable to determine user from commit or user fields")
		}
	}

	//axErr = s.LabelAll(&context, isPartial)
	//if axErr != nil {
	//	c.JSON(axerror.REST_BAD_REQ, axErr)
	//}

	// for debug purpose
	s.PrintAll()

	if s.PolicyId == "" {
		utils.DebugLog.Println("Job: Submmitted by user, ignore the throttling.")
	} else {

		utils.DebugLog.Printf("Job: Submmitted by policy %v. Pending Jobs [%v/%v] Pending Containers [%v/%v]\n", s.PolicyId, PendingJobNum, common.GetMaxPendingJobs(), PendingContainerNum, common.GetMaxPendingContainers())

		if PendingJobNum > common.GetMaxPendingJobs() {
			utils.InfoLog.Printf("Job: The system is hitting the max pending job limit %v. Please try again later.", common.GetMaxPendingJobs())
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_FORBIDDEN_REQ.NewWithMessagef("The system is hitting the max pending job limit %v. Please try again later.", common.GetMaxPendingJobs()))
			return
		}

		if PendingContainerNum > common.GetMaxPendingContainers() {
			utils.InfoLog.Printf("Job: The system is hitting the max pending container limit %v. Please try again later.", common.GetMaxPendingContainers())
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_FORBIDDEN_REQ.NewWithMessagef("The system is hitting the max pending job limit %v. Please try again later.", common.GetMaxPendingContainers()))
			return
		}
	}

	// Before we save the sevice to the database and submit the job to ADC, substitute arguments and revalidate
	// the templates after substitution has been performed.
	// NOTE: we throw away the substituted version of the template because we do not want to store the substituted
	// version of the template with the service. This is to support the resubmit use case where parameters have changed,
	// but the template remains identical. If we were to store the substituted template, this would not be possible.
	substituted, axErr := s.Preprocess()
	if axErr != nil {
		utils.InfoLog.Printf("Preprocessing failed: %v", axErr)
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	if dryRun {
		utils.InfoLog.Println("Job: Dry run requested. Skip submission and database save")
		c.JSON(axerror.REST_CREATE_OK, substituted)
		return
	}

	axErr = s.SaveAll(&context)
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axErr)
		return
	}

	go SubmitJob(&s, false)

	if s.Commit != nil {
		if s.Commit.Repo != "" && s.Commit.Revision != "" {
			go service.UpdateCommitJobHistory(s.Commit.Revision, s.Commit.Repo)
		}
	}

	go service.UpdateTemplateJobCounts(s.TemplateID, -99999, s.Status)

	c.JSON(axerror.REST_CREATE_OK, s)
	service.UpdateServiceETag()
}

// @Title DeleteService
// @Description Delete a service
// @Accept  json
// @Param   serviceId   path   string     true       "UUID of the service"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /services
// @Router /services/{serviceId} [DELETE]
func ServiceDeleteHandler(c *gin.Context, serviceId string) {
	if len(serviceId) == 0 {
		c.JSON(axerror.REST_BAD_REQ, nullMap)
		return
	}

	s, axErr := service.GetServiceDetail(serviceId, nil)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	if s == nil {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage(fmt.Sprintf("Cannot find service with id %v", serviceId)))
		return
	}

	if s.Status <= 0 {
		c.JSON(axerror.REST_STATUS_OK, nullMap)
		return
	}

	u := GetContextUser(c)

	// in sandbox mode, only admins can delete other user's job
	if sandbox.IsSandboxEnabled() {
		if strings.Compare(u.Username, s.User) != 0 && u.IsDeveloper() {
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_AUTH_PERMISSION_DENIED.NewWithMessage("You don't have enough privilege to perform this operation."))
			c.Abort()
			return
		}
	}

	detail := map[string]interface{}{
		"code":    "CANCELLED",
		"message": fmt.Sprintf("The job is cancelled by %v.", u.Username),
	}

	axErr, code := utils.WorkflowAdcCl.Delete2("workflows/"+serviceId, nil, detail, nil)
	if axErr != nil {
		utils.InfoLog.Printf("[JOB] Cancel job %v/%v returned error(%v): %v\n", serviceId, s.Name, code, axErr)

		if code >= 500 {
			c.JSON(code, axErr)
			return
		} else if code >= 400 {
			axErr := service.HandleServiceUpdate(serviceId, utils.ServiceStatusCancelled, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
		}

	} else {

		if service.ValidateStateTransition(s.Status, utils.ServiceStatusCanceling) {
			detail := map[string]interface{}{
				"code":    "CANCELING",
				"message": fmt.Sprintf("The job is cancelled by %v. The job will be stopped shortly.", u.Username),
			}

			axErr := service.HandleServiceUpdate(serviceId, utils.ServiceStatusCanceling, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
		}

	}

	c.JSON(axerror.REST_STATUS_OK, nullMap)
	service.UpdateServiceETag()
}

// @Title UpdateService
// @Description Update a service. Support update Termination Policy, Labels, Annotations fields.
// @Accept  json
// @Param   serviceId   path   string     true       "UUID of the service"
// @Param   service     body    service.Service	true        "Service object"
// @Success 201 {object} service.Service
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Router /services/{serviceId} [PUT]
func ServicePutHandler(c *gin.Context, serviceId string) {
	if len(serviceId) == 0 {
		c.JSON(axerror.REST_BAD_REQ, nullMap)
		return
	}

	oldMap, axErr := service.GetServiceMapByID(serviceId)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	if oldMap == nil {
		c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
		return
	}

	var s service.Service
	err := utils.GetUnmarshalledBody(c, &s)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid service object, err: "+err.Error()))
		return
	}

	newMap, axErr := s.CreateServiceMap(nil)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	// Support only several field update in Service
	if newMap[service.ServiceAnnotations] != nil {
		oldMap[service.ServiceAnnotations] = newMap[service.ServiceAnnotations]
	}

	if newMap[service.ServiceLabels] != nil {
		oldMap[service.ServiceLabels] = newMap[service.ServiceLabels]
	}

	if newMap[service.ServiceNotifications] != nil && newMap[service.ServiceNotifications].(string) != "" {
		oldMap[service.ServiceNotifications] = newMap[service.ServiceNotifications]
	}

	if newMap[service.ServiceTerminationPolicy] != nil && newMap[service.ServiceTerminationPolicy].(string) != "" {
		oldMap[service.ServiceTerminationPolicy] = newMap[service.ServiceTerminationPolicy]
	}

	var updated service.Service
	if axErr := updated.InitFromMap(oldMap); axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	status := updated.Status
	var table string
	if status <= 0 {
		table = service.DoneServiceTable
	} else {
		table = service.RunningServiceTable
	}

	if axErr := service.UpdateServiceInDB(table, oldMap); axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
		return
	}

	c.JSON(axerror.REST_STATUS_OK, updated)
	service.UpdateServiceETag()
}

func ServicePerfHandler(c *gin.Context, intervalStr string, templateName string) {
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting interval to be int64 got %s", intervalStr)
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return
	}
	minTime := queryParameterInt(c, axdb.AXDBQueryMinTime)
	maxTime := queryParameterInt(c, axdb.AXDBQueryMaxTime)

	params := map[string]interface{}{
		axdb.AXDBIntervalColumnName: interval,
		//ServiceTemplateName:         temaplateName,  don't filter for template
	}
	if minTime != 0 {
		params[axdb.AXDBQueryMinTime] = minTime * 1e6
	}
	if maxTime != 0 {
		params[axdb.AXDBQueryMaxTime] = maxTime * 1e6
	}

	// We don't break down by app for now.
	var waitTimeArray []map[string]interface{}
	axErr := Dbcl.Get(axdb.AXDBAppAXOPS, service.DoneServiceTable, params, &waitTimeArray)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axErr)
	}

	dataLen := len(waitTimeArray)
	throughputArray := make([]PerfData, dataLen)
	delayArray := make([]PerfData, dataLen)
	count := 0
	if dataLen > 0 {
		waitCountColName := service.ServiceWaitTime + axdb.AXDBCountColumnSuffix
		wait10ColName := service.ServiceWaitTime + axdb.AXDB10ColumnSuffix
		wait50ColName := service.ServiceWaitTime + axdb.AXDB50ColumnSuffix
		wait90ColName := service.ServiceWaitTime + axdb.AXDB90ColumnSuffix

		// if isAllStats is "true" we will return both summary and sub-summary on each partition key value
		// we will filter out unneeded rows in axops at this moment.
		// TODO: think to filter out unnecessary result from AXDB
		isAllStats := queryParameter(c, StatsSummaryPlusGroupBy)
		for i := 0; i < dataLen; i++ {
			resultMap := waitTimeArray[i]
			//if isAllStats not defined, or it's false
			if isAllStats == "" || isAllStats == "false" {
				// check if template_name is defined in URL
				tn := queryParameter(c, service.ServiceTemplateName)
				partitionValue := resultMap[service.ServiceTemplateName].(string)
				//template_name isn't defined, we only return the summary
				//if template_name is defined, we only return the sub-summary stat for that template_name
				// TODO: if a set of template names is passed in with template_name URL parameter, we need to deal with it later.
				if tn == "" && partitionValue != "" || tn != "" && partitionValue != tn {
					continue
				}
			}
			throughputArray[count].Time = int64(resultMap[axdb.AXDBTimeColumnName].(float64)) / 1e6
			throughputArray[count].Data = resultMap[waitCountColName].(float64)

			delayArray[count].Time = throughputArray[count].Time
			delayArray[count].Min = resultMap[wait10ColName].(float64) / 1e6
			delayArray[count].Data = resultMap[wait50ColName].(float64) / 1e6
			delayArray[count].Max = resultMap[wait90ColName].(float64) / 1e6
			count++
		}
	}

	resultMap := map[string]interface{}{
		"throughput": throughputArray[0:count],
		"delay":      delayArray[0:count],
	}
	c.JSON(axerror.REST_STATUS_OK, resultMap)
}

func ResubmitServices() {
	params := map[string]interface{}{
		service.ServiceIsTask:      true,
		service.ServiceIsSubmitted: false,
	}

	for {
		serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, false, params)
		if axErr != nil {
			utils.InfoLog.Printf("ResubmitServices failure: %v, will retry", axErr)
			time.Sleep(time.Second * 60)
			continue
		}

		foundErr := false
		for _, s := range serviceArray {
			if s.Status == utils.ServiceStatusInitiating {
				axErr = SubmitJob(s, true)
				if axErr != nil {
					foundErr = true
				}
			}
		}

		if !foundErr {
			return
		}
		time.Sleep(time.Second * 60)
		continue
	}
}

var PendingJobNum int64
var PendingContainerNum int64

func MonitorRunningService() {
	go func() {
		for {
			monitorRunningService()
			// Fixed delay scheduling
			time.Sleep(time.Minute * 5)
		}
	}()
}

func monitorRunningService() {
	var axErr *axerror.AXError
	utils.DebugLog.Println("[JobMonitor] Start monitor running jobs ...")
	// get all running jobs
	params := map[string]interface{}{
		service.ServiceIsTask:  true,
		axdb.AXDBSelectColumns: []string{service.ServiceCost, service.ServiceTerminationPolicy, service.ServiceTemplateStr},
	}

	limit := 20
	pendingJobs := 0
	pendingContainers := 0
	maxTime := time.Now().Unix() * 1e6
	var serviceArray []*service.Service
	for {
		params[axdb.AXDBQueryMaxEntries] = limit
		params[axdb.AXDBQueryMaxTime] = maxTime

		serviceArray, axErr = service.GetServicesFromTable(service.RunningServiceTable, true, params)
		if axErr != nil {
			utils.InfoLog.Printf("[JobMonitor] Get running jobs failed with: %v, will retry later\n", axErr)
			return
		}

		utils.DebugLog.Printf("[JobMonitor] Get %v running jobs\n", len(serviceArray))

		pendingJobs += len(serviceArray)

		for _, s := range serviceArray {
			var totalCost float64
			var reason string
			cancel := false

			pendingContainers += s.GetEstimatedParallelContainers()

			if s.Status != utils.ServiceStatusInitiating {
				utils.DebugLog.Printf("[JobMonitor] Job (%v %v %v) has %v children.\n", s.Name, s.Id, s.Status, len(s.Children))
				for _, child := range s.Children {
					cost := child.Cost
					if child.Template != nil && child.Template.GetType() == template.TemplateTypeContainer {
						utils.DebugLog.Printf("[JobMonitor] Job child (%v %v) cost is %v.\n", child.Name, child.Id, child.Cost)
						totalCost += cost
					}

					// We only respect the root job termination policy
					//if cancel, reason = ShouldTerminate(child, child.Cost, float64(child.RunTime)); cancel == true {
					//	break
					//}
				}

				if !cancel {
					cancel, reason = ShouldTerminate(s, totalCost, float64(s.RunTime))
				}

				if cancel {

					detail := map[string]interface{}{
						"code":    "CANCELLED",
						"message": fmt.Sprintf("The job is cancelled by system due to %v.", reason),
					}

					utils.InfoLog.Printf("[JobMonitor] Canceling job %v %v %v. Reason: %v\n", s.Name, s.Id, s.Status, reason)
					axErr, code := utils.WorkflowAdcCl.Delete2("workflows/"+s.Id, nil, detail, nil)
					if axErr != nil && (code >= 400 && code < 500) {

						utils.InfoLog.Printf("[JobMonitor] Canceling job %v %v %v failed: %v\n", s.Name, s.Id, s.Status, axErr)
						axErr = service.HandleServiceUpdate(s.Id, utils.ServiceStatusCancelled, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
						if axErr != nil {
							utils.InfoLog.Printf("[JobMonitor] Update job %v %v %v status to cancelled failed: %v, will retry later\n", s.Name, s.Id, s.Status, axErr)
						}

					} else {
						detail = map[string]interface{}{
							"code":    "CANCELING",
							"message": fmt.Sprintf("The job is cancelled by system due to %v.", reason),
						}

						axErr = service.HandleServiceUpdate(s.Id, utils.ServiceStatusCanceling, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
						if axErr != nil {
							utils.InfoLog.Printf("[JobMonitor] Update job %v %v %v status to canceling failed: %v, will retry later\n", s.Name, s.Id, s.Status, axErr)
						}
					}
				}
			} else {
				utils.InfoLog.Printf("[JobMonitor] Skip job %v %v %v\n", s.Name, s.Id, s.Status)
			}
		}

		if len(serviceArray) < limit {
			if len(serviceArray) != 0 {
				service.UpdateServiceETag()
			}
			break
		} else {
			uuid, err := gocql.ParseUUID(serviceArray[len(serviceArray)-1].Id)
			if err != nil {
				utils.InfoLog.Printf("Invalid service uuid: " + serviceArray[len(serviceArray)-1].Id)
			}
			maxTime = uuid.Time().UnixNano()/1e3 - 1
		}
	}

	PendingJobNum = int64(pendingJobs)
	PendingContainerNum = int64(pendingContainers)
	utils.DebugLog.Printf("[JobMonitor] PendingJobs: %v  PendingJobContainers: %v\n", PendingJobNum, PendingContainerNum)

	params = map[string]interface{}{
		axdb.AXDBSelectColumns: []string{deployment.ServiceStatus},
	}
	deployments, axErr := deployment.GetLatestDeployments(params, false)
	runningDeployments := []*deployment.Deployment{}
	for _, d := range deployments {
		switch d.Status {
		case deployment.DeployStateTerminated:
			continue
		}
		runningDeployments = append(runningDeployments, d)
	}
	utils.DebugLog.Printf("[JobMonitor] PendingDeployments: %v  PendingDeploymentContainers: %v\n", len(runningDeployments), len(runningDeployments))

	PendingJobNum += int64(len(runningDeployments))
	PendingContainerNum += int64(len(runningDeployments))

	utils.DebugLog.Printf("[JobMonitor] TotalPendingJobs: %v  TotalPendingContainers: %v\n", PendingJobNum, PendingContainerNum)
	utils.DebugLog.Println("[JobMonitor] End monitor running jobs ...")
}
