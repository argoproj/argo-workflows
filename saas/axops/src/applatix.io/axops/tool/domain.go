// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

var (
	ErrToolMissingDomain = axerror.ERR_API_INVALID_PARAM.New()
)

type Domain struct {
	Name string `json:"name"`
}

type DomainConfig struct {
	*ToolBase
	Domains    []Domain `json:"domains,omitempty"`
	AllDomains []string `json:"all_domains,omitempty"`
}

type DomainModel struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Type       string   `json:"type"`
	Password   string   `json:"password"`
	Domains    []Domain `json:"domains,omitempty"`
	AllDomains []string `json:"all_domains,omitempty"`
}

func (t *DomainConfig) Omit() {
	t.Password = ""

	domains := &DomainResult{}
	axErr := utils.AxmonCl.Get("domains", nil, domains)
	if axErr == nil {
		t.AllDomains = domains.Result
	}
}

type DomainResult struct {
	Result []string `json:"result"`
}

func (t *DomainConfig) Test() (*axerror.AXError, int) {

	domains := &DomainResult{}
	axErr := utils.AxmonCl.Get("domains", nil, domains)
	if axErr != nil {
		return axErr, 500
	}

	t.AllDomains = domains.Result
	if t.Domains == nil {
		t.Domains = []Domain{}
	}

	domainMap := map[string]interface{}{}
	for _, domain := range domains.Result {
		domainMap[domain] = nil
	}

	for _, domain := range t.Domains {
		if _, ok := domainMap[domain.Name]; !ok {
			return ErrToolMissingDomain.NewWithMessagef("Domain %v is not available. Please check your Route53 configurations.", domain), 500
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *DomainConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryDomainManagement {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeRoute53)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*DomainConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_REQ.NewWithMessage("The system can support only one domain management configuration, please delete the old one first."), axerror.REST_BAD_REQ
		}
	}

	domainMap := map[string]interface{}{}
	for _, domain := range t.AllDomains {
		domainMap[domain] = nil
	}

	for _, domain := range t.Domains {
		if _, ok := domainMap[domain.Name]; !ok {
			return ErrToolMissingDomain.NewWithMessagef("Domain %v is not available. Please check your Route53 configurations.", domain), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *DomainConfig) pre() (*axerror.AXError, int) {

	t.URL = "Amazon Route 53"
	t.Category = CategoryDomainManagement
	t.Type = TypeRoute53

	if t.Domains == nil {
		t.Domains = []Domain{}
	}

	if t.Domains != nil {
		set := map[string]interface{}{}
		for _, domain := range t.Domains {
			if _, ok := set[domain.Name]; ok {
				return axerror.ERR_API_INVALID_REQ.NewWithMessagef("There is duplicated domain(%v) in the domain list.", domain.Name), axerror.REST_BAD_REQ
			} else {
				set[domain.Name] = nil
			}
		}
	}

	domains := &DomainResult{}
	axErr := utils.AxmonCl.Get("domains", nil, domains)
	if axErr != nil {
		return axErr, 500
	}

	t.AllDomains = domains.Result

	return nil, axerror.REST_STATUS_OK
}

func (t *DomainConfig) PushUpdate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *DomainConfig) pushDelete() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *DomainConfig) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
