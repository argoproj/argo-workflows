package axnc

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"

	"fmt"
	"time"
)

const (
	AcknowledgedBy   = "acknowledged_by"
	AcknowledgedTime = "acknowledged_time"
	Channel          = "channel"
	Channels         = "channels"
	Cluster          = "cluster"
	Code             = "code"
	Codes            = "codes"
	Detail           = "detail"
	EventID          = "event_id"
	Facility         = "facility"
	Message          = "message"
	Name             = "name"
	Recipients       = "recipients"
	RuleID           = "rule_id"
	Severity         = "severity"
	Severities       = "severities"
	TraceID          = "trace_id"
	Timestamp        = "timestamp"

	CodeTableName    = "code"
	EventTableName   = "events"
	RuleTableName    = "rule"
	Enabled          = "enabled"
	CreateTime       = "create_time"
	LastModifiedTime = "last_modified_time"
)

type Rule struct {
	Channels         []string `json:"channels,omitempty"`
	Codes            []string `json:"codes,omitempty"`
	Name             string   `json:"name,omitempty"`
	Recipients       []string `json:"recipients,omitempty"`
	RuleID           string   `json:"rule_id,omitempty"`
	Severities       []string `json:"severities,omitempty"`
	Enabled          bool     `json:"enabled,omitempty"`
	CreateTime       int64    `json:"create_time,omitempty"`
	LastModifiedTime int64    `json:"last_modified_time,omitempty"`
}

type Event struct {
	EventID    string            `json:"event_id"`
	TraceID    string            `json:"trace_id,omitempty"`
	Code       string            `json:"code"`
	Message    string            `json:"message,omitempty"`
	Facility   string            `json:"facility,omitempty"`
	Cluster    string            `json:"cluster,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Severity   string            `json:"severity,omitempty"`
	Timestamp  int64             `json:"timestamp"`
	Recipients []string          `json:"recipients,omitempty"`
	Detail     map[string]string `json:"detail,omitempty"`
}

type EventDetail struct {
	EventID          string            `json:"event_id"`
	TraceID          string            `json:"trace_id,omitempty"`
	Code             string            `json:"code"`
	Message          string            `json:"message,omitempty"`
	Facility         string            `json:"facility,omitempty"`
	Cluster          string            `json:"cluster,omitempty"`
	Channel          string            `json:"channel,omitempty"`
	Severity         string            `json:"severity,omitempty"`
	Timestamp        int64             `json:"timestamp"`
	Recipients       []string          `json:"recipients,omitempty"`
	Detail           map[string]string `json:"detail,omitempty"`
	AcknowledgedBy   string            `json:"acknowledged_by,omitempty"`
	AcknowledgedTime int64             `json:"acknowledged_time,omitempty"`
}

var defaultRules = []map[string]interface{}{
	map[string]interface{}{
		"rule_id":            "00000000-0000-0000-0000-000000000000",
		"name":               "DEFAULT_RULE",
		"channels":           []string{"system", "configuration", "job", "deployment", "spending"},
		"severities":         []string{"critical", "warning"},
		"codes":              []string{},
		"recipients":         []string{"super_admin@group", "admin@group"},
		"enabled":            true,
		"create_time":        time.Now().Unix(),
		"last_modified_time": time.Now().Unix(),
	},
}

var CodeSchema = axdb.Table{
	AppName: axdb.AXDBAppAXNC,
	Name:    CodeTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		Channel:  {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		Code:     {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClustering},
		Message:  {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Severity: {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	},
}

var RuleSchema = axdb.Table{
	AppName: axdb.AXDBAppAXNC,
	Name:    RuleTableName,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		Channels:         {Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		Codes:            {Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		Name:             {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Recipients:       {Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		RuleID:           {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		Severities:       {Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexStrong},
		Enabled:          {Type: axdb.ColumnTypeBoolean, Index: axdb.ColumnIndexNone},
		CreateTime:       {Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		LastModifiedTime: {Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
	},
}

var EventSchema = axdb.Table{
	AppName: axdb.AXDBAppAXNC,
	Name:    EventTableName,
	Type:    axdb.TableTypeTimedKeyValue,
	Columns: map[string]axdb.Column{
		AcknowledgedBy:   {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		AcknowledgedTime: {Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		Channel:          {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		Cluster:          {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Code:             {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		Detail:           {Type: axdb.ColumnTypeMap, Index: axdb.ColumnIndexNone},
		EventID:          {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Facility:         {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		Message:          {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
		Recipients:       {Type: axdb.ColumnTypeSet, Index: axdb.ColumnIndexNone},
		Severity:         {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
		Timestamp:        {Type: axdb.ColumnTypeInteger, Index: axdb.ColumnIndexNone},
		TraceID:          {Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexStrong},
	},
	Configs: map[string]interface{}{
		"default_time_to_live": int64(axdb.OneYear),
	},
	UseSearch: true,
	ExcludedIndexColumns: map[string]bool{
		AcknowledgedBy:   true,
		AcknowledgedTime: true,
		Channel:          true,
		Cluster:          true,
		Code:             true,
		EventID:          true,
		Facility:         true,
		Recipients:       true,
		Severity:         true,
		TraceID:          true,
	},
}

func PopulateDefaultRules() *axerror.AXError {
	admin, _ := user.GetUserByName("admin@internal")
	if admin == nil || time.Now().Unix()-admin.Ctime <= int64(time.Hour.Seconds())*24 {
		for _, rule := range defaultRules {
			var results []map[string]interface{}
			axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, RuleTableName, map[string]interface{}{"rule_id": rule["rule_id"]}, &results)
			if axErr != nil {
				var message = fmt.Sprintf("Failed to query default rules (err: %v)", axErr)
				utils.ErrorLog.Print(message)
				return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
			}

			if len(results) == 0 {
				_, axErr = utils.Dbcl.Put(axdb.AXDBAppAXNC, RuleTableName, rule)
				if axErr != nil {
					var message = fmt.Sprintf("Failed to populate default rules (err: %v)", axErr)
					utils.ErrorLog.Print(message)
					return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
				}
			}
		}
	}
	return nil
}
