package service

import "text/template"

var onStartBody = template.Must(template.New("onStartBody").Parse(
	`<html>
<body>
  <table class="email-container" style="font-size: 14px;color: #333;font-family: arial;">
    <tr>
      <td class="msg-content" style="padding: 20px 0px;">
        New {{.Name}} job is started on {{.Repo}}:{{.Branch}} by {{.Submitter}}.
      </td>
    </tr>
    <tr>
      <td class="commit-details" style="padding: 20px 0px;">
        <table cellspacing="0" style="border-left: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;border-top: 1px solid #e3e3e3;">
           <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Cluster</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Cluster}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Author</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Author}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Repo</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Repo}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Branch</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Branch}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Description</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Description}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Revision</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Revision}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Submitter</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Submitter}}</td>
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
<![endif]--><a href="{{.Link}}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">VIEW JOB</a></div>
      </td>
    </tr>
    <tr>
      <td class="thank-you" style="padding-top: 20px;line-height: 22px;">
          Thanks<br>
        Team Applatix
      </td>
    </tr>
  </table>
</body>
</html>`))

var onSuccessBody = template.Must(template.New("onSuccessBody").Parse(
	`<html>
<body>
  <table class="email-container" style="font-size: 14px;color: #333;font-family: arial;">
    <tr>
      <td class="msg-content" style="padding: 20px 0px;">
        The {{.Name}} job is  <span style="color:green;">successful</span> on {{.Repo}}:{{.Branch}}. The job was triggered by {{.Submitter}}.
      </td>
    </tr>
    <tr>
      <td class="commit-details" style="padding: 20px 0px;">
        <table cellspacing="0" style="border-left: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;border-top: 1px solid #e3e3e3;">
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Cluster</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Cluster}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Author</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Author}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Repo</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Repo}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Branch</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Branch}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Description</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Description}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Revision</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Revision}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Submitter</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Submitter}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Runtime</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Runtime}}</td>
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
<![endif]--><a href="{{.Link}}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">VIEW JOB</a></div>
      </td>
    </tr>
    <tr>
      <td class="thank-you" style="padding-top: 20px;line-height: 22px;">
          Thanks<br>
        Team Applatix
      </td>
    </tr>
  </table>
</body>
</html>`))

var onFailedBody = template.Must(template.New("onFailedBody").Parse(
	`<html>
<body>
  <table class="email-container" style="font-size: 14px;color: #333;font-family: arial;">
    <tr>
      <td class="msg-content" style="padding: 20px 0px;">
        The {{.Name}} job is <span style="color:red;">failed</span> on {{.Repo}}:{{.Branch}}. The job was triggered by {{.Submitter}}.
      </td>
    </tr>
    <tr>
      <td class="commit-details" style="padding: 20px 0px;">
        <table cellspacing="0" style="border-left: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;border-top: 1px solid #e3e3e3;">
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Cluster</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Cluster}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Author</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Author}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Repo</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Repo}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Branch</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Branch}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Description</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Description}}</td>
          </tr>
          <tr>
            <td class="item-label" style="font-weight: bold;height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;border-right: 1px solid #e3e3e3;">Revision</td>
            <td class="item-value" style="height: 20px;padding: 10px;border-bottom: 1px solid #e3e3e3;">{{.Revision}}</td>
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
<![endif]--><a href="{{.Link}}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:150px;-webkit-text-size-adjust:none;mso-hide:all;">VIEW JOB</a></div>
      </td>
    </tr>
    <tr>
      <td class="thank-you" style="padding-top: 20px;line-height: 22px;">
          Thanks<br>
        Team Applatix
      </td>
    </tr>
  </table>
</body>
</html>`))
