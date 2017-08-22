// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package email

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

type Email struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    bool     `json:"html,omitempty"`
	Body    string   `json:"body"`
}

func (e *Email) Send() *axerror.AXError {
	if _, axErr := utils.AxNotifierCl.Post("email", e); axErr != nil {
		return axErr
	}
	return nil
}
