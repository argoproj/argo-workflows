package email

import (
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
)

type Email struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    bool     `json:"html,omitempty"`
	Body    string   `json:"body"`
}

func (e *Email) Send() *axerror.AXError {
	if _, axErr := utils.AxNotifierCl.Post("notifications/email", e); axErr != nil {
		return axErr
	}
	return nil
}
