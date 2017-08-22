package dispatcher

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axnc"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"

	"gopkg.in/check.v1"

	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const (
	eventSkeletonFile = "/tmp/event-skeleton"
)

func (s *S) TestInitSkeleton(c *check.C) {
	var dsp = dispatcher{}

	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)
	dsp.eventSkeleton = make(map[string]eventSkeleton)

	var originalSkeleton = eventSkeleton{
		Code:     "job.failure",
		Message:  "Job failed",
		Channel:  "job",
		Severity: "warning",
	}

	content, _ := json.Marshal([]eventSkeleton{originalSkeleton})

	err := ioutil.WriteFile(eventSkeletonFile, content, 0644)
	if err != nil {
		c.Fatal("Unable to create event skeleton file for test")
	}

	axErr := dsp.initSkeleton(eventSkeletonFile)
	if axErr != nil {
		c.Fatalf("Failed to initialize event skeleton (err: %v)", axErr)
	}

	c.Assert(dsp.eventSkeleton["job.failure"], check.DeepEquals, originalSkeleton)

	utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.CodeTableName, nil)
	utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, axnc.CodeSchema)
	os.Remove(eventSkeletonFile)
}

func (s *S) TestLoadRules(c *check.C) {
	var dsp = dispatcher{}

	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)
	dsp.lock = &sync.Mutex{}

	var rule = map[string]interface{}{
		axnc.Channels:         []string{"job"},
		axnc.Codes:            []string{"job.failure", "job.success", "job.delay"},
		axnc.Name:             "",
		axnc.Recipients:       []string{"tester@email.com"},
		axnc.RuleID:           "00000000-0000-0000-0000-000000000000",
		axnc.Severities:       []string{"critical", "warning"},
		axnc.Enabled:          true,
		axnc.CreateTime:       time.Now().Unix(),
		axnc.LastModifiedTime: time.Now().Unix(),
	}

	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXNC, axnc.RuleTableName, rule)
	if axErr != nil {
		c.Fatal("Unable to create rule for test")
	}

	axErr = dsp.loadRules()
	if axErr != nil {
		c.Fatalf("Failed to load rules (err: %v)", axErr)
	}

	c.Assert(dsp.rules["job.success"], check.NotNil)
	c.Assert(dsp.rules["job.failure"], check.NotNil)
	c.Assert(dsp.rules["job.delay"], check.NotNil)
	c.Assert(common.StringInSlice("tester@email.com", dsp.rules["job.success"]), check.Equals, true)
	c.Assert(common.StringInSlice("tester@email.com", dsp.rules["job.failure"]), check.Equals, true)
	c.Assert(common.StringInSlice("tester@email.com", dsp.rules["job.delay"]), check.Equals, true)

	utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.RuleTableName, nil)
	utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, axnc.RuleSchema)
}

func (s *S) TestLoadGroups(c *check.C) {
	var dsp = dispatcher{}

	dsp.lock = &sync.Mutex{}

	axErr := dsp.loadGroups()
	if axErr != nil {
		c.Fatalf("Failed to load groups (err: %v)", axErr)
	}

	c.Assert(common.StringInSlice("admin@internal", dsp.groups[user.GroupSuperAdmin]), check.Equals, true)
	c.Assert(common.StringInSlice("tester@email.com", dsp.groups[user.GroupSuperAdmin]), check.Equals, true)
}

func (s *S) TestLoadPreferences(c *check.C) {
	var dsp = dispatcher{}

	dsp.lock = &sync.Mutex{}

	axErr := dsp.loadPreferences()
	if axErr != nil {
		c.Fatalf("Failed to load preferences (err: %v)", axErr)
	}

	c.Assert(dsp.preferences["admin@internal"], check.NotNil)
	c.Assert(dsp.preferences["admin@internal"][axnc.TopicEmail], check.Equals, true)
	c.Assert(dsp.preferences["admin@internal"][axnc.TopicSlack], check.Equals, true)
	c.Assert(dsp.preferences["tester@email.com"], check.NotNil)
	c.Assert(dsp.preferences["tester@email.com"][axnc.TopicEmail], check.Equals, true)
	c.Assert(dsp.preferences["tester@email.com"][axnc.TopicSlack], check.Equals, true)
}

func (s *S) TestCreateEvent(c *check.C) {
	var dsp = dispatcher{}
	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)
	dsp.lock = &sync.Mutex{}
	dsp.eventSkeleton = make(map[string]eventSkeleton)

	// Initialize event skeleton
	var originalSkeleton = eventSkeleton{
		Code:     "job.failure",
		Message:  "Job failed",
		Channel:  "job",
		Severity: "warning",
	}

	content, _ := json.Marshal([]eventSkeleton{originalSkeleton})
	err := ioutil.WriteFile(eventSkeletonFile, content, 0644)
	if err != nil {
		c.Fatal("Unable to create event skeleton file for test")
	}
	axErr := dsp.initSkeleton(eventSkeletonFile)
	if axErr != nil {
		c.Fatalf("Failed to initialize event skeleton (err: %v)", axErr)
	}

	// Load rules
	var rule = map[string]interface{}{
		axnc.Channels:         []string{"job"},
		axnc.Codes:            []string{"job.failure", "job.success", "job.delay"},
		axnc.Name:             "",
		axnc.Recipients:       []string{"tester@email.com"},
		axnc.RuleID:           "00000000-0000-0000-0000-000000000000",
		axnc.Severities:       []string{"critical", "warning"},
		axnc.Enabled:          true,
		axnc.CreateTime:       time.Now().Unix(),
		axnc.LastModifiedTime: time.Now().Unix(),
	}
	_, axErr = utils.Dbcl.Put(axdb.AXDBAppAXNC, axnc.RuleTableName, rule)
	if axErr != nil {
		c.Fatal("Unable to create rule for test")
	}
	axErr = dsp.loadRules()
	if axErr != nil {
		c.Fatalf("Failed to load rules (err: %v)", axErr)
	}

	// Load groups
	axErr = dsp.loadGroups()
	if axErr != nil {
		c.Fatalf("Failed to load groups (err: %v)", axErr)
	}

	// Load preferences
	axErr = dsp.loadPreferences()
	if axErr != nil {
		c.Fatalf("Failed to load preferences (err: %v)", axErr)
	}

	eventID := common.GenerateUUIDv1()
	event := Event{
		EventID:   eventID,
		Code:      "job.failure",
		Facility:  "axsys.axops",
		Timestamp: time.Now().Unix(),
		Detail:    map[string]string{},
	}
	processedEvent, axErr := dsp.createEvent(&event)
	if axErr != nil {
		c.Errorf("Failed to create event (err: %v)", axErr)
		c.FailNow()
	}

	c.Assert(processedEvent.Code, check.Equals, "job.failure")
	c.Assert(processedEvent.Channel, check.Equals, "job")
	c.Assert(processedEvent.Severity, check.Equals, "warning")
	c.Assert(processedEvent.Facility, check.Equals, "axsys.axops")
	c.Assert(processedEvent.Message, check.Equals, "Job failed")
	c.Assert(processedEvent.TraceID, check.Equals, eventID)
	c.Assert(common.StringInSlice("tester@email.com", processedEvent.Recipients), check.Equals, true)

	utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.CodeTableName, nil)
	utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, axnc.CodeSchema)
	os.Remove(eventSkeletonFile)

	utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.RuleTableName, nil)
	utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, axnc.RuleSchema)
}

func (s *S) TestRateLimit(c *check.C) {
	var dsp = dispatcher{}
	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 30*time.Minute)
	dsp.lock = &sync.Mutex{}
	dsp.eventSkeleton = make(map[string]eventSkeleton)

	// Initialize event skeleton
	var noLimit = eventSkeleton{
		Code:     "test.nolimit",
		Message:  "throttling test",
		Channel:  "test",
		Severity: "warning",
		NoLimit:  true,
	}

	var limit = eventSkeleton{
		Code:     "test.limit",
		Message:  "throttling test",
		Channel:  "test",
		Severity: "warning",
		NoLimit:  false,
	}

	content, _ := json.Marshal([]eventSkeleton{limit, noLimit})
	err := ioutil.WriteFile(eventSkeletonFile, content, 0644)
	if err != nil {
		c.Fatal("Unable to create event skeleton file for test")
	}
	axErr := dsp.initSkeleton(eventSkeletonFile)
	if axErr != nil {
		c.Fatalf("Failed to initialize event skeleton (err: %v)", axErr)
	}

	var generateEvent = func(code string, ts int64) *Event {
		eventID := common.GenerateUUIDv1()
		event := Event{
			EventID:   eventID,
			Code:      code,
			Facility:  "test",
			Timestamp: ts,
			Detail:    map[string]string{},
		}
		return &event

	}

	tsNow := time.Now().UnixNano() / 1e3
	tsNowMinus10 := tsNow - int64(10*time.Minute/time.Microsecond)

	eventNoLimit := generateEvent(noLimit.Code, tsNowMinus10)

	c.Assert(dsp.rateLimitReached(eventNoLimit), check.Equals, false)
	dsp.updateRateLimit(eventNoLimit)
	c.Assert(dsp.rateLimitReached(eventNoLimit), check.Equals, false)

	eventLimit := generateEvent(limit.Code, tsNowMinus10)
	eventLimit10 := generateEvent(limit.Code, tsNow)

	c.Assert(dsp.rateLimitReached(eventLimit), check.Equals, false)
	dsp.updateRateLimit(eventLimit)
	c.Assert(dsp.rateLimitReached(eventLimit), check.Equals, true)
	c.Assert(dsp.rateLimitReached(eventLimit10), check.Equals, false)

	utils.Dbcl.Delete(axdb.AXDBAppAXNC, axnc.CodeTableName, nil)
	os.Remove(eventSkeletonFile)

}

func (s *S) TestGetUiRecipients(c *check.C) {
	var dsp = dispatcher{}
	dsp.lock = &sync.Mutex{}
	axErr := dsp.loadPreferences()
	if axErr != nil {
		c.Fatalf("Failed to load preferences (err: %v)", axErr)
	}

	var recipients = []string{"admin@internal", "prod@slack", "tester@email.com"}
	uiRecipients := dsp.getUiRecipients(recipients)
	c.Assert(common.StringInSlice("admin@internal", uiRecipients), check.Equals, true)
	c.Assert(common.StringInSlice("prod@slack", uiRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("tester@email.com", uiRecipients), check.Equals, true)
}

func (s *S) TestGetEmailRecipients(c *check.C) {
	var dsp = dispatcher{}
	dsp.lock = &sync.Mutex{}
	axErr := dsp.loadPreferences()
	if axErr != nil {
		c.Fatalf("Failed to load preferences (err: %v)", axErr)
	}

	var recipients = []string{"admin@internal", "ax-dev-internal", "prod@slack", "tester@email.com"}
	emailRecipients := dsp.getEmailRecipients(recipients)
	c.Assert(common.StringInSlice("admin@internal", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("ax-dev-internal", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("prod@slack", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("tester@email.com", emailRecipients), check.Equals, true)

	dsp.preferences["tester@email.com"][axnc.TopicEmail] = false
	emailRecipients = dsp.getEmailRecipients(recipients)
	c.Assert(common.StringInSlice("admin@internal", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("ax-dev-internal", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("prod@slack", emailRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("tester@email.com", emailRecipients), check.Equals, false)
}

func (s *S) TestSlackRecipients(c *check.C) {
	var dsp = dispatcher{}
	dsp.lock = &sync.Mutex{}
	axErr := dsp.loadPreferences()
	if axErr != nil {
		c.Fatalf("Failed to load preferences (err: %v)", axErr)
	}

	var recipients = []string{"admin@internal", "prod@slack", "tester@email.com"}
	slackRecipients := dsp.getSlackRecipients(recipients)
	c.Assert(common.StringInSlice("admin@internal", slackRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("prod@slack", slackRecipients), check.Equals, true)
	c.Assert(common.StringInSlice("tester@email.com", slackRecipients), check.Equals, true)

	dsp.preferences["tester@email.com"][axnc.TopicSlack] = false
	slackRecipients = dsp.getSlackRecipients(recipients)
	c.Assert(common.StringInSlice("admin@internal", slackRecipients), check.Equals, false)
	c.Assert(common.StringInSlice("prod@slack", slackRecipients), check.Equals, true)
	c.Assert(common.StringInSlice("tester@email.com", slackRecipients), check.Equals, false)
}
