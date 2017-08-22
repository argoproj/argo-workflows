package deployment

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/notification_center"
	"fmt"
)

func (d *Deployment) SendEventToNotificationCenter() *axerror.AXError {

	utils.InfoLog.Println("Sending event to Notification center")
	eventDetails := map[string]interface{}{}
	eventDetails["App Name"] = d.ApplicationName
	eventDetails["Deployment Name"] = d.Name
	eventDetails["Status"] = d.Status
	eventDetails["Details"] = fmt.Sprintf("https://%%%%AXOPS_EXT_DNS%%%%/app/applications/details/%v/", d.ApplicationGeneration)

	_, axErr := notification_center.Producer.SendMessage(notification_center.CodeDeploymentStatusChanged, "", nil, eventDetails)
	return axErr
}
