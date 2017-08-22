package deployment

import (
	"applatix.io/axamm/email"
	"applatix.io/axerror"
	"applatix.io/common"
	"bytes"
	"fmt"
	"strconv"
	"text/template"
)

func (d *Deployment) SendEmail() *axerror.AXError {

	summary := d.summaryEmail()
	var bodyBytes bytes.Buffer
	err := onErrorBody.Execute(&bodyBytes, summary)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	mail := email.Email{
		To:      []string{d.User},
		Subject: fmt.Sprintf("[%v] DEPLOYMENT %v:%v status changes to %v", summary.Status, summary.AppName, summary.Name, summary.Status),
		Html:    true,
		Body:    bodyBytes.String(),
	}

	return mail.Send()
}

func (d *Deployment) summaryEmail() *Summary {

	runtime := d.RunTime / 1e6
	summary := &Summary{
		Cluster:   common.GetPublicDNS(),
		AppName:   d.ApplicationName,
		Name:      d.Name,
		Submitter: d.User,
		Status:    d.Status,
		Runtime:   fmt.Sprintf("%v days %v hours %v minutes", strconv.FormatInt(runtime/86400, 10), strconv.FormatInt((runtime%86400)/3600, 10), strconv.FormatInt((runtime%3600)/60, 10)),
		Cost:      fmt.Sprintf("%v dollars %v cents", strconv.FormatInt(int64(d.Cost)/60, 10), strconv.FormatInt(int64(d.Cost)%60, 10)),
		Link:      fmt.Sprintf("https://%v/app/applications/details/%v/deployment/%v", common.GetPublicDNS(), d.ApplicationGeneration, d.Id),
	}

	if d.StatusDetail != nil {
		if val, ok := d.StatusDetail["code"]; ok {
			summary.Code = val.(string)
		}

		if val, ok := d.StatusDetail["message"]; ok {
			summary.Message = val.(string)
		}

		if val, ok := d.StatusDetail["detail"]; ok {
			summary.Detail = val.(string)
		}
	}

	return summary
}

type Summary struct {
	Cluster   string `json:"cluster"`
	AppName   string `json:"application_name"`
	Name      string `json:"name"`
	Submitter string `json:"submitter"`
	Status    string `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail"`
	Runtime   string `json:"runtime"`
	Cost      string `json:"cost"`
	Link      string `json:"link"`
}

var onErrorBody = template.Must(template.New("onFailedBody").Parse(
	`<html>
<body>
  <table class="email-container" style="font-size: 14px;color: #333;font-family: arial;">
    <tr>
      <td class="commit-details" style="padding: 20px 0px;">
        <table cellspacing="0" style="border-left: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;border-top: 1px solid #e3e3e3;">
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Cluster</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Cluster}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Application</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.AppName}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Name</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Name}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Submitter</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Submitter}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Status</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Status}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Code</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Code}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Message</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Message}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Runtime</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Runtime}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Cost</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Cost}}</td>
          </tr>
        </table>
      </td>
    </tr>
    <tr>
      <td class="view-job">
        <div>
          <!--[if mso]>
  <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{{.Link}}" style="height:40px;v-text-anchor:middle;width:150px;" arcsize="125%" strokecolor="#00BDCE" fillcolor="#7fdee6">
    <w:anchorlock/>
    <center style="color:#333;font-family:arial;font-size:14px;font-weight:bold;">VIEW JOB</center>
  </v:roundrect>
<![endif]--><a href="{{.Link}}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">VIEW DETAIL</a></div>
      </td>
    </tr>
    <tr>
      <td class="thank-you" style="padding-top: 20px;line-height: 22px;">
          Thanks,<br>
        Team Applatix
      </td>
    </tr>
  </table>
</body>
</html>`))
