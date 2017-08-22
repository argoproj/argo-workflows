// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"applatix.io/axops/yaml"
	"github.com/gin-gonic/gin"
)

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type YAMLs struct {
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
	Files    []File `json:"files"`
}

func PostYAMLs() gin.HandlerFunc {
	return func(c *gin.Context) {
		y := YAMLs{}
		err := utils.GetUnmarshalledBody(c, &y)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		if len(y.Repo) == 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("repo field is required."))
			return
		}

		if len(y.Branch) == 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("branch field is required."))
			return
		}

		repo := y.Repo
		branch := y.Branch
		revision := y.Revision
		bodies := []interface{}{}

		for _, f := range y.Files {
			bodies = append(bodies, f.Content)
		}

		tool.SCMRWMutex.RLock()
		defer tool.SCMRWMutex.RUnlock()
		// if the repo is reserved word "ax-builtin", those are builtin service templates.
		if _, ok := tool.ActiveRepos[repo]; ok || repo == "ax-builtin" {
			axErr := yaml.HandleYamlUpdateEvent(repo, branch, revision, bodies)
			if axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
			c.JSON(axerror.REST_CREATE_OK, utils.NullMap)
			return
		} else {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("The repository is not available."))
			return
		}

	}
}
