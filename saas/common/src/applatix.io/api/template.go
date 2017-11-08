package api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"applatix.io/axerror"
	"applatix.io/axops/service"
)

type TemplatesData struct {
	Data []service.EmbeddedTemplateIf `json:"data"`
}

func (td *TemplatesData) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}
	//data := objMap["data"]
	var rawList []*json.RawMessage
	err = json.Unmarshal(*objMap["data"], &rawList)
	if err != nil {
		return err
	}
	//rawList := ([]*json.RawMessage)(objMap["data"])
	data := make([]service.EmbeddedTemplateIf, len(rawList))
	for i, raw := range rawList {
		tmpl, axErr := service.UnmarshalEmbeddedTemplate(*raw)
		if axErr != nil {
			return fmt.Errorf(axErr.Error())
		}
		data[i] = tmpl
	}
	td.Data = data
	return nil
}

func (c *ArgoClient) GetTemplateByName(name, repo, branch string) (service.EmbeddedTemplateIf, *axerror.AXError) {
	var templatesData TemplatesData
	uri := fmt.Sprintf("templates?name=%s&repo=%s&branch=%s", url.QueryEscape(name), url.QueryEscape(repo), url.QueryEscape(branch))
	axErr := c.get(uri, &templatesData)
	if axErr != nil {
		return nil, axErr
	}
	if len(templatesData.Data) == 0 {
		return nil, nil
	}
	if len(templatesData.Data) > 1 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Found multiple templates with name: %s, repo: %s, branch: %s", name, repo, branch)
	}
	return templatesData.Data[0], nil
}
