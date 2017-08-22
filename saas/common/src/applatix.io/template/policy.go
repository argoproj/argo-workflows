package template

import (
	"strings"
	"time"

	"applatix.io/axerror"
	"github.com/robfig/cron"
)

// Policy Events
const (
	EventOnPush        = "on_push"
	EventOnPullRequest = "on_pull_request"
	EventOnPullMerge   = "on_pull_request_merge"
	EventOnCron        = "on_cron"
	EventOnTag         = "on_tag"
)

var policyEvents = []string{EventOnPush, EventOnPullRequest, EventOnPullMerge, EventOnCron, EventOnTag}

// Notification Events
const (
	//Notify when task is started
	NotificationOnStart = "on_start"
	//Notify when task is successful
	NotificationOnSuccess = "on_success"
	//Notify when task is failed
	NotificationOnFailure = "on_failure"
	//Notify when task status is changed, eg. task is failed for the first time, task is not failing any more
	NotificationOnChange = "on_change"
	//Notify when task completes (success, failure, cancelled, skipped)
	NotificationOnCompletion = "on_completion"
)

var notificationEvents = []string{NotificationOnStart, NotificationOnSuccess, NotificationOnFailure, NotificationOnChange, NotificationOnCompletion}
var notificationEventMap = map[string]bool{}

func init() {
	for _, n := range notificationEvents {
		notificationEventMap[n] = true
	}
}

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type PolicyTemplate struct {
	BaseTemplate  `json:",squash"`
	TemplateRef   `json:",squash"`
	Notifications []Notification `json:"notifications,omitempty"`
	When          []When         `json:"when,omitempty"`
}

type Notification struct {
	Whom []string `json:"whom,omitempty"`
	When []string `json:"when,omitempty"`
}

type When struct {
	Event    string  `json:"event,omitempty"`
	Schedule *string `json:"schedule,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

func (tmpl *PolicyTemplate) GetInputs() *Inputs {
	return nil
}

func (tmpl *PolicyTemplate) GetOutputs() *Outputs {
	return nil
}

func (tmpl *PolicyTemplate) Validate(preproc ...bool) *axerror.AXError {
	if axErr := tmpl.BaseTemplate.Validate(); axErr != nil {
		return axErr
	}
	axErr := tmpl.TemplateRef.Validate()
	if axErr != nil {
		return axErr
	}
	if len(tmpl.When) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Policy does not have any events in 'when'")
	}

	for _, when := range tmpl.When {
		switch when.Event {
		case EventOnPullMerge, EventOnPullRequest, EventOnPush, EventOnTag:
			continue
		case EventOnCron:
			if when.Schedule == nil || *when.Schedule == "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Policy 'schedule' is required for 'on_cron' events")
			}
			schedule := strings.TrimSpace(*when.Schedule)
			_, err := cron.Parse(schedule)
			if err != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid 'on_cron' schedule %v: %v", schedule, err)
			}
			when.Schedule = &schedule

			if when.Timezone != nil {
				timezone := strings.TrimSpace(*when.Timezone)
				if timezone != "" {
					_, err := time.LoadLocation(timezone)
					if err != nil {
						return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid 'on_cron' timezone %v: %v", timezone, err)
					}
				} else {
					timezone = "UTC"
				}
				when.Timezone = &timezone
			} else {
				timezone := "UTC"
				when.Timezone = &timezone
			}
		default:
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid event type '%s'. Valid options: %s", when.Event, strings.Join(policyEvents, ", "))
		}
	}

	for _, n := range tmpl.Notifications {
		for _, w := range n.When {
			if _, ok := notificationEventMap[w]; !ok {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid notification event '%s': Valid options: %s", w, notificationEvents)
			}
		}
	}

	if len(tmpl.Template) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Policy template is empty")
	}
	return nil
}

func (tmpl *PolicyTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	result, exists := context.Results[tmpl.Template]
	if !exists {
		// template should exist
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("template %s does not exist", tmpl.Template)
	}
	if result.AXErr != nil {
		// template should be valid
		return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("template %s is not valid", tmpl.Template)
	}
	st := result.Template
	switch st.GetType() {
	case TemplateTypeWorkflow, TemplateTypeContainer:
		break
	default:
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("template %s must be of type container or workflow", tmpl.Template)
	}
	// for policies, all reciever params need to be resolved since policies are triggered automatically
	// from events, with no opportunity to supply inputs.
	emptyScope := make(paramMap)
	unresolved, axErr := validateReceiverParamsPartial(st.GetName(), st.GetInputs(), tmpl.Arguments, emptyScope)
	if axErr != nil {
		return axErr
	}
	// The following special check needs to be made for %%session.commit%%, and %%session.repo%%
	// because validateReceiverParamsPartial does not consider parameters with those as defaults
	// as resolved. With policies, the only acceptable "unresolved" variables should be ones who
	// have them as defaults.
	for rcvrInputName, param := range unresolved {
		if param.defaultVal != nil {
			if *param.defaultVal == "%%session.commit%%" || *param.defaultVal == "%%session.repo%%" {
				continue
			}
		}
		return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("%s.%s parameter was not satisfied by caller", st.GetName(), rcvrInputName)
	}

	return nil
}
