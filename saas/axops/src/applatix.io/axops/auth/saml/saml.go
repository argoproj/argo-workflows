// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package saml

import (
	"applatix.io/axerror"
	"applatix.io/axops/auth"
	"applatix.io/axops/session"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"fmt"
	"github.com/diego-araujo/go-saml"
	"strings"
)

var (
	ErrMissingSAMLConfig          = axerror.ERR_API_AUTH_SAML_MISS_CONFIG.NewWithMessage("SAML configuration missing.")
	ErrMissingSAMLConfigAttribute = axerror.ERR_API_INVALID_PARAM.New()
	ErrMissingServiceProvider     = axerror.ERR_AX_INTERNAL.NewWithMessage("SAML servier provider is not configured.")
	ErrSignAuthRequest            = axerror.ERR_API_AUTH_SAML_CREATE_REQ_FAILED.New()
	ErrCreateRequestURL           = axerror.ERR_API_AUTH_SAML_CREATE_REQ_FAILED.NewWithMessage("Creating SAML authentication request URL is failed.")
	ErrMissingSAMLResponse        = axerror.ERR_API_AUTH_SAML_INVALID_RESPONSE.NewWithMessage("SAML authentication response is missing.")
	ErrMissingUsername            = axerror.ERR_API_AUTH_SAML_MISS_USERNAME.NewWithMessage("Cannot find a validate email format username in NameID or statements.Please check your IdP and SP configurations. You can either set the NameID to be email in IdP or add email to be a response statement and provide the statement attribute name to SP configuration.")
	ErrInvalidResponse            = axerror.ERR_API_AUTH_SAML_INVALID_RESPONSE.New()
)

type SAMLScheme struct {
	*auth.BaseScheme
	Config          *SAMLConfig
	ServiceProvider *saml.ServiceProviderSettings
}

const AUTH_SAML_SCHEME = "saml"

type SAMLConfig struct {
	ButtonLabel             string `json:"button_label"`
	EntityID                string `json:"-"`
	DisplayName             string `json:"sp_display_name"`
	Description             string `json:"sp_description"`
	PublicCertPath          string `json:"-"`
	PrivateKeyPath          string `json:"-"`
	IdPSsoUrl               string `json:"idp_sso_url"`
	IDPPublicCert           string `json:"idp_public_cert"`
	SignedResponseAssertion bool   `json:"signed_response_assertion"`
	DeflateResponseEncoded  bool   `json:"deflate_response_encoded"`

	SignedResponse        bool   `json:"signed_response"`
	IDPEmailAttribute     string `json:"email_attribute"`
	IDPLastNameAttribute  string `json:"last_name_attribute"`
	IDPFirstNameAttribute string `json:"first_name_attribute"`
	SignRequest           bool   `json:"sign_request"`
	IDPGroupAttribute     string `json:"group_attribute"`
}

func NewSAMLScheme(params map[string]interface{}) (*SAMLScheme, *axerror.AXError) {
	var axErr *axerror.AXError
	config := &SAMLConfig{}

	if config.ButtonLabel, axErr = getSAMLAttributeString(params, "button_label"); axErr != nil {
		utils.InfoLog.Println(axErr)
		utils.InfoLog.Println("The SAML button_label attribute is set to be the default value.")
		config.ButtonLabel = "SAML"
		params["button_label"] = "SAML"
	}

	if config.DisplayName, axErr = getSAMLAttributeString(params, "sp_display_name"); axErr != nil {
		utils.InfoLog.Println(axErr)
		utils.InfoLog.Println("The SAML display_name attribute is set to be the default value.")
		config.DisplayName = "Applatix"
		params["display_name"] = "Applatix"
	}

	if config.Description, axErr = getSAMLAttributeString(params, "sp_description"); axErr != nil {
		utils.InfoLog.Println(axErr)
		utils.InfoLog.Println("The SAML description attribute is set to be the default value.")
		config.Description = "Applatix Enterprise DevOps"
		params["description"] = "Applatix Enterprise DevOps"
	}

	if config.IdPSsoUrl, axErr = getSAMLAttributeString(params, "idp_sso_url"); axErr != nil {
		return nil, axErr
	}

	if config.SignRequest, axErr = getSAMLAttributeBool(params, "sign_request"); axErr != nil {
		utils.InfoLog.Println("The SAML sign_request attribute is set to be the default value: true.")
		config.SignRequest = true
		params["sign_request"] = true
	}

	if config.IDPEmailAttribute, axErr = getSAMLAttributeString(params, "email_attribute"); axErr != nil {
		utils.InfoLog.Println("The SAML email attribute is set to be the default value: User.Email")
		config.IDPEmailAttribute = "User.Email"
		params["email_attribute"] = "User.Email"
	}

	if config.IDPFirstNameAttribute, axErr = getSAMLAttributeString(params, "first_name_attribute"); axErr != nil {
		utils.InfoLog.Println("The SAML first name attribute is set to be the default value: User.FirstName")
		config.IDPFirstNameAttribute = "User.FirstName"
		params["first_name_attribute"] = "User.FirstName"
	}

	if config.IDPLastNameAttribute, axErr = getSAMLAttributeString(params, "last_name_attribute"); axErr != nil {
		utils.InfoLog.Println("The SAML last name attribute is set to be the default value: User.LastName")
		config.IDPLastNameAttribute = "User.LastName"
		params["last_name_attribute"] = "User.LastName"
	}

	//if config.IDPGroupAttribute, axErr = getSAMLAttributeString(params, "idp_group_attribute"); axErr != nil {
	//	return nil, axErr
	//}

	if config.SignedResponse, axErr = getSAMLAttributeBool(params, "signed_response"); axErr != nil {
		utils.InfoLog.Println("The SAML signed_response attribute is set to be the default value: true.")
		config.SignedResponse = true
		params["signed_response"] = true
	}

	if config.SignedResponseAssertion, axErr = getSAMLAttributeBool(params, "signed_response_assertion"); axErr != nil {
		utils.InfoLog.Println("The SAML signed_response_assertion attribute is set to be the default value: false.")
		config.SignedResponseAssertion = false
		params["signed_response_assertion"] = false
	}

	if config.DeflateResponseEncoded, axErr = getSAMLAttributeBool(params, "deflate_response_encoded"); axErr != nil {
		utils.InfoLog.Println("The SAML deflate_response_encoded attribute is set to be the default value: false.")
		config.DeflateResponseEncoded = false
		params["deflate_response_encoded"] = false
	}

	if config.SignedResponse || config.SignedResponseAssertion {
		if config.IDPPublicCert, axErr = getSAMLAttributeString(params, "idp_public_cert"); axErr != nil {
			return nil, axErr
		}
	}

	config.LoadSystemConfigs()

	scheme := &SAMLScheme{&auth.BaseScheme{}, config, nil}
	_, axErr = scheme.CreateServiceProvider()
	if axErr != nil {
		return nil, axErr
	}

	return scheme, nil
}

func getSAMLAttributeString(params map[string]interface{}, name string) (string, *axerror.AXError) {
	attribute, ok := params[name]
	if !ok || attribute.(string) == "" {
		return "", ErrMissingSAMLConfigAttribute.NewWithMessagef("Missing %s in SAML configuration", name)
	}
	return attribute.(string), nil
}

func getSAMLAttributeBool(params map[string]interface{}, name string) (bool, *axerror.AXError) {
	attribute, ok := params[name]
	if !ok {
		return false, ErrMissingSAMLConfigAttribute.NewWithMessagef("Missing %s in SAML configuration", name)
	}
	return attribute.(bool), nil
}

func (c *SAMLConfig) LoadSystemConfigs() {
	// EntityID should be valid URI
	c.EntityID = utils.GetEntityID()
	c.PublicCertPath = utils.GetPublicCertPath()
	c.PrivateKeyPath = utils.GetPrivateKeyPath()
}

func (s *SAMLScheme) Name() string {
	return AUTH_SAML_SCHEME
}

func (s *SAMLScheme) Scheme() map[string]interface{} {
	scheme := map[string]interface{}{}
	scheme["name"] = s.Name()
	scheme["button_label"] = s.Config.ButtonLabel
	scheme["enabled"] = true
	return scheme
}

func (s *SAMLScheme) CreateServiceProvider() (*saml.ServiceProviderSettings, *axerror.AXError) {
	if s.Config == nil || s.Config.IdPSsoUrl == "" {
		return nil, ErrMissingSAMLConfig
	}

	if s.Config.SignedResponseAssertion || s.Config.SignedResponse {
		if s.Config.IDPPublicCert == "" {
			return nil, ErrMissingSAMLConfig.NewWithMessage("Mising IDP public certificate.")
		}
		axErr := utils.WriteToFile(s.Config.IDPPublicCert, "idp_public_cert.crt")
		if axErr != nil {
			return nil, axErr
		}
	}

	serviceProvider := saml.ServiceProviderSettings{
		PublicCertPath:              s.Config.PublicCertPath,
		PrivateKeyPath:              s.Config.PrivateKeyPath,
		IDPSSOURL:                   s.Config.IdPSsoUrl,
		DisplayName:                 s.Config.DisplayName,
		Description:                 s.Config.Description,
		IDPPublicCertPath:           "idp_public_cert.crt",
		Id:                          s.Config.EntityID,
		SPSignRequest:               s.Config.SignRequest,
		IDPSignResponse:             s.Config.SignedResponse,
		IDPSignResponseAssertion:    s.Config.SignedResponseAssertion,
		AssertionConsumerServiceURL: utils.GetSSOURL(),
	}

	serviceProvider.Init()
	s.ServiceProvider = &serviceProvider
	return &serviceProvider, nil
}

func (s *SAMLScheme) Metadata() (string, *axerror.AXError) {
	if s.ServiceProvider == nil {
		return "", ErrMissingServiceProvider
	}
	metadata, err := s.ServiceProvider.GetEntityDescriptor()
	if err != nil {
		return "", axerror.ERR_AX_INTERNAL.NewWithMessagef("Get SAML servier provider metadata failed:%v", err)
	}
	return metadata, nil
}

func (s *SAMLScheme) CreateRequest(data map[string]string) (*auth.AuthRequest, *axerror.AXError) {
	var err error
	if s.ServiceProvider == nil {
		return nil, ErrMissingServiceProvider
	}

	// create request
	authnRequest := s.ServiceProvider.GetAuthnRequest()
	b64XML := ""
	fmt.Println("sign_request:", s.Config.SignRequest)
	if s.Config.SignRequest {
		b64XML, err = authnRequest.CompressedEncodedSignedString(s.ServiceProvider.PrivateKeyPath)
		if err != nil {
			return nil, ErrSignAuthRequest.NewWithMessagef("Failed to sign SAML authentication request:%v", err)
		}
	} else {
		b64XML, err = authnRequest.CompressedEncodedString()
		if err != nil {
			return nil, ErrSignAuthRequest.NewWithMessagef("Failed to encode SAML authentication request:%v", err)
		}
	}

	// create request URL with request embedded
	url, err := saml.GetAuthnRequestURL(s.ServiceProvider.IDPSSOURL, b64XML, s.ServiceProvider.AssertionConsumerServiceURL)
	if err != nil {
		return nil, ErrCreateRequestURL
	}

	if data == nil {
		data = map[string]string{}
	}

	// persist the request
	r := &auth.AuthRequest{
		ID:      authnRequest.ID,
		Scheme:  "saml",
		Request: url,
		Data:    data,
	}

	if _, axErr := r.CreateWithID(); axErr != nil {
		return nil, axErr
	}

	return r, nil
}

func (s *SAMLScheme) ParseResponse(responseStr string) (*saml.Response, *axerror.AXError) {
	if responseStr == "" {
		return nil, ErrMissingSAMLResponse
	}

	if s.Config == nil {
		return nil, ErrMissingSAMLConfig
	}

	if s.ServiceProvider == nil {
		return nil, ErrMissingServiceProvider
	}

	var response *saml.Response
	var err error
	if !s.Config.DeflateResponseEncoded {
		response, err = saml.ParseEncodedResponse(responseStr)
	} else {
		response, err = saml.ParseCompressedEncodedResponse(responseStr)
	}

	if err != nil || response == nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("unable to parse identity provider data: %s - %v", response, err)
	}

	if response.IsEncrypted() {
		if err = response.Decrypt(s.ServiceProvider.PrivateKeyPath); err != nil {
			axErr := axerror.ERR_API_AUTH_SAML_DECRYPTION_FAILED.NewWithMessage("Failed to decrypt the SAML response.")
			axErr.Detail = fmt.Sprintf("ErrDetail:%s,ResponseStr:%s", err.Error(), response.OriginalString())
			return nil, axErr
		}
	}

	utils.DebugLog.Printf("Data received from identity provider decoded: %s\n", response.OriginalString())
	return response, nil
}

func (s *SAMLScheme) Login(params map[string]string) (*user.User, *session.Session, *axerror.AXError) {
	if s.Config == nil {
		return nil, nil, ErrMissingSAMLConfig
	}

	if s.ServiceProvider == nil {
		return nil, nil, ErrMissingServiceProvider
	}

	responseStr, ok := params["response"]
	if !ok {
		return nil, nil, ErrMissingSAMLResponse
	}

	utils.DebugLog.Printf("Data received from identity provider: %s\n", responseStr)

	// parse and validate response
	response, axErr := s.ParseResponse(responseStr)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error while parsing IDP data: %v\n", axErr)
		return nil, nil, axErr
	}

	axErr = ValidateResponseSignature(response, s.ServiceProvider)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error while validing IDP data: %v", axErr)
		if strings.Contains(axErr.Error(), "assertion has expired") {
			return nil, nil, auth.ErrAuthRequestNotFound
		}
		return nil, nil, axErr
	}

	axErr = ValidateResponseConfirmation(response, s.ServiceProvider)
	if axErr != nil {
		utils.ErrorLog.Printf("Got error while validing IDP data: %v", axErr)
		if strings.Contains(axErr.Error(), "assertion has expired") {
			return nil, nil, auth.ErrAuthRequestNotFound
		}
		return nil, nil, axErr
	}

	requestId := GetSAMLRequestID(response)

	if requestId != "" {
		request, axErr := auth.GetAuthRequestById(requestId)
		if axErr != nil {
			return nil, nil, axErr
		}

		if request == nil {
			return nil, nil, auth.ErrAuthRequestNotFound
		}

		if axErr = request.Validate(); axErr != nil {
			return nil, nil, auth.ErrAuthRequestNotFound
		}

		if request.Scheme != "saml" {
			return nil, nil, auth.ErrAuthRequestNotFound
		}

		if url, ok := request.Data["redirect_url"]; ok {
			params["redirect_url"] = url
		}

		axErr = request.Delete()
		if axErr != nil {
			// best effort deletion
			utils.ErrorLog.Printf("Failed to delete the SAML request:%v\n", axErr)
		}
	} else {
		// This is IdP initiated SSO with no request ID
		if s.Config.SignedResponse != true && s.Config.SignedResponseAssertion != true {
			utils.InfoLog.Println("It is recommended to enable response/assertion signature in SAML response!")
		}
	}

	// fetch the Name ID, it may or may not be email format
	var username string
	nameID := strings.ToLower(response.GetNameID())
	if !user.ValidateEmail(nameID) {
		errDetail := fmt.Sprintf("1. NameID %s is not a valid email address.", nameID)
		if s.Config.IDPEmailAttribute != "" {
			// fetch user information from statements
			email := strings.ToLower(response.GetAttribute(s.Config.IDPEmailAttribute))
			if email == "" {
				errDetail = errDetail + fmt.Sprintf("2. Failed to find %s statement attribute in response.", s.Config.IDPEmailAttribute)
				axErr = ErrMissingUsername.New()
				axErr.Detail = errDetail
				return nil, nil, axErr
			} else {
				if !user.ValidateEmail(email) {
					errDetail = errDetail + fmt.Sprintf("2. Emaill attribute '%s' has value '%s' which is not a valid email address.", s.Config.IDPEmailAttribute, email)
					axErr = ErrMissingUsername.New()
					axErr.Detail = errDetail
					return nil, nil, axErr
				} else {
					username = email
				}
			}
		}
	} else {
		username = nameID
	}

	//TODO(hong): get the group information
	u, axErr := user.GetUserByName(username)
	if axErr != nil {
		return nil, nil, axErr
	}

	// Optional and best effort attributes
	firstName := response.GetAttribute(s.Config.IDPFirstNameAttribute)
	lastName := response.GetAttribute(s.Config.IDPLastNameAttribute)

	if u == nil {
		// create user if needed
		u = &user.User{
			Username:    username,
			FirstName:   firstName,
			LastName:    lastName,
			AuthSchemes: []string{"saml"},
			Groups:      []string{user.GroupDeveloper},
			State:       user.UserStateActive,
		}
		u, axErr = s.CreateUser(u)
		if axErr != nil {
			return nil, nil, axErr
		}
	} else {
		hasSAML := false
		for _, scheme := range u.AuthSchemes {
			if scheme == "saml" {
				hasSAML = true
			}
		}

		if !hasSAML {
			u.FirstName = firstName
			u.LastName = lastName
			// add SAML to the scheme list
			u.AuthSchemes = append(u.AuthSchemes, "saml")
			u.Update()
			if axErr != nil {
				return nil, nil, axErr
			}
		}
	}

	if u.State == user.UserStateBanned {
		return nil, nil, auth.ErrUserBanned
	}

	if u.State == user.UserStateInit {
		return nil, nil, auth.ErrUserNotConfirmed
	}

	ssn := &session.Session{
		UserID:   u.ID,
		Username: u.Username,
		State:    u.State,
		Scheme:   AUTH_SAML_SCHEME,
	}

	ssn, axErr = ssn.Create()
	if axErr != nil {
		return nil, nil, axErr
	}

	return u, ssn, nil
}

func GetSAMLRequestID(r *saml.Response) string {
	var idRequest string
	if r.IsEncrypted() {
		idRequest = r.EncryptedAssertion.Assertion.Subject.SubjectConfirmation.SubjectConfirmationData.InResponseTo
	} else {
		idRequest = r.Assertion.Subject.SubjectConfirmation.SubjectConfirmationData.InResponseTo
	}
	return idRequest
}

func ValidateResponseSignature(r *saml.Response, sp *saml.ServiceProviderSettings) *axerror.AXError {
	if err := r.Validate(sp); err != nil {
		return ErrInvalidResponse.NewWithMessage(err.Error())
	}

	if sp.IDPSignResponse {
		if err := r.ValidateResponseSignature(sp); err != nil {
			axErr := ErrInvalidResponse.NewWithMessage("Failed to validate the SAML response signature.")
			axErr.Detail = err.Error()
			return axErr
		}
	}

	if sp.IDPSignResponseAssertion {
		if err := r.ValidateAssertionSignature(sp); err != nil {
			axErr := ErrInvalidResponse.NewWithMessage("Failed to validate the SAML response assertion signature.")
			axErr.Detail = err.Error()
			return axErr
		}
	}

	return nil
}

func ValidateResponseConfirmation(r *saml.Response, sp *saml.ServiceProviderSettings) *axerror.AXError {
	if err := r.ValidateExpiredConfirmation(sp); err != nil {
		axErr := ErrInvalidResponse.NewWithMessage("SAML assertion is expired.")
		axErr.Detail = err.Error()
		return axErr
	}
	return nil
}
