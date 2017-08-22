// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/email"
	"applatix.io/axops/session"
	"applatix.io/axops/utils"
	"applatix.io/common"
)

const (
	SysReqUserInvite  = 1
	SysReqUserConfirm = 2
	SysReqPassReset   = 3
)

const DEFAULT_REQUEST_DURATION = 24 * time.Hour

type SystemRequest struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	Username     string            `json:"user_name"`
	Target       string            `json:"target"`
	TargetBase64 string            `json:"-"`
	Type         int64             `json:"type"`
	Expiry       int64             `json:"expiry"`
	Hostname     string            `json:"-"`
	Data         map[string]string `json:"data"`
}

func (r *SystemRequest) Create(lifeDuration time.Duration) (*SystemRequest, *axerror.AXError) {
	if r.UserID == "" || r.Username == "" {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing user information during system request creation.")
	}

	if r.Target == "" {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing target information during system request creation.")

	}

	if r.Type == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing type information during system request creation.")

	}

	r.ID = session.GenerateSessionID()
	r.ID = strings.Replace(r.ID, "-", "", -1)
	r.ID = strings.Replace(r.ID, "_", "", -1)
	r.ID = strings.Replace(r.ID, "=", "", -1)
	r.Expiry = time.Now().Add(lifeDuration).Unix()
	r.Hostname = common.GetPublicDNS()
	r.TargetBase64 = base64.URLEncoding.EncodeToString([]byte(r.Target))

	if r.Type == SysReqUserInvite {
		if group, ok := r.Data["group"]; !ok || group == "" {
			return nil, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing group information during user invitation request creation.")
		}
	}

	if axErr := r.Save(); axErr != nil {
		return nil, axErr
	}

	return r, nil
}

func (r *SystemRequest) Reload() (*SystemRequest, *axerror.AXError) {
	if r.ID == "" {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("Missing request ID.")
	}

	request, axErr := GetSysReqById(r.ID)
	if axErr != nil {
		return nil, axErr
	}

	if request == nil {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessagef("Cannot find request with ID: %v", r.ID)
	}

	return request, nil
}

func (r *SystemRequest) Save() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, SysReqTableName, r); axErr != nil {
		return axErr
	}
	return nil
}

func (r *SystemRequest) Delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, SysReqTableName, []*SystemRequest{r})
	if axErr != nil {
		utils.ErrorLog.Printf("Delete request failed:%v\n", axErr)
	}
	return nil
}

func (r *SystemRequest) Validate() *axerror.AXError {
	if r.Expiry > time.Now().Unix() {
		return nil
	} else {
		r.Delete()
		return axerror.ERR_API_INVALID_REQ.NewWithMessage("The link is expired.")
	}
}

func (r *SystemRequest) SendRequest() *axerror.AXError {
	var body bytes.Buffer
	email := email.Email{
		To:   []string{r.Target},
		Html: true,
	}
	switch r.Type {
	case SysReqUserInvite:
		email.Subject = UserOnboardSubject
		signUpURL := fmt.Sprintf("https://%s/setup/signup/%s", r.Hostname, r.ID)
		if r.Data["singleUser"] == "true" {
			signUpURL += ";email=" + r.TargetBase64
		}
		var err error
		context := map[string]string{
			"SignupURL": signUpURL,
		}
		if r.Data["sandbox"] == "true" {
			err = SandboxUserOnboardBody.Execute(&body, context)
		} else {
			err = UserOnboardBody.Execute(&body, context)
		}
		if err != nil {
			return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to create the email body:%v", err)
		}
	case SysReqUserConfirm:
		email.Subject = VerifyEmailSubject
		err := VerifyEmailBody.Execute(&body, r)
		if err != nil {
			return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to create the email body:%v", err)
		}
	case SysReqPassReset:
		email.Subject = ResetPasswordSubject
		err := ResetPasswordBody.Execute(&body, r)
		if err != nil {
			return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to create the email body:%v", err)
		}
	default:
		panic(fmt.Sprintf("Unexpected system request type:%v", r.Type))
	}

	email.Body = body.String()

	if axErr := email.Send(); axErr != nil {
		return axErr
	}
	return nil
}

func GetSysReqById(id string) (*SystemRequest, *axerror.AXError) {
	sysReqs, axErr := GetSysReqs(map[string]interface{}{
		SysReqID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(sysReqs) == 0 {
		return nil, nil
	}

	r := sysReqs[0]
	return &r, nil
}

func GetSysReqsByTarget(target string) ([]SystemRequest, *axerror.AXError) {
	reqs, axErr := GetSysReqs(map[string]interface{}{
		SysReqTarget: target,
	})

	if axErr != nil {
		return nil, axErr
	}

	return reqs, axErr
}

func GetSysReqs(params map[string]interface{}) ([]SystemRequest, *axerror.AXError) {
	sysReqs := []SystemRequest{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, SysReqTableName, params, &sysReqs)
	if axErr != nil {
		return nil, axErr
	}
	return sysReqs, nil
}
