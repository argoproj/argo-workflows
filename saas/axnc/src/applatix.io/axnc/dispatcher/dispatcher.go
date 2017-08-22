package dispatcher

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/kafkacl"
	"applatix.io/notification_center"
	"applatix.io/retry"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

var substitutableParameters = []string{
	"%%AX_CLUSTER_NAME_ID%%",
	"%%AXOPS_EXT_DNS%%",
}

var eventCodeLastUsedMap = map[string]int64{}

type eventSkeleton struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Channel  string `json:"channel"`
	Severity string `json:"severity"`
	NoLimit  bool   `json:"no_limit"`
}

type Event axnc.Event

type dispatcher struct {
	eventSkeleton map[string]eventSkeleton

	lock *sync.Mutex

	rules       map[string][]string
	groups      map[string][]string
	preferences map[string]map[string]bool

	KafkaConsumer *kafkacl.EventConsumer
	KafkaProducer *kafkacl.EventProducer
}

func NewDispatcher(axdbAddr, kafkaAddr, eventSkeletonFile string) (*dispatcher, *axerror.AXError) {
	var producerConfig = sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Return.Successes = true
	kafkaProducer, axErr := kafkacl.NewEventProducer(axnc.NameGeneral, kafkaAddr, producerConfig)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event producer (err: %v)", axErr)
		return nil, axErr
	}
	var consumerConfig = cluster.NewConfig()
	consumerConfig.Consumer.Fetch.Max = 100
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Return.Errors = true
	kafkaConsumer, axErr := kafkacl.NewEventConsumer(axnc.NameGeneral, kafkaAddr, axnc.ConsumerGroupGeneral, notification_center.TopicAxnc, consumerConfig)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to create event consumer (err: %v)", axErr)
		return nil, axErr
	}

	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)

	var dispatcher = &dispatcher{
		eventSkeleton: make(map[string]eventSkeleton),
		lock:          &sync.Mutex{},
		rules:         make(map[string][]string),
		preferences:   make(map[string]map[string]bool),
		KafkaConsumer: kafkaConsumer,
		KafkaProducer: kafkaProducer,
	}

	// Initialize event skeleton
	common.InfoLog.Printf("Initializing event skeleton (file: %s) ...", eventSkeletonFile)
	axErr = dispatcher.initSkeleton(eventSkeletonFile)
	if axErr != nil {
		common.ErrorLog.Printf("Failed to initialize event skeleton (err: %v)", axErr)
		return nil, axErr
	}
	common.InfoLog.Print("Successfully initialized event skeleton")

	// Initial loading of metadata
	common.InfoLog.Print("Initializing metadata ...")
	axErr = dispatcher.loadMetaData()
	if axErr != nil {
		common.ErrorLog.Printf("Failed to initialize metadata (err: %v)", axErr)
		return nil, axErr
	}
	common.InfoLog.Print("Successfully initialized metadata")

	return dispatcher, nil
}

func (dsp *dispatcher) initSkeleton(eventSkeletonFile string) *axerror.AXError {
	var eventSkeletons []eventSkeleton

	data, err := ioutil.ReadFile(eventSkeletonFile)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to initialize event codes (err: %v)", err))
	}

	err = json.Unmarshal(data, &eventSkeletons)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to initialize event codes (err: %v)", err))
	}

	var successCount = 0
	for _, eventSkeleton := range eventSkeletons {
		var payload = map[string]string{
			axnc.Code:     eventSkeleton.Code,
			axnc.Message:  eventSkeleton.Message,
			axnc.Channel:  eventSkeleton.Channel,
			axnc.Severity: eventSkeleton.Severity,
		}
		_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXNC, axnc.CodeTableName, payload)
		if axErr != nil {
			common.ErrorLog.Printf("Unable to write code (code: %s) into AXDB (err: %v)", eventSkeleton.Code, axErr)
			continue
		}
		dsp.eventSkeleton[eventSkeleton.Code] = eventSkeleton
		successCount++
	}

	return nil
}

func (dsp *dispatcher) loadRules() *axerror.AXError {
	var rulesRaw = []map[string]interface{}{}

	axErr := utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.RuleTableName, nil, &rulesRaw)
	if axErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to load rules (err: %v)", axErr))
	}

	var rulesMap = map[string]map[string]bool{}
	for _, rule := range rulesRaw {
		if rule[axnc.Recipients] == nil {
			continue
		}
		if (rule[axnc.Codes] == nil || len(rule[axnc.Codes].([]interface{})) == 0) &&
			(rule[axnc.Channels] == nil || len(rule[axnc.Channels].([]interface{})) == 0 || rule[axnc.Severities] == nil || len(rule[axnc.Severities].([]interface{})) == 0) {
			continue
		}

		if rule[axnc.Enabled] == false {
			continue
		}

		var codes = []string{}
		if rule[axnc.Codes] != nil && len(rule[axnc.Codes].([]interface{})) > 0 {
			// Ignore channels and severities
			for _, code := range rule[axnc.Codes].([]interface{}) {
				codes = append(codes, code.(string))
			}
		} else {
			// Populate codes from channels and severities
			channels := rule[axnc.Channels].([]interface{})
			severities := rule[axnc.Severities].([]interface{})
			for _, channel := range channels {
				for _, severity := range severities {
					var results []map[string]interface{}
					axErr = utils.Dbcl.Get(axdb.AXDBAppAXNC, axnc.CodeTableName, map[string]interface{}{axnc.Channel: channel, axnc.Severity: severity}, &results)
					if axErr != nil {
						var message = fmt.Sprintf("Failed to query codes (err: %v)", axErr)
						common.ErrorLog.Print(message)
						return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
					}
					for _, result := range results {
						codes = append(codes, result[axnc.Code].(string))
					}
				}
			}
		}

		var recipients = rule[axnc.Recipients].([]interface{})
		for _, code := range codes {
			if rulesMap[code] == nil {
				rulesMap[code] = map[string]bool{}
			}
			for _, recipient := range recipients {
				var r = recipient.(string)
				rulesMap[code][r] = true
			}
		}
	}

	var rules = map[string][]string{}
	for code, recipients := range rulesMap {
		if rules[code] == nil {
			rules[code] = []string{}
		}
		for recipient := range recipients {
			// Do not send anything to inactive users
			if dsp.preferences[recipient] == nil || dsp.preferences[recipient]["active"] == true {
				rules[code] = append(rules[code], recipient)
			}
		}
	}

	dsp.lock.Lock()
	dsp.rules = rules
	dsp.lock.Unlock()

	return nil
}

func (dsp *dispatcher) loadGroups() *axerror.AXError {
	groupsRaw, axErr := user.GetGroups(nil)
	if axErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to load groups (err: %v)", axErr))
	}

	usersRaw, axErr := user.GetUsers(nil)
	if axErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to load users (err: %v)", axErr))
	}

	var groupMap = map[string]map[string]bool{}

	for _, g := range groupsRaw {
		if _, ok := groupMap[g.Name]; !ok {
			groupMap[g.Name] = map[string]bool{}
		}
		for _, u := range g.Usernames {
			groupMap[g.Name][u] = true
		}
	}

	for _, u := range usersRaw {
		for _, g := range u.Groups {
			if _, ok := groupMap[g]; !ok {
				groupMap[g] = map[string]bool{}
			}
			groupMap[g][u.Username] = true
		}
	}

	var groups = map[string][]string{}
	for g, v := range groupMap {
		if _, ok := groups[g]; !ok {
			groups[g] = []string{}
		}
		for u := range v {
			groups[g] = append(groups[g], u)
		}
	}

	dsp.lock.Lock()
	dsp.groups = groups
	dsp.lock.Unlock()

	return nil
}

func (dsp *dispatcher) loadPreferences() *axerror.AXError {
	users, axErr := user.GetUsers(nil)
	if axErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Unable to load user preferences (err: %v)", axErr))
	}

	var preferences = map[string]map[string]bool{}
	for _, u := range users {
		if preferences[u.Username] == nil {
			preferences[u.Username] = map[string]bool{
				axnc.TopicEmail: true,
				axnc.TopicSlack: true,
				"active":        true,
			}
		}

		if u.State != user.UserStateActive {
			preferences[u.Username]["active"] = false
		}

		for _, topic := range []string{axnc.TopicEmail, axnc.TopicSlack} {
			if u.Settings != nil && u.Settings[topic] == "no" {
				preferences[u.Username][topic] = false
			}
		}
	}

	dsp.lock.Lock()
	dsp.preferences = preferences
	dsp.lock.Unlock()

	return nil
}

func (dsp *dispatcher) loadMetaData() *axerror.AXError {
	// Load groups
	axErr := dsp.loadGroups()
	if axErr != nil {
		common.ErrorLog.Printf("Failed to load user groups")
		return axErr
	}

	// Load preferences
	axErr = dsp.loadPreferences()
	if axErr != nil {
		common.ErrorLog.Printf("Failed to load user preferences")
		return axErr
	}

	// Load rules
	axErr = dsp.loadRules()
	if axErr != nil {
		common.ErrorLog.Printf("Failed to load rules")
		return axErr
	}

	return nil
}

func (dsp *dispatcher) RefreshMetaData(interval int) {
	for {
		dsp.loadMetaData()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (dsp *dispatcher) rateLimitReached(event *Event) bool {
	skeleton, ok := dsp.eventSkeleton[event.Code]
	if !ok {
		return true
	}
	if skeleton.NoLimit {
		return false
	}
	eventTSInMicros := event.Timestamp
	lastEventTSInMicros, exists := eventCodeLastUsedMap[event.Code]
	if !exists {
		return false
	}
	return (eventTSInMicros - lastEventTSInMicros) < int64(time.Minute*10/time.Microsecond) // rate limit to 1 per 10 mins
}

func (dsp *dispatcher) updateRateLimit(event *Event) {
	eventCodeLastUsedMap[event.Code] = event.Timestamp
}

func (dsp *dispatcher) createEvent(event *Event) (*Event, *axerror.AXError) {
	common.InfoLog.Printf("Searching event skeleton (id: %s, code: %s) ...", event.EventID, event.Code)

	skeleton, ok := dsp.eventSkeleton[event.Code]
	if !ok {
		common.ErrorLog.Printf("Unable to recognize event code (id: %s, code: %s)", event.EventID, event.Code)
		return nil, axerror.ERR_EVENT_INVALID.NewWithMessage(fmt.Sprintf("Unrecognizable event code (%s)", event.Code))
	}

	if event.TraceID == "" {
		event.TraceID = event.EventID
	}
	event.Message = skeleton.Message
	event.Channel = skeleton.Channel
	event.Cluster = os.Getenv("AX_CLUSTER_NAME_ID")
	event.Severity = skeleton.Severity

	// Locate applicable rules
	common.InfoLog.Printf("Searching applicable rules (code: %s) ...", event.Code)
	dsp.lock.Lock()
	var recipientsByRule = dsp.rules[event.Code]
	dsp.lock.Unlock()
	if recipientsByRule == nil || len(recipientsByRule) == 0 {
		common.WarnLog.Printf("Unable to find applicable rules (code: %s), skip", event.Code)
		return event, nil
	}

	// Determine recipients
	common.InfoLog.Printf("Determining recipients ...")
	var recipients = make(map[string]bool)
	for _, recipient := range append(event.Recipients, recipientsByRule...) {
		recipients[recipient] = true
	}

	// Resolve group into recipients
	common.InfoLog.Printf("Resolving user groups ...")
	for recipient := range recipients {
		if strings.HasSuffix(recipient, "@group") {
			var groupName = strings.TrimSuffix(recipient, "@group")

			dsp.lock.Lock()
			users := dsp.groups[groupName]
			dsp.lock.Unlock()

			if users != nil && len(users) > 0 {
				for _, u := range users {
					recipients[u] = true
				}
			}
			delete(recipients, recipient)
		}
	}

	event.Recipients = make([]string, 0)
	for recipient := range recipients {
		event.Recipients = append(event.Recipients, recipient)
	}

	// Substitute parameters
	for k, v := range event.Detail {
		var s = v
		for _, p := range substitutableParameters {
			if strings.Contains(s, p) {
				s = strings.Replace(s, p, os.Getenv(p[2:len(p)-2]), -1)
			}
		}
		event.Detail[k] = s
	}

	return event, nil
}

func (dsp *dispatcher) getUiRecipients(recipients []string) []string {
	var uiRecipients = make([]string, 0)

	for _, recipient := range recipients {
		if strings.HasSuffix(recipient, "@slack") {
			continue
		}

		dsp.lock.Lock()
		preference := dsp.preferences[recipient]
		dsp.lock.Unlock()

		if preference != nil {
			uiRecipients = append(uiRecipients, recipient)
		}
	}

	return uiRecipients
}

func (dsp *dispatcher) getEmailRecipients(recipients []string) []string {
	var emailRecipients = make([]string, 0)

	for _, recipient := range recipients {
		if strings.HasSuffix(recipient, "@slack") || !strings.Contains(recipient, "@") || strings.HasSuffix(recipient, "@internal") {
			continue
		}

		dsp.lock.Lock()
		preference := dsp.preferences[recipient]
		dsp.lock.Unlock()

		if preference == nil || preference[axnc.TopicEmail] {
			emailRecipients = append(emailRecipients, recipient)
		}
	}

	return emailRecipients
}

func (dsp *dispatcher) getSlackRecipients(recipients []string) []string {
	var slackRecipients = make([]string, 0)

	for _, recipient := range recipients {
		if strings.HasSuffix(recipient, "@slack") {
			slackRecipients = append(slackRecipients, recipient)
			continue
		}
		if !strings.Contains(recipient, "@") || strings.HasSuffix(recipient, "@internal") {
			continue
		}

		dsp.lock.Lock()
		preference := dsp.preferences[recipient]
		dsp.lock.Unlock()

		if preference != nil && preference[axnc.TopicSlack] {
			slackRecipients = append(slackRecipients, recipient)
		}
	}

	return slackRecipients
}

func (dsp *dispatcher) ProcessEvent(msg *sarama.ConsumerMessage) *axerror.AXError {
	var retryConfig = retry.NewRetryConfig(15*60, 1, 60, 2, nil)
	var event *Event = &Event{}

	err := json.Unmarshal(msg.Value, event)
	if err != nil {
		// When failed to unmarshal event, we simply drop the event
		common.ErrorLog.Printf("Failed to unmarshal event body (err: %v), skip", err)
		return nil
	}

	common.InfoLog.Printf("Preparing event payload (id: %s) ...", event.EventID)
	event, axErr := dsp.createEvent(event)
	if axErr != nil {
		// When failed to create event, we simply drop the event
		common.ErrorLog.Printf("Failed to prepare event payload (err: %v), skip", axErr)
		return nil
	}

	var topics []string
	// check rate limit for the event code
	if dsp.rateLimitReached(event) {
		common.InfoLog.Printf("Throttling event (id: %s, code: %s)\n", event.EventID, event.Code)
		topics = []string{axnc.TopicAxSupport}
	} else {
		topics = []string{axnc.TopicUI, axnc.TopicEmail, axnc.TopicSlack, axnc.TopicAxSupport}
	}

	dsp.updateRateLimit(event)

	var recipients = make([]string, len(event.Recipients))
	copy(recipients, event.Recipients)
	for _, topic := range topics {
		common.InfoLog.Printf("Deriving recipients (topic: %s) ...", topic)

		if topic != axnc.TopicAxSupport && (event.Code == notification_center.CodeConfigurationNotificationInvalidSmtp ||
			event.Code == notification_center.CodeConfigurationNotificationInvalidSlack) {
			continue
		}
		if topic == axnc.TopicAxSupport && event.Channel != notification_center.ChannelSystem {
			continue
		}

		switch topic {
		case axnc.TopicUI:
			event.Recipients = dsp.getUiRecipients(recipients)
		case axnc.TopicEmail:
			event.Recipients = dsp.getEmailRecipients(recipients)
		case axnc.TopicSlack:
			event.Recipients = dsp.getSlackRecipients(recipients)
		case axnc.TopicAxSupport:
			event.Recipients = []string{}
		default:
			continue
		}

		if (topic == axnc.TopicEmail || topic == axnc.TopicSlack) && len(event.Recipients) == 0 {
			continue
		}

		value, err := json.Marshal(event)
		if err != nil {
			// When failed to marshal event, we simply drop the event corresponding to that topic
			common.ErrorLog.Printf("Failed to marshal event (id: %s, topic: %s, err: %s)", event.EventID, topic, err)
			continue
		}

		retryConfig.Retry(
			func() *axerror.AXError {
				common.InfoLog.Printf("Sending event (id: %s, topic: %s) ...", event.EventID, topic)
				producerMsg := &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(value)}
				_, _, err = dsp.KafkaProducer.Producer.SendMessage(producerMsg)

				if err == nil {
					common.InfoLog.Printf("Successfully sent event (id: %s, topic: %s)", event.EventID, topic)
					return nil
				} else {
					var message = fmt.Sprintf("Failed to send event (id: %s, topic: %s, err: %v)", event.EventID, topic, err)
					common.DebugLog.Print(message)
					return axerror.ERR_AX_INTERNAL.NewWithMessage(message)
				}
			},
		)
	}

	return nil
}
