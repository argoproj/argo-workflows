package handler

var emailTop = `<html>
<body>
<table class='email-container' style='font-size: 13px; color: #000000; font-family: Courier;'><tr><td class='event-detail' style='padding: 5px;'>
<table cellspacing='0' style='border: 0;'>
<tr>
<td>
<ul style='line-height: 22px; padding-left: 25px;'>
`

var emailMiddle = `</ul><table cellspacing='0' style='border: 1px solid #e3e3e3; margin-left: 28px;'>`

var emailBottom = `</table></ul></td></tr></table>
</td>
</tr>
<tr>
<td class='thank-you' style='padding-top: 20px;line-height: 22px;'>Thanks<br>Team Applatix</td>
</tr>
</table>
</body>
</html>
`

var emailBodyListTemplate = `<li><strong>%s: </strong>%s</li>`

var emailBodyTableTemplate = `<tr>
<td class='item-label' style='text-transform: capitalize; font-weight: bold; height: 15px; padding: 5px 10px 5px 10px; border-bottom: 1px solid #e3e3e3; border-right: 1px solid #e3e3e3;'>%s</td>
<td class='item-value' style='height: 15px; padding: 5px 10px 5px 10px; border-bottom: 1px solid #e3e3e3;'>%s</td>
</tr>`
