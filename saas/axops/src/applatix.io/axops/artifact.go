package axops

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"applatix.io/axerror"
	"applatix.io/axops/service"
	"applatix.io/axops/utils"
	"github.com/gin-gonic/gin"
)

// Fake Object to make swagger happy
type Artifact struct {
	ArtifactID        string                 `json:"artifact_id,omitempty"`
	SourceArtifactID  string                 `json:"source_artifact_id,omitempty"`
	ServiceInstanceID string                 `json:"service_instance_id,omitempty"`
	Name              string                 `json:"name,omitempty"`
	IsAlias           int                    `json:"is_alias,omitempty"`
	Description       string                 `json:"description,omitempty"`
	SrcPath           string                 `json:"src_path,omitempty"`
	SrcName           string                 `json:"src_name,omitempty"`
	Excludes          string                 `json:"excludes,omitempty"`
	StorageMethod     string                 `json:"storage_method,omitempty"`
	StoragePath       map[string]interface{} `json:"storage_path,omitempty"`
	InlineStorage     string                 `json:"inline_storage,omitempty"`
	CompressionMode   string                 `json:"compression_mode,omitempty"`
	SymlinkMode       string                 `json:"symlink_mode,omitempty"`
	ArchiveMode       string                 `json:"archive_mode,omitempty"`
	NumByte           int64                  `json:"num_byte,omitempty"`
	NumFile           int64                  `json:"num_file,omitempty"`
	NumDir            int64                  `json:"num_dir,omitempty"`
	NumSymlink        int64                  `json:"num_symlink,omitempty"`
	NumOther          int64                  `json:"num_other,omitempty"`
	NumSkipByte       int64                  `json:"num_skip_byte,omitempty"`
	NumSkip           int64                  `json:"num_skip,omitempty"`
	StoredByte        int64                  `json:"stored_byte,omitempty"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	Timestamp         int64                  `json:"timestamp,omitempty"`
	WorkflowID        string                 `json:"workflow_id,omitempty"`
	Checksum          string                 `json:"checksum,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	RetentionTags     string                 `json:"retention_tags,omitempty"`
	Deleted           int64                  `json:"deleted,omitempty"`
	DeletedDate       int64                  `json:"deleted_date,omitempty"`
	DeletedBy         string                 `json:"delete_by,omitempty"`
	ThirdParty        string                 `json:"third_party,omitempty"`
	RelativePath      string                 `json:"relative_path,omitempty"`
	StructurePath     map[string]interface{} `json:"structure_path,omitempty"`
}

type ArtifactsData struct {
	Data []*Artifact `json:"data"`
}

type ArtifactSpaceData struct {
	Data map[string]int64 `json:"data"`
}

// Fake Object to make swagger happy
type RetentionPolicyData struct {
	Data []*RetentionPolicy `json:"data"`
}

// Fake Object to make swagger happy
type RetentionPolicy struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	Policy      int64  `json:"policy"`
}

//@Title ListRetentionPolicies
//@Description List all retention policies
//@Accept  json
//@Success 200 {object} RetentionPolicyData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /retention_policies
//@Router /retention_policies [GET]
func ListRetentionPolicies() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title GetRetentionPolicies
//@Description List retention policy of a given name
//@Accept  json
//@Param   name     	 path    string  true   "Name of retention policy"
//@Success 200 {object} RetentionPolicyData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /retention_policies
//@Router /retention_policies/{name} [GET]
func GetRetentionPolicy() gin.HandlerFunc {
	return ArtifactProcessor()
}

// @Title CreateRetentionPolicy
// @Description Create a retention policy
// @Accept	json
// @Param	policy	body    RetentionPolicy	true        "Retention policy object"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /retention_policies
// @Router /retention_policies [POST]
func CreateRetentionPolicy() gin.HandlerFunc {
	return ArtifactProcessor()
}

// @Title UpdateRetentionPolicy
// @Description Update a retention policy
// @Accept	json
// @Param       name    path    string  true    "Name of retention policy"
// @Param	policy	body    RetentionPolicy	true        "Retention policy object"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /retention_policies
// @Router /retention_policies/{name} [PUT]
func UpdateRetentionPolicy() gin.HandlerFunc {
	return ArtifactProcessor()
}

// @Title DeleteRetentionPolicy
// @Description Delete a retention policy
// @Accept	json
// @Param       name    path    string  true    "Name of retention policy"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /retention_policies
// @Router /retention_policies/{name} [DELETE]
func DeleteRetentionPolicy() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title ListArtifacts
//@Description List Artifacts satisfying some query condition
//@Accept  json
//@Param   action        query   string     true        "Action(search | retrieve | browse | download | list_tags | get_usage). i.e. retrieve: query single artifact; search: query all artifacts; browse: browse single artifact; download: download artifacts; list_tags: list all artifact tags; get_usage: artifact usage information."
//@Param   artifact_id   query   string     false       "Artifact ID."
//@Param   workflow_id	 query   string     false       "Workflow ID."
//@Param   service_id    query   string     false       "Leaf service ID."
//@Param   is_alias      query   int        false       "Alias flag, eg: 0, 1"
//@Param   deleted       query   string     false       "Deleted, in JSON format"
//@Param   tags		 query   string     false       "Tag array, in JSON format"
//@Param   retention_tags         query   string     false       "Retention tag array, in JSON format."
//@Success 200 {object} ArtifactsData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /artifacts
//@Router /artifacts [GET]
func ListArtifacts() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title ListArtifact
//@Description List Artifact with specified artifact ID
//@Accept  json
//@Param   artifact_id   query   string     false       "Artifact ID."
//@Success 200 {object} ArtifactsData
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /artifacts
//@Router /artifacts/{artifact_id} [GET]
func ListArtifact() gin.HandlerFunc {
	return ArtifactProcessor()
}

func BrowseArtifact() gin.HandlerFunc {
	return ArtifactProcessor()
}

func DownloadArtifact() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title DeleteArtifacts
//@Description Delete Artifacts that satisfy the specified condition
//@Accept  json
//@Param   artifact_id   query   string     false       "Artifact ID."
//@Param   retention_tag   query   string     false       "Retention Tag."
//@Param   deleted_by   query   string     false       "Who does deletion."
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /artifacts
//@Router /artifacts/delete [PUT]
func DeleteArtifacts() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title RestoreArtifacts
//@Description Restore Artifacts that satisfy the specified condition
//@Accept  json
//@Param   artifact_id   query   string     false       "Artifact ID."
//@Param   retention_tag   query   string     false       "Retention Tag."
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /artifacts
//@Router /artifacts/restore [PUT]
func RestoreArtifact() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title TagWorkflows
//@Description Apply a tag to workflows that have artifacts
//@Accept  json
//@Param   tag   query   string     false       "Artifact Tag."
//@Param   workflow_ids   query   string     false       "Workflow ID list, it's a comma separated string. e.g. <wk_id1>,<wk_id2>,..."
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /workflows
//@Router /workflows/tag [PUT]
func TagWorkflows() gin.HandlerFunc {
	return ArtifactProcessor()
}

//@Title UnTagWorkflows
//@Description Remove a tag from workflows that have artifacts
//@Accept  json
//@Param   tag   query   string     false       "Artifact Tag."
//@Param   workflow_ids   query   string     false       "Workflow ID list, it's a comma separated string. e.g. <wk_id1>,<wk_id2>,..."
//@Success 200 {object} MapType
//@Failure 500 {object} axerror.AXError "Internal server error"
//@Resource /workflows
//@Router /workflows/untag [PUT]
func UntagWorkflows() gin.HandlerFunc {
	return ArtifactProcessor()
}

// @Title ArtifactOperationHandler
// @Description Artifact PUT operations for delete/restore/tag/untag/clean
// @Accept  json
// @Param   name  	  query   string     false       "Name."
// @Param   action	  query   string     true        "Action of artifact operation. eg: action=tag | untag | delete | restore | clean"
// @Param   workflow_ids  query   string     false       "Comma separated workflow IDs."
// @Param   tag           query   string     false       "The tag to be tagged/untagged from workflow."
// @Param   deleted_by    query   string     false       "Who delete the artifact."
// @Param   retention_tag query   string     false       "Retention Tag."
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
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,description,status,status_detail,commit,endpoint"
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /artifacts
// @Router /artifacts [PUT]
func ArtifactOperationHandler(c *gin.Context) {
	action := c.Request.URL.Query().Get("action")
	// no action is specified
	if action == "" {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("No action is specified in Request"))
		return
	}
	body := make(map[string]interface{})

	if action == "clean" {
		body["action"] = "clean"
	} else {
		var workflowIds []string
		workflowIdStr := c.Request.URL.Query().Get("workflow_ids")
		if len(workflowIdStr) == 0 {
			utils.InfoLog.Printf("no workflow_ids are specified, query service table")
			// search service table for the workflow
			workflows, axErr := GetWorkflowList(c)
			if axErr != nil {
				if axErr.Code == axerror.ERR_AX_INTERNAL.Code {
					c.JSON(axerror.REST_INTERNAL_ERR, axErr)
					return
				} else { //if axErr.Code == axerror.ERR_AXDB_INVALID_PARAM.Code {
					c.JSON(axerror.REST_BAD_REQ, axErr)
					return
				}
			}
			for _, workflow := range workflows {
				workflowIds = append(workflowIds, workflow.Id)
			}
		} else {
			workflowIds = strings.Split(workflowIdStr, ",")
		}

		body["action"] = action
		body["workflow_ids"] = workflowIds
		params := []string{"retention_tag", "deleted_by", "tag"}
		for _, param := range params {
			if val := c.Request.URL.Query().Get(param); len(val) != 0 {
				body[param] = val
			}
		}
	}

	var jsonReader io.Reader = nil
	if body != nil {
		payloadJson, axErr := json.Marshal(body)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("failed to marshal payload"))
		}
		jsonReader = bytes.NewBuffer(payloadJson)
	}

	client := http.Client{
		Timeout: 5 * time.Minute,
	}
	req, err := http.NewRequest("PUT", "http://axartifactmanager.axsys:9892/v1/artifacts", jsonReader)
	if err != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessage("failed to create http request, err: "+err.Error()))
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, err)
	}
	c.Status(resp.StatusCode)
	for name, value := range resp.Header {
		c.Header(name, value[0])
	}
	defer resp.Body.Close()
	_, _ = io.Copy(c.Writer, resp.Body)
}

func GetWorkflowList(c *gin.Context) ([]*service.Service, *axerror.AXError) {

	params, axErr := GetServiceQueryParamsFromContext(c)
	if axErr != nil {
		return nil, axErr
	}

	utils.InfoLog.Printf("params = %v", params)
	serviceArray, axErr := service.GetServicesFromTable(service.RunningServiceTable, false, params)
	if axErr != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to query workflow objects, err: %v", axErr)
	}

	params, axErr = GetContextTimeParams(c, params)
	if axErr != nil {
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Bad request parameters, err: %v", axErr)
	}

	doneServiceArray, axErr := service.GetServicesFromTable(service.DoneServiceTable, false, params)
	if axErr != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to query workflow objects, err: %v", axErr)
	} else {
		// no sorting, for now always show running ones first
		serviceArray = append(serviceArray, doneServiceArray...)
	}

	return serviceArray, nil
}
