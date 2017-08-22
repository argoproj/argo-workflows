package notification_center

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/notification_center"
	"time"
)

const (
	Ascending  = "asc"
	Descending = "desc"
)

type Rule axnc.Rule
type Event axnc.Event
type EventDetail axnc.EventDetail

func GetChannels() []string {
	return []string{
		notification_center.ChannelConfiguration,
		notification_center.ChannelDeployment,
		notification_center.ChannelJob,
		notification_center.ChannelSpending,
		notification_center.ChannelSystem,
	}
}

func GetSeverities() []string {
	return []string{
		notification_center.SeverityCritical,
		notification_center.SeverityWarning,
		notification_center.SeverityInfo,
	}
}

func GetCodes() ([]string, *axerror.AXError) {
	var codesRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.CodeTableName, nil, &codesRaw)
	if axErr != nil {
		return []string{}, axErr
	}

	var codes = []string{}
	for _, code := range codesRaw {
		codes = append(codes, code[axnc.Code].(string))
	}
	return codes, nil
}

func ListRules(params map[string]interface{}) ([]Rule, *axerror.AXError) {
	var rulesRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.RuleTableName, params, &rulesRaw)
	if axErr != nil {
		return []Rule{}, axErr
	}

	rules := []Rule{}
	for _, r := range rulesRaw {
		var rule = Rule{
			Channels:         []string{},
			Codes:            []string{},
			Name:             r[axnc.Name].(string),
			Recipients:       []string{},
			RuleID:           r[axnc.RuleID].(string),
			Severities:       []string{},
			Enabled:          r[axnc.Enabled].(bool),
			CreateTime:       int64(r[axnc.CreateTime].(float64)),
			LastModifiedTime: int64(r[axnc.LastModifiedTime].(float64)),
		}
		for _, channel := range r[axnc.Channels].([]interface{}) {
			rule.Channels = append(rule.Channels, channel.(string))
		}
		for _, code := range r[axnc.Codes].([]interface{}) {
			rule.Codes = append(rule.Codes, code.(string))
		}
		for _, recipient := range r[axnc.Recipients].([]interface{}) {
			rule.Recipients = append(rule.Recipients, recipient.(string))
		}
		for _, severity := range r[axnc.Severities].([]interface{}) {
			rule.Severities = append(rule.Severities, severity.(string))
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func GetRule(ruleID string) (*Rule, *axerror.AXError) {
	var params = map[string]interface{}{axnc.RuleID: ruleID}
	var rulesRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.RuleTableName, params, &rulesRaw)
	if axErr != nil {
		return nil, axErr
	}

	if len(rulesRaw) == 0 {
		return nil, nil
	}

	var rule = Rule{
		Channels:         []string{},
		Codes:            []string{},
		Name:             rulesRaw[0][axnc.Name].(string),
		Recipients:       []string{},
		RuleID:           rulesRaw[0][axnc.RuleID].(string),
		Severities:       []string{},
		Enabled:          rulesRaw[0][axnc.Enabled].(bool),
		CreateTime:       int64(rulesRaw[0][axnc.CreateTime].(float64)),
		LastModifiedTime: int64(rulesRaw[0][axnc.LastModifiedTime].(float64)),
	}
	for _, channel := range rulesRaw[0][axnc.Channels].([]interface{}) {
		rule.Channels = append(rule.Channels, channel.(string))
	}
	for _, code := range rulesRaw[0][axnc.Codes].([]interface{}) {
		rule.Codes = append(rule.Codes, code.(string))
	}
	for _, recipient := range rulesRaw[0][axnc.Recipients].([]interface{}) {
		rule.Recipients = append(rule.Recipients, recipient.(string))
	}
	for _, severity := range rulesRaw[0][axnc.Severities].([]interface{}) {
		rule.Severities = append(rule.Severities, severity.(string))
	}

	return &rule, nil
}

func UpdateRule(rule *Rule) *axerror.AXError {

	now := time.Now().Unix()
	if rule.RuleID == "" {
		rule.RuleID = common.GenerateUUIDv1()
	}

	var payload = map[string]interface{}{
		axnc.Channels:         rule.Channels,
		axnc.Codes:            rule.Codes,
		axnc.Name:             rule.Name,
		axnc.Recipients:       rule.Recipients,
		axnc.RuleID:           rule.RuleID,
		axnc.Severities:       rule.Severities,
		axnc.Enabled:          rule.Enabled,
		axnc.LastModifiedTime: now,
		axnc.CreateTime:       now,
	}

	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXNC, axnc.RuleTableName, payload)
	if axErr != nil {
		return axErr
	}

	return nil
}

func DeleteRule(ruleID string) *axerror.AXError {
	var params = []map[string]string{
		map[string]string{
			axnc.RuleID: ruleID,
		},
	}

	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.RuleTableName, params)
	if axErr != nil {
		return axErr
	}

	return nil
}

func UpdateEvent(event *EventDetail) *axerror.AXError {
	payload := dbEventFromEventDetail(event)
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXNC, axnc.EventTableName, payload)
	if axErr != nil {
		return axErr
	}
	return nil
}

func GetEvent(eventId string) (*EventDetail, *axerror.AXError) {
	var params = map[string]interface{}{}
	params[axnc.EventID] = eventId

	var eventsRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.EventTableName, params, &eventsRaw)
	if axErr != nil {
		return nil, axErr
	}

	if len(eventsRaw) == 0 {
		return nil, nil
	}

	e := eventsRaw[0]
	return eventDetailFromDBEvent(e), nil
}

func dbEventFromEventDetail(event *EventDetail) map[string]interface{} {
	var payload = map[string]interface{}{}
	payload[axnc.AcknowledgedBy] = event.AcknowledgedBy
	payload[axnc.AcknowledgedTime] = event.AcknowledgedTime
	payload[axnc.Channel] = event.Channel
	payload[axnc.Code] = event.Code
	payload[axnc.Detail] = event.Detail
	payload[axnc.EventID] = event.EventID
	payload[axnc.Facility] = event.Facility
	payload[axnc.Cluster] = event.Cluster
	payload[axnc.Message] = event.Message
	payload[axnc.Recipients] = event.Recipients
	payload[axnc.Severity] = event.Severity
	payload[axnc.Timestamp] = event.Timestamp
	payload[axdb.AXDBTimeColumnName] = event.Timestamp
	payload[axnc.TraceID] = event.TraceID
	return payload
}

func eventDetailFromDBEvent(e map[string]interface{}) *EventDetail {
	var event = EventDetail{
		EventID:    e[axnc.EventID].(string),
		TraceID:    e[axnc.TraceID].(string),
		Code:       e[axnc.Code].(string),
		Message:    e[axnc.Message].(string),
		Facility:   e[axnc.Facility].(string),
		Cluster:    e[axnc.Cluster].(string),
		Channel:    e[axnc.Channel].(string),
		Severity:   e[axnc.Severity].(string),
		Timestamp:  int64(e[axnc.Timestamp].(float64)),
		Recipients: []string{},
		Detail:     map[string]string{},
	}

	if ab, exists := e[axnc.AcknowledgedBy]; exists {
		event.AcknowledgedBy = ab.(string)
	}

	if at, exists := e[axnc.AcknowledgedTime]; exists {
		event.AcknowledgedTime = int64(at.(float64))
	}

	for _, recipient := range e[axnc.Recipients].([]interface{}) {
		event.Recipients = append(event.Recipients, recipient.(string))
	}
	if e[axnc.Detail] != nil {
		for k, v := range e[axnc.Detail].(map[string]interface{}) {
			event.Detail[k] = v.(string)
		}
	}
	return &event
}

func GetEvents(channel, facility, severity, traceID, recipient, ordering, orderBy string, minTime, maxTime, limit, offset int64) ([]*EventDetail, *axerror.AXError) {
	var params = map[string]interface{}{}
	if channel != "" {
		params[axnc.Channel] = channel
	}
	if facility != "" {
		params[axnc.Facility] = facility
	}
	if severity != "" {
		params[axnc.Severity] = severity
	}
	if traceID != "" {
		params[axnc.TraceID] = traceID
	}
	if recipient != "" {
		params[axnc.Recipients] = recipient
	}

	if minTime <= 0 {
		minTime = 0
	}
	if maxTime <= 0 {
		maxTime = time.Now().Unix() * 1e6
	}

	params[axdb.AXDBQueryMinTime] = minTime
	params[axdb.AXDBQueryMaxTime] = maxTime

	if limit > 0 {
		params[axdb.AXDBQueryMaxEntries] = limit
	}

	if offset > 0 {
		params[axdb.AXDBQueryOffsetEntries] = offset
	}

	var eventsRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.EventTableName, params, &eventsRaw)
	if axErr != nil {
		return nil, axErr
	}

	var events = []*EventDetail{}
	for _, e := range eventsRaw {
		event := eventDetailFromDBEvent(e)
		events = append(events, event)
	}

	return events, nil
}
