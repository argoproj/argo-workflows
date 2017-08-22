// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/auth"
	"applatix.io/axops/auth/saml"
	"encoding/json"
	"fmt"
	"strings"
)

var (
	ErrToolMissingIDPSSOURL = axerror.ERR_API_INVALID_PARAM.NewWithMessage("idp_sso_url is required.")
	ErrToolMissingIDPCert   = axerror.ERR_API_INVALID_PARAM.NewWithMessage("idp_public_cert is required.")
)

type SAMLConfig struct {
	*ToolBase
	ButtonLabel             string `json:"button_label"`
	DisplayName             string `json:"sp_display_name"`
	Description             string `json:"sp_description"`
	IdPSsoUrl               string `json:"idp_sso_url"`
	IDPPublicCert           string `json:"idp_public_cert"`
	SignRequest             *bool  `json:"sign_request"`
	SignedResponseAssertion *bool  `json:"signed_response_assertion"`
	DeflateResponseEncoded  *bool  `json:"deflate_response_encoded"`
	SignedResponse          *bool  `json:"signed_response"`
	IDPEmailAttribute       string `json:"email_attribute"`
	IDPLastNameAttribute    string `json:"last_name_attribute"`
	IDPFirstNameAttribute   string `json:"first_name_attribute"`
	IDPGroupAttribute       string `json:"group_attribute"`
}

type SAMLModel struct {
	ID                      string `json:"id"`
	URL                     string `json:"url"`
	Category                string `json:"category"`
	Type                    string `json:"type"`
	ButtonLabel             string `json:"button_label" description:"default: SAML"`
	DisplayName             string `json:"sp_display_name" description:"Service Provider name, default: Applatix"`
	Description             string `json:"sp_description" description:"Service Provider description, default:Applatix Enterprise DevOps"`
	IdPSsoUrl               string `json:"idp_sso_url" description:"required, Identity provider SSO URL"`
	IDPPublicCert           string `json:"idp_public_cert" description:"required if response is signed, Identity provider public certificate"`
	SignRequest             *bool  `json:"sign_request" description:"indicate to sign the SAML Authnrequest or not, default: true"`
	SignedResponseAssertion *bool  `json:"signed_response_assertion" description:"indicate if SAML response assertion is signed, default: false"`
	DeflateResponseEncoded  *bool  `json:"deflate_response_encoded" description:"indicate if SAML response is deflate encoded, default: false"`
	SignedResponse          *bool  `json:"signed_response" description:"indicate if SAML response is signed, default: true"`
	IDPEmailAttribute       string `json:"email_attribute" description:"email attribute name in SAML response, default: User.Email"`
	IDPLastNameAttribute    string `json:"last_name_attribute" description:"last name attribute name in SAML response, default: User.LastName"`
	IDPFirstNameAttribute   string `json:"first_name_attribute" description:"first name attribute name in SAML response, default: User.FirstName"`
	IDPGroupAttribute       string `json:"group_attribute" description:"group attribute name in SAML response, default: User.Group"`
}

func (t *SAMLConfig) Omit() {
}

func (t *SAMLConfig) Test() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (t *SAMLConfig) pre() (*axerror.AXError, int) {

	t.Category = CategoryAuthentication

	if t.IdPSsoUrl = strings.TrimSpace(t.IdPSsoUrl); t.IdPSsoUrl != "" {
		t.URL = t.IdPSsoUrl
	}

	if t.URL = strings.TrimSpace(t.URL); t.URL != "" {
		t.IdPSsoUrl = t.URL
	}

	if t.ButtonLabel = strings.TrimSpace(t.ButtonLabel); t.ButtonLabel == "" {
		t.ButtonLabel = "SAML"
	}

	if t.DisplayName = strings.TrimSpace(t.DisplayName); t.DisplayName == "" {
		t.DisplayName = "Applatix"
	}

	if t.Description = strings.TrimSpace(t.Description); t.Description == "" {
		t.Description = "Applatix Enterprise DevOps"
	}

	if t.SignRequest == nil {
		signRequest := true
		t.SignRequest = &signRequest
	}

	if t.SignedResponse == nil {
		signedResponse := true
		t.SignedResponse = &signedResponse
	}

	if t.SignedResponseAssertion == nil {
		signedResponseAssertion := false
		t.SignedResponseAssertion = &signedResponseAssertion
	}

	if t.DeflateResponseEncoded == nil {
		deflateResponseEncoded := false
		t.DeflateResponseEncoded = &deflateResponseEncoded
	}

	if t.IDPEmailAttribute = strings.TrimSpace(t.IDPEmailAttribute); t.IDPEmailAttribute == "" {
		t.IDPEmailAttribute = "User.Email"
	}

	if t.IDPFirstNameAttribute = strings.TrimSpace(t.IDPFirstNameAttribute); t.IDPFirstNameAttribute == "" {
		t.IDPFirstNameAttribute = "User.FirstName"
	}

	if t.IDPLastNameAttribute = strings.TrimSpace(t.IDPLastNameAttribute); t.IDPLastNameAttribute == "" {
		t.IDPLastNameAttribute = "User.LastName"
	}

	if t.IDPGroupAttribute = strings.TrimSpace(t.IDPGroupAttribute); t.IDPGroupAttribute == "" {
		t.IDPGroupAttribute = "User.Group"
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SAMLConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryAuthentication {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.IdPSsoUrl == "" {
		return ErrToolMissingIDPSSOURL, axerror.REST_BAD_REQ
	}

	if *t.SignedResponse || *t.SignedResponseAssertion {
		if t.IDPPublicCert == "" {
			return ErrToolMissingIDPCert, axerror.REST_BAD_REQ
		} else {
			//TODO: Add the certification string validation
		}
	}

	tools, axErr := GetToolsByType(TypeSAML)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*SAMLConfig).IdPSsoUrl == t.IdPSsoUrl && oldTool.(*SAMLConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("The idp_sso_url(%v) has been used by another configuration.", t.IdPSsoUrl)), axerror.REST_BAD_REQ
		} else if oldTool.(*SAMLConfig).IdPSsoUrl != t.IdPSsoUrl && oldTool.(*SAMLConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_REQ.NewWithMessage("The system can support only one SAML configuration, please delete the old one first."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *SAMLConfig) PushUpdate() (*axerror.AXError, int) {

	createSAML := func(t *SAMLConfig) (*saml.SAMLScheme, *axerror.AXError) {
		params := map[string]interface{}{}
		config, err := json.Marshal(t)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("%v", err)
		}

		err = json.Unmarshal(config, &params)
		if err != nil {
			return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("%v", err)
		}
		return saml.NewSAMLScheme(params)
	}

	if scheme, axErr := createSAML(t); axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	} else {
		auth.RegisterScheme("saml", scheme)
	}
	return nil, axerror.REST_STATUS_OK
}

func (t *SAMLConfig) pushDelete() (*axerror.AXError, int) {
	auth.UnregisterScheme("saml")
	return nil, axerror.REST_STATUS_OK
}
