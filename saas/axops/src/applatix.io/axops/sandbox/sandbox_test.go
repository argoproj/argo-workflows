package sandbox_test

import (
	"os"
	"strconv"
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/event"
	"applatix.io/axops/sandbox"
	"applatix.io/axops/service"
	"applatix.io/axops/usage"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/template"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func createUser(c *check.C, group string) *user.User {
	u := &user.User{
		Username:    group + test.RandStr() + "@company.com",
		FirstName:   group,
		LastName:    " ",
		AuthSchemes: []string{"native"},
		Password:    "Test@test100",
		Groups:      []string{group},
		State:       user.UserStateActive,
	}
	ur, err := u.Create()
	c.Assert(err, check.IsNil)
	return ur
}

func login(c *check.C, u *user.User) map[string]interface{} {
	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": u.Username,
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)
	return map[string]interface{}{
		"session": session,
	}
}

func workflowTemplate(c *check.C) service.EmbeddedTemplateIf {
	workflowTemplateStr, axErr := utils.ReadFromFile("../testdata/template/workflow.json")
	c.Assert(axErr, check.IsNil)

	workflowTemplate, axErr := service.UnmarshalEmbeddedTemplate([]byte(workflowTemplateStr))
	c.Assert(axErr, check.IsNil)
	return workflowTemplate
}

func cancelAJob(c *check.C, jobId string, params map[string]interface{}) *axerror.AXError {
	axErr, _ := axopsExternalClient.Delete2("services/"+jobId, params, nil, nil)
	return axErr
}

func submitAJob(c *check.C, u *user.User, workflowTemplate service.EmbeddedTemplateIf, params map[string]interface{}) *axerror.AXError {
	commit := "fake-commit"
	branch := "fake-branch"
	repo := "fake-repo"

	s := service.Service{}
	s.Template = workflowTemplate
	s.User = u.Username
	s.Arguments = make(template.Arguments)
	s.Arguments["session.commit"] = &commit
	s.Arguments["session.branch"] = &branch
	s.Arguments["session.repo"] = &repo
	axErr, _ := axopsExternalClient.Post2("services", params, s, nil)
	return axErr
}

func getServiceList(c *check.C, params map[string]interface{}) []service.Service {

	sl := make(map[string][]service.Service)
	axErr := axopsExternalClient.Get("services", params, &sl)
	c.Assert(axErr, check.IsNil)
	return sl["data"]
}

func getServiceDetail(c *check.C, jobId string, params map[string]interface{}) service.Service {

	var s service.Service
	axErr := axopsExternalClient.Get("services/"+jobId, params, &s)
	c.Assert(axErr, check.IsNil)
	c.Assert(s, check.NotNil)
	return s
}

func getUserList(c *check.C, params map[string]interface{}) []user.User {

	ul := make(map[string][]user.User)
	axErr := axopsExternalClient.Get("users", params, &ul)
	c.Assert(axErr, check.IsNil)
	return ul["data"]
}

func createUsage(c *check.C, u *user.User) {
	containerUsage := usage.ContainerUsage{
		CostId: map[string]string{
			"user": u.Username,
			"Proj": "a",
		},
		HostId:         "aaaa",
		ContainerId:    "bbbb",
		ContainerName:  "container-bbbb",
		CPU:            10.0,
		CPUUsed:        10.0,
		CPUTotal:       10.0,
		CPUPercent:     10.0,
		Mem:            10.0,
		MemPercent:     10.0,
		CPURequest:     10.0,
		CPURequestUsed: 10.0,
		MemRequest:     10.0,
	}

	PostOneEvent(c, event.TopicContainerUsage, containerUsage.ContainerId, "blahblah", containerUsage)
	time.Sleep(2 * time.Second)

	var containerUsages []usage.ContainerUsage
	axErr := axdbClient.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableContainerUsage, nil, &containerUsages)
	c.Assert(axErr, check.IsNil)
	c.Assert(len(containerUsages), check.Equals, 1)

}

func assertUserList(c *check.C, params map[string]interface{}, emptyUsername bool) {
	ul := getUserList(c, params)
	c.Assert(len(ul) > 0, check.Equals, true)
	for _, u := range ul {
		if emptyUsername {
			c.Assert(u.Username, check.Equals, "")
		} else {
			c.Assert(u.Username == "", check.Equals, false)
		}
	}
}

func assertUsage(c *check.C, params map[string]interface{}, hideEmail bool) {
	usgl := make(map[string][]map[string]interface{})
	start := strconv.FormatInt(time.Now().Unix()-86400, 10)
	end := strconv.FormatInt(time.Now().Unix()+86400, 10)
	axErr := axopsExternalClient.Get("spendings/detail/"+start+"/"+end, params, &usgl)
	c.Assert(axErr, check.IsNil)
	c.Logf("usage:%v", usgl)
	c.Assert(len(usgl["data"]) > 0, check.Equals, true)
	for _, usg := range usgl["data"] {
		costId := usg["cost_id"].(map[string]string)
		if hideEmail {
			c.Assert(strings.Contains(costId["user"], "@"), check.Equals, false)
		} else {
			c.Assert(strings.Contains(costId["user"], "@"), check.Equals, true)
		}
	}
}

func assertServiceList(c *check.C, params map[string]interface{}, username string) {
	sl := getServiceList(c, params)
	c.Assert(len(sl) > 0, check.Equals, true)
	for _, s := range sl {
		c.Assert(s.User == username, check.Equals, true)
	}
}

func assertServiceDetail(c *check.C, params map[string]interface{}, username string) {
	sl := getServiceList(c, params)
	c.Assert(len(sl) > 0, check.Equals, true)
	for _, s := range sl {
		sd := getServiceDetail(c, s.TaskId, params)
		c.Assert(sd.User, check.Equals, username)
	}
}

func (s *S) TestSandbox(c *check.C) {
	c.Log("TestSandboxEnabled")
	// create 1 developer and 1 admin
	dev := createUser(c, user.GroupDeveloper)
	devParams := login(c, dev)
	admin := createUser(c, user.GroupAdmin)
	adminParams := login(c, admin)

	// before sanbox is enabled, user list should show email
	assertUserList(c, devParams, false)
	assertUserList(c, adminParams, false)

	// before sandbox is enabled, service list and detail should show email
	// first submit a dev job
	workflowTemplate := workflowTemplate(c)
	c.Assert(submitAJob(c, dev, workflowTemplate, devParams), check.IsNil)
	assertServiceList(c, devParams, dev.Username)
	assertServiceDetail(c, devParams, dev.Username)

	// before sandbox is enabled, usage should show email
	//createUsage(c, dev)
	//assertUsage(c,devParams,false)
	//assertUsage(c,adminParams,false)

	// enable sandbox mode
	os.Setenv("SANDBOX_ENABLED", "true")
	sandbox.InitSandbox()
	defer sandbox.ResetSandbox()
	time.Sleep(2 * time.Second)

	// after sandbox is enabled, service list and detail should show first name
	assertServiceList(c, devParams, dev.FirstName)
	assertServiceDetail(c, devParams, dev.FirstName)

	// after sandbox is enabled, user list should hide Username when queried by a developer
	assertUserList(c, devParams, true)
	assertUserList(c, adminParams, false)

	// after sandbox is enabled, usage should hide Username when queried by a developer
	//assertUsage(c,devParams,true)
	//assertUsage(c,adminParams,false)

	// max 10 conncurrent jobs can be submitted by a developer
	for i := 0; i < 9; i++ {
		c.Assert(submitAJob(c, dev, workflowTemplate, devParams), check.IsNil)
	}
	c.Assert(submitAJob(c, dev, workflowTemplate, devParams), check.NotNil)

	/*TODO: developer cannot cancel other people's jobs but can cancel it's own jobs. admin can cancel anyone's jobs
	dev2 := createUser(c,user.GroupDeveloper)
	dev2Params := login(c,dev2)
	sl := getServiceList(c, devParams)
	//c.Assert(cancelAJob(c,sl[0].Id,devParams),check.IsNil)
	c.Assert(cancelAJob(c,sl[1].Id,dev2Params),check.NotNil)
	c.Assert(cancelAJob(c,sl[1].Id,adminParams),check.IsNil)
	*/

	// TODO: jobs running for more than 1 hr should be cancelled

}
