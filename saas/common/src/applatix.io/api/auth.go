package api

import (
	"applatix.io/axerror"
	"applatix.io/axops"
)

var emptyMap = map[string]interface{}{}

func (c *ArgoClient) Login() (*axops.LoginInfo, *axerror.AXError) {
	creds := axops.LoginCredential{
		Username: c.Config.Username,
		Password: c.Config.Password,
	}
	var loginInfo axops.LoginInfo
	err := c.post("auth/login", creds, &loginInfo)
	return &loginInfo, err
}

func (c *ArgoClient) Logout() {
	c.post("auth/logout", emptyMap, nil)
}
