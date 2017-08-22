// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user

import "text/template"

var VerifyEmailSubject = "Applatix Email Confirmation"
var VerifyEmailBody = template.Must(template.New("VerifyEmail").Parse(`
<html>
<body style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 1.2em">

<h1>Welcome to Applatix!</h1>

<p>Click this link to confirm your account and start using Applatix:</p>
<p><a href="https://{{.Hostname}}/v1/users/{{.Target}}/confirm/{{.ID}}">https://{{.Hostname}}/v1/users/{{.Target}}/confirm/{{.ID}}</a></p>

<p>Thanks!</p>

<p>Team Applatix</p>
</body>
</html>
`))

var ResetPasswordSubject = "Applatix Password Reset"
var ResetPasswordBody = template.Must(template.New("RestPassword").Parse(`
<html>
<body style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 1.2em">

<h1>Password Reset!</h1>

<p>Click this link to reset your password:</p>
<p><a href="https://{{.Hostname}}/reset-password/{{.ID}};username={{.TargetBase64}}">https://{{.Hostname}}/reset-password/{{.ID}};username={{.TargetBase64}}</a></p>

<p>Thanks!</p>

<p>Team Applatix</p>
</body>
</html>
`))

var UserOnboardSubject = "Welcome to Applatix"
var UserOnboardBody = template.Must(template.New("UserOnboard").Parse(`
<html>
<body style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 1.2em">

<h1>Welcome to Applatix!</h1>

<p>Click this link to register your account:</p>
<p><a href="{{.SignupURL}}">{{.SignupURL}}</a></p>

<p>Thanks!</p>

<p>Team Applatix</p>
</body>
</html>
`))

var SandboxUserOnboardBody = template.Must(template.New("SandboxUserOnboard").Parse(`
<html>
<body style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 1.2em">

<h1>Welcome to Applatix!</h1>

<p>Build and run containerized apps NOW in the Applatix Playground - an interactive demo environment.</p>
<p>
<!--[if mso]>
  <v:roundrect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{{.SignupURL}}" style="height:40px;v-text-anchor:middle;width:300px;" arcsize="125%" strokecolor="#00BDCE" fillcolor="#7fdee6">
    <w:anchorlock/>
    <center style="color:#333;font-family:arial;font-size:14px;font-weight:bold;">Get Started with Applatix Playground</center>
  </v:roundrect>
<![endif]--><a href="{{.SignupURL}}" style="background-color:#7fdee6;border:1px solid #00BDCE;border-radius:50px;color:#333;display:inline-block;font-family:arial;font-size:14px;font-weight:bold;line-height:40px;text-align:center;text-decoration:none;width:300px;-webkit-text-size-adjust:none;mso-hide:all;">Get Started with Applatix Playground</a></div>
</p>

<p>Thank you for your interest in Applatix. If you require help at any time, please contact us!</p>

<p>Team Applatix</p>
</body>
</html>
`))
