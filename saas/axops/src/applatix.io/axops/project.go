// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi Project API [/projects]
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/project"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/s3cl"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type ProjectsData struct {
	Data []ProjectInfo `json:"data"`
}

type ProjectInfo struct {
	*project.Project
	Assets *ProjectAssetInfo `json:"assets,omitempty"`
}

type ProjectAssetInfo struct {
	Icon          string `json:"icon,omitempty"`
	Detail        string `json:"detail,omitempty"`
	PublisherIcon string `json:"publisher_icon,omitempty"`
}

// @Title GetProjects
// @Description List projects
// @Accept  json
// @Param   id  	 query   string     false       "ID."
// @Param   name  	 query   string     false       "Name."
// @Param   description	 query   string     false       "Description."
// @Param   repo	 query   string     false       "Repo."
// @Param   branch	 query   string     false       "Branch."
// @Param   repo_branch	 query   string     false       "Repo_Branch, eg:repo_branch=https://bitbucket.org/example.git_master or [{"repo":"https://bitbucket.org/example.git","branch":"master"},{"repo":"https://bitbucket.org/example.git","branch":"test"}] (encode needed)"
// @Param   labels	 query   string     false       "Labels, eg:labels=k1:v1,v2;k2:v1"
// @Param   categories	 query   string     false       "Categories."
// @Param   published	 query   bool       false       "Published. Flag that indicates whether an app is published"
// @Param   search	 query   string     false       "Search."
// @Param   fields	 query   string     false       "Fields, eg:fields=id,name,repo,branch,description"
// @Param   limit	 query   int 	    false       "Limit."
// @Param   offset       query   int        false       "Offset."
// @Param   sort         query   string     false       "Sort, eg:sort=-name,repo which is sorting by name DESC and repo ASC"
// @ Success 200 {object} ProjectsData
// @Failure 400 {object} axerror.AXError "Invalid parameters"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /projects
// @Router /projects [GET]
func ListProjects() gin.HandlerFunc {
	return func(c *gin.Context) {

		if projectsNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := GetContextParams(c,
			[]string{
				project.ProjectID,
				project.ProjectName,
				project.ProjectDescription,
				project.ProjectRepo,
				project.ProjectBranch,
				project.ProjectCategories,
				project.ProjectRepoBranch,
				project.ProjectLabelsKeys,
				project.ProjectLabelsValues,
			},
			[]string{project.ProjectPublished},
			[]string{},
			[]string{project.ProjectLabels})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if projects, axErr := project.GetProjects(params); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			projectList := []*ProjectInfo{}
			// replace project icon url from db to axops url and empty the overview and detail in assets
			for i := range projects {
				projectList = append(projectList, transformToProjectInfo(&projects[i], false))

			}

			c.Header("ETag", project.GetETag())
			resultMap := map[string]interface{}{utils.RestData: projectList}
			c.JSON(axerror.REST_STATUS_OK, resultMap)
		}
	}
}

func transformToProjectInfo(p *project.Project, fetchDetails bool) *ProjectInfo {
	pi := &ProjectInfo{Project: p}
	if p.Assets != nil {
		assets := &ProjectAssetInfo{}
		if p.Assets.Icon != nil && len(p.Assets.Icon.S3Bucket) > 0 {
			assets.Icon = projectIconUrl(p.ID)
		}
		if p.Assets.PublisherIcon != nil && len(p.Assets.PublisherIcon.S3Bucket) > 0 {
			assets.PublisherIcon = projectPublisherIconUrl(p.ID)
		}
		if fetchDetails {
			assets.Detail = readAsset(p.Assets.Detail)
		} else {
			assets.Detail = ""
		}
		pi.Assets = assets
	}
	return pi
}

// @Title GetProjectByID
// @Description Get project by ID
// @Accept  json
// @Param   id     	 path    string     true        "ID of project"
// @ Success 200 {object} ProjectInfo
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /projects
// @Router /projects/{id} [GET]
func GetProject() gin.HandlerFunc {
	return func(c *gin.Context) {

		if projectsNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}

		id := c.Param("id")
		p, axErr := project.GetProjectByID(id)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if p == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		c.Header("ETag", project.GetETag())
		pi := transformToProjectInfo(p, true)
		c.JSON(axerror.REST_STATUS_OK, pi)
		return
	}
}

func projectsNotModified(c *gin.Context) bool {
	if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && project.GetETag() == etag {
		return true
	}
	return false

}

func projectIconUrl(pid string) string {
	return "https://" + common.GetPublicDNS() + "/v1/projects/" + pid + "/icon"
}

func projectPublisherIconUrl(pid string) string {
	return "https://" + common.GetPublicDNS() + "/v1/projects/" + pid + "/publisher_icon"
}

// @Title GetProjectIcon
// @Description Get icon for a project
// @Accept  json
// @Param   id     	 path    string     true        "ID of project"
// @Success 200
// @Failure 404
// @Failure 500
// @Resource /projects
// @Router /projects/{id}/icon [GET]

func GetProjectIcon() gin.HandlerFunc {
	return func(c *gin.Context) {
		if projectsNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}
		id := c.Param("id")
		p, axErr := project.GetProjectByID(id)
		if axErr != nil {
			c.Status(axerror.REST_INTERNAL_ERR)
			return
		}

		if p == nil {
			c.Status(axerror.REST_NOT_FOUND)
			return
		}

		if p.Assets != nil && p.Assets.Icon != nil {
			err := writeIcon(c, p.Assets.Icon)
			if err == nil {
				c.Status(axerror.REST_STATUS_OK)
			} else {
				c.Status(axerror.REST_INTERNAL_ERR)
			}
		} else {
			c.Status(axerror.REST_NOT_FOUND)
		}
		return
	}
}

// @Title GetProjectPublisherIcon
// @Description Get publisher icon for a project
// @Accept  json
// @Param   id     	 path    string     true        "ID of project"
// @Success 200
// @Failure 404
// @Failure 500
// @Resource /projects
// @Router /projects/{id}/publisher_icon [GET]

func GetProjectPublisherIcon() gin.HandlerFunc {
	return func(c *gin.Context) {
		if projectsNotModified(c) {
			c.Status(http.StatusNotModified)
			return
		}
		id := c.Param("id")
		p, axErr := project.GetProjectByID(id)
		if axErr != nil {
			c.Status(axerror.REST_INTERNAL_ERR)
			return
		}

		if p == nil {
			c.Status(axerror.REST_NOT_FOUND)
			return
		}

		if p.Assets != nil && p.Assets.PublisherIcon != nil && strings.HasSuffix(p.Assets.PublisherIcon.Path, ".png") {
			err := writeIcon(c, p.Assets.PublisherIcon)
			if err == nil {
				c.Status(axerror.REST_STATUS_OK)
			} else {
				c.Status(axerror.REST_INTERNAL_ERR)
			}
		} else {
			c.Status(axerror.REST_NOT_FOUND)
		}
		return
	}
}

func writeIcon(c *gin.Context, assetDetail *project.AssetDetail) error {

	output, err := getAssetFromS3(assetDetail)

	if err != nil {
		utils.ErrorLog.Printf("Unable to read project icon %v from s3 due to %v", assetDetail, err)
		return err
	}
	c.Header("Content-Type", *output.ContentType)
	c.Header("ETag", project.GetETag())
	_, err = io.Copy(c.Writer, output.Body)
	output.Body.Close()
	return err
}

func getAssetFromS3(assetDetail *project.AssetDetail) (*s3.GetObjectOutput, error) {
	bucket := assetDetail.S3Bucket
	key := assetDetail.S3Key
	return s3cl.GetObjectFromS3(&bucket, &key)

}

func readAsset(assetDetail *project.AssetDetail) string {

	if assetDetail == nil || len(assetDetail.Path) == 0 {
		return ""
	}
	output, err := getAssetFromS3(assetDetail)

	if err != nil {
		utils.ErrorLog.Printf("Unable to read project asset %v from s3 due to %v", assetDetail, err)
		return ""
	}
	d, err := ioutil.ReadAll(output.Body)
	output.Body.Close()
	if err != nil {
		utils.ErrorLog.Printf("Unable to read project asset %v from s3 due to %v", assetDetail, err)
		return ""
	}
	return string(d)

}
