// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package auth

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"time"
)

var (
	ErrAuthRequestNotFound = axerror.ERR_API_INVALID_REQ.NewWithMessage("Authentication request is not found or expired.")
)

const (
	AUTH_REQUST_RETENTION_NS = 5 * time.Minute
)

type AuthRequest struct {
	ID      string            `json:"id,omitempty"`
	Scheme  string            `json:"scheme,omitempty"`
	Request string            `json:"request,omitempty"`
	Ctime   int64             `json:"ctime,omitempty"`
	Expiry  int64             `json:"expiry,omitempty"`
	Data    map[string]string `json:"data"`
}

func (r *AuthRequest) CreateWithID() (*AuthRequest, *axerror.AXError) {
	if r.ID == "" {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing request ID in authentication request creation.")
	}

	r.Ctime = time.Now().Unix()
	r.Expiry = time.Now().Add(AUTH_REQUST_RETENTION_NS).Unix()

	if axErr := r.Save(); axErr != nil {
		return nil, axErr
	}
	return r, nil
}

func (r *AuthRequest) Validate() *axerror.AXError {
	if r.Expiry > time.Now().Unix() {
		return nil
	} else {
		r.Delete()
		return ErrAuthRequestNotFound
	}
}

func (r *AuthRequest) Save() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, AuthRequestTableName, r); axErr != nil {
		return axErr
	}
	return nil
}

func (r *AuthRequest) Delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, AuthRequestTableName, []*AuthRequest{r})
	if axErr != nil {
		utils.ErrorLog.Printf("Delete authentication request failed:%v\n", axErr)
	}
	return nil
}

func GetAuthRequestById(id string) (*AuthRequest, *axerror.AXError) {
	reqs, axErr := GeAuthRequests(map[string]interface{}{
		AuthRequestID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(reqs) == 0 {
		return nil, nil
	}

	r := reqs[0]
	return &r, nil
}

func GeAuthRequests(params map[string]interface{}) ([]AuthRequest, *axerror.AXError) {
	reqs := []AuthRequest{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, AuthRequestTableName, params, &reqs)
	if axErr != nil {
		return nil, axErr
	}
	return reqs, nil
}
