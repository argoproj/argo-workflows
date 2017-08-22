// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package project

import (
	"encoding/json"
	"fmt"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/index"
	"applatix.io/axops/label"
	"applatix.io/axops/utils"
	"applatix.io/template"
)

type Project struct {
	template.ProjectTemplate
	Assets    *Assets `json:"assets,omitempty"`
	Published bool    `json:"published,omitempty"`
}

type Assets struct {
	Icon          *AssetDetail `json:"icon,omitempty"`
	Detail        *AssetDetail `json:"detail,omitempty"`
	PublisherIcon *AssetDetail `json:"publisher_icon,omitempty"`
}

func (a *Assets) Same(b *Assets) bool {
	if b == nil {
		return false
	}
	iconsMatch := (a.Icon == b.Icon) || (a.Icon != nil && b.Icon != nil && *a.Icon == *b.Icon)
	detailsMatch := (a.Detail == b.Detail) || (a.Detail != nil && b.Detail != nil && *a.Detail == *b.Detail)
	publisherIconsMatch := (a.PublisherIcon == b.PublisherIcon) || (a.PublisherIcon != nil && b.PublisherIcon != nil && *a.PublisherIcon == *b.PublisherIcon)
	return iconsMatch && detailsMatch && publisherIconsMatch
}

type AssetDetail struct {
	Path     string `json:"path,omitempty"`
	S3Bucket string `json:"bucket,omitempty"`
	S3Key    string `json:"key,omitempty"`
	S3Etag   string `json:"etag,omitempty"`
}

func (p *Project) ProjectDB() (*projectDB, *axerror.AXError) {
	projectDB := &projectDB{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Repo:        p.Repo,
		Branch:      p.Branch,
		Revision:    p.Revision,
		Categories:  p.Categories,
		Labels:      p.Labels,
		RepoBranch:  p.Repo + "_" + p.Branch,
		Published:   p.Published,
	}

	if p.Actions != nil {
		actionsBytes, err := json.Marshal(p.Actions)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the actions object: %v", err))
		}
		projectDB.Actions = string(actionsBytes)
	} else {
		projectDB.Actions = ""
	}

	if &p.Assets != nil {
		assetsBytes, err := json.Marshal(p.Assets)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the assets object: %v", err))
		}
		projectDB.Assets = string(assetsBytes)
	} else {
		projectDB.Assets = ""
	}

	if projectDB.Labels == nil {
		projectDB.Labels = map[string]string{}
		projectDB.LabelsValues = []string{}
		projectDB.LabelsKeys = []string{}
	} else {

		var labelValues []string
		var labelKeys []string
		for k, v := range projectDB.Labels {
			labelKeys = append(labelKeys, k)
			labelValues = append(labelValues, v)
		}
		projectDB.LabelsValues = labelValues
		projectDB.LabelsKeys = labelKeys
	}

	index.SendToSearchIndexChan("projects", "name", p.Name)
	index.SendToSearchIndexChan("projects", "description", p.Description)
	index.SendToSearchIndexChan("projects", "repo", p.Repo)
	index.SendToSearchIndexChan("projects", "branch", p.Branch)

	return projectDB, nil
}

func (p *Project) Insert() (*Project, *axerror.AXError) {

	p.ID = utils.GenerateUUIDv5(fmt.Sprintf("%s:%s:%s", p.Repo, p.Branch, p.Name))

	// first add project assets to S3
	utils.InfoLog.Println("storing project assets in s3")
	axErr := p.StoreAssets()
	if axErr != nil {
		return nil, axErr
	}
	projectDB, axErr := p.ProjectDB()
	if axErr != nil {
		return nil, axErr
	}
	utils.InfoLog.Println("inserting project in db")
	axErr = projectDB.insert()
	if axErr != nil {
		return nil, axErr
	}

	// add labels
	utils.InfoLog.Println("inserting project labels")
	for key, value := range projectDB.Labels {
		lb := label.Label{
			Type:  label.LabelTypeProject,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return nil, axErr
			}
		}
	}
	UpdateETag()
	return p, nil
}

func (p *Project) UpdateAssets() *axerror.AXError {
	projectFromDB, axErr := GetProjectByID(p.ID)
	if axErr != nil {
		return axErr
	}
	if projectFromDB == nil {
		return nil
	}
	oldAssets := projectFromDB.Assets
	// do nothing if assets have not changed
	if (oldAssets == p.Assets) || (oldAssets != nil && p.Assets != nil && p.Assets.Same(oldAssets)) {
		return nil
	}
	// try to delete old assets
	if oldAssets != nil {
		projectFromDB.TryDeleteAssets()
	}
	// store new assets
	if p.Assets != nil {
		axErr = p.StoreAssets()
		if axErr != nil {
			return axErr
		}

	}
	UpdateETag()
	return nil
}

func (p *Project) Update() (*Project, *axerror.AXError) {
	if p.ID == "" {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Project ID is missing")
	}

	axErr := p.UpdateAssets()
	if axErr != nil {
		return nil, axErr
	}

	projectDB, axErr := p.ProjectDB()
	if axErr != nil {
		return nil, axErr
	}

	axErr = projectDB.update()
	if axErr != nil {
		return nil, axErr
	}

	for key, value := range projectDB.Labels {
		lb := label.Label{
			Type:  label.LabelTypeProject,
			Key:   key,
			Value: value,
		}

		if _, axErr := lb.Create(); axErr != nil {
			if axErr.Code == axerror.ERR_API_DUP_LABEL.Code {
				continue
			} else {
				return nil, axErr
			}
		}
	}

	UpdateETag()
	return p, nil
}

func (p *Project) Delete() *axerror.AXError {

	// get the project with all the details
	completeProject, axErr := GetProjectByID(p.ID)
	if axErr != nil {
		return axErr
	}

	//try to delete assets
	completeProject.TryDeleteAssets()

	projectDB, axErr := completeProject.ProjectDB()
	if axErr != nil {
		return axErr
	}

	UpdateETag()
	return projectDB.delete()
}

func (p *Project) StoreAssets() *axerror.AXError {
	if p.Assets == nil {
		return nil
	}
	assets := &Assets{}
	var axErr *axerror.AXError = nil
	if p.Assets.Icon != nil && len(p.Assets.Icon.Path) > 0 {
		if assets.Icon, axErr = storeProjectAssetInS3(p, p.Assets.Icon.Path); axErr != nil {
			return axErr
		}
	}
	if p.Assets.Detail != nil && len(p.Assets.Detail.Path) > 0 {
		if assets.Detail, axErr = storeProjectAssetInS3(p, p.Assets.Detail.Path); axErr != nil {
			return axErr
		}
	}
	if p.Assets.PublisherIcon != nil && len(p.Assets.PublisherIcon.Path) > 0 {
		if assets.PublisherIcon, axErr = storeProjectAssetInS3(p, p.Assets.PublisherIcon.Path); axErr != nil {
			return axErr
		}
	}
	utils.InfoLog.Printf("assets stored: %v", assets)
	p.Assets = assets
	UpdateETag()
	return nil

}

func (p *Project) TryDeleteAssets() {

	utils.InfoLog.Printf("try deleting assets for project: %v", p)
	if p.Assets == nil {
		return
	}
	// delete icon
	if p.Assets.Icon != nil && len(p.Assets.Icon.Path) > 0 && len(p.Assets.Icon.S3Key) > 0 {
		deleteProjectAssetFromS3(p, p.Assets.Icon.Path)
	}
	// delete detail
	if p.Assets.Detail != nil && len(p.Assets.Detail.Path) > 0 && len(p.Assets.Detail.S3Key) > 0 {
		deleteProjectAssetFromS3(p, p.Assets.Detail.Path)
	}

	// delete publisher icon
	if p.Assets.PublisherIcon != nil && len(p.Assets.PublisherIcon.Path) > 0 && len(p.Assets.PublisherIcon.S3Key) > 0 {
		deleteProjectAssetFromS3(p, p.Assets.PublisherIcon.Path)
	}

	UpdateETag()
}

func storeProjectAssetInS3(p *Project, assetPath string) (*AssetDetail, *axerror.AXError) {

	params := map[string]interface{}{
		"repo":   p.Repo,
		"branch": p.Branch,
		"path":   assetPath,
	}

	result := &AssetDetail{Path: assetPath}
	if axErr, _ := utils.DevopsCl.Put2("scm/files", params, nil, result); axErr != nil {
		// log error but move forward
		utils.ErrorLog.Printf("storing asset in s3 failed for project %v with asset %v due to error:%v", p, assetPath, axErr)
		return nil, axErr
	}
	utils.InfoLog.Printf("stored asset %v", result)
	return result, nil
}

func deleteProjectAssetFromS3(p *Project, assetPath string) *axerror.AXError {

	params := map[string]interface{}{
		"repo":   p.Repo,
		"branch": p.Branch,
		"path":   assetPath,
	}

	utils.InfoLog.Printf("deleting asset %v from s3 for project %v", assetPath, p)
	if axErr, _ := utils.DevopsCl.Delete2("scm/files", params, nil, nil); axErr != nil {
		// log error but move forward
		utils.ErrorLog.Printf("deleting asset from s3 failed for project %v with asset %v due to error:%v", p, assetPath, axErr)
		return axErr
	}

	return nil
}

func GetProjectByID(id string) (*Project, *axerror.AXError) {
	projects, axErr := GetProjects(map[string]interface{}{
		ProjectID: id,
	})
	if axErr != nil {
		return nil, axErr
	}

	if len(projects) == 0 {
		return nil, nil
	}

	return &projects[0], nil
}

func GetProjects(params map[string]interface{}) ([]Project, *axerror.AXError) {

	if params != nil && params[axdb.AXDBSelectColumns] != nil {
		fields := params[axdb.AXDBSelectColumns].([]string)
		fields = append(fields, ProjectID)
		fields = utils.DedupStringList(fields)
		params[axdb.AXDBSelectColumns] = fields
	}

	projects := []Project{}
	projectDBs, axErr := getProjectDBs(params)
	if axErr != nil {
		return nil, axErr
	}

	for i := range projectDBs {
		project, axErr := projectDBs[i].project()
		if axErr != nil {
			return nil, axErr
		}
		projects = append(projects, *project)
	}
	return projects, nil
}
