package axnc

const (
	RoleDispatcher       = "dispatcher"
	RoleUiHandler        = "ui_handler"
	RoleEmailHandler     = "email_handler"
	RoleSlackHandler     = "slack_handler"
	RoleAxSupportHandler = "ax_support_handler"

	NameGeneral   = "Dispatcher"
	NameUI        = "UI Handler"
	NameEmail     = "Email Handler"
	NameSlack     = "Slack Handler"
	NameAxSupport = "AX Support Handler"

	ConsumerGroupGeneral   = "axnc"
	ConsumerGroupUI        = "axnc.ui"
	ConsumerGroupEmail     = "axnc.email"
	ConsumerGroupSlack     = "axnc.slack"
	ConsumerGroupAxSupport = "axnc.ax_support"

	TopicUI        = "axnc.ui"
	TopicEmail     = "axnc.email"
	TopicSlack     = "axnc.slack"
	TopicAxSupport = "axnc.ax_support"
)
