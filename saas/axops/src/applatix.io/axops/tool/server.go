// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"os"
	"time"
)

var (
	ErrToolMissingPublicCert = axerror.ERR_API_INVALID_PARAM.NewWithMessage("public_cert is required.")
	ErrToolMissingPrivateKey = axerror.ERR_API_INVALID_PARAM.NewWithMessage("private_key is required.")
)

type ServerCertConfig struct {
	*ToolBase
	PublicCert string `json:"public_cert,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

type CertificateModel struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Type       string `json:"type"`
	PublicCert string `json:"public_cert"`
	PrivateKey string `json:"private_key"`
}

func (t *ServerCertConfig) Omit() {
	t.PrivateKey = ""
}

func (t *ServerCertConfig) Test() (*axerror.AXError, int) {

	if t.PrivateKey == "" {
		tools, axErr := GetToolsByType(TypeServer)
		if axErr != nil {
			return axErr, axerror.REST_INTERNAL_ERR
		}

		if len(tools) == 0 {
			return nil, axerror.REST_STATUS_OK
		}

		t.PrivateKey = tools[0].(*ServerCertConfig).PrivateKey
	}

	//TODO: validate the public cert and private key are compatible
	if axErr := ValidateCertKeyPair(t.PublicCert, t.PrivateKey); axErr != nil {
		return axErr, axerror.REST_BAD_REQ
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ServerCertConfig) validate() (*axerror.AXError, int) {

	if t.Category != CategoryCertificate {
		return ErrToolCategoryNotMatchType, axerror.REST_BAD_REQ
	}

	if t.PublicCert == "" {
		return ErrToolMissingPublicCert, axerror.REST_BAD_REQ
	}

	if t.PrivateKey == "" {
		return ErrToolMissingPrivateKey, axerror.REST_BAD_REQ
	}

	if axErr := ValidateCertKeyPair(t.PublicCert, t.PrivateKey); axErr != nil {
		return axErr, axerror.REST_BAD_REQ
	}

	tools, axErr := GetToolsByType(TypeServer)
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	if len(tools) == 0 {
		return nil, axerror.REST_STATUS_OK
	}

	for _, oldTool := range tools {
		if oldTool.(*ServerCertConfig).ID != t.ID {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("You can only have one server certificate."), axerror.REST_BAD_REQ
		}
	}

	return nil, axerror.REST_STATUS_OK
}

func (t *ServerCertConfig) pre() (*axerror.AXError, int) {
	t.URL = "certificate/server"
	return nil, axerror.REST_STATUS_OK
}

func Restart() {
	if IsSystemInitialized {
		time.Sleep(10 * time.Second)
		os.Exit(0)
	} else {
		utils.InfoLog.Println("The server is not initialized, restart is not needed.")
	}
}

func (t *ServerCertConfig) PushUpdate() (*axerror.AXError, int) {
	go Restart()
	utils.InfoLog.Println("The server will be restarted in 10 seconds.")
	return nil, axerror.REST_STATUS_OK
}

func (t *ServerCertConfig) pushDelete() (*axerror.AXError, int) {
	go Restart()
	utils.InfoLog.Println("The server will be restarted in 10 seconds.")
	return nil, axerror.REST_STATUS_OK
}

func (t *ServerCertConfig) Post(old, new interface{}) (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}
