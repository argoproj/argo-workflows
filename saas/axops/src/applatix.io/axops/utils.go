// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"applatix.io/axamm/application"
	"applatix.io/axamm/deployment"
	axammutils "applatix.io/axamm/utils"
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axerror"
	"applatix.io/axnc"
	"applatix.io/axops/auth"
	"applatix.io/axops/auth/native"
	"applatix.io/axops/cluster"
	"applatix.io/axops/commit"
	"applatix.io/axops/configuration"
	"applatix.io/axops/container"
	"applatix.io/axops/custom_view"
	"applatix.io/axops/event"
	"applatix.io/axops/fixture"
	"applatix.io/axops/host"
	"applatix.io/axops/index"
	"applatix.io/axops/jira"
	"applatix.io/axops/label"
	"applatix.io/axops/policy"
	"applatix.io/axops/project"
	"applatix.io/axops/schema_devops"
	"applatix.io/axops/schema_internal"
	"applatix.io/axops/schema_platform"
	"applatix.io/axops/service"
	"applatix.io/axops/session"
	"applatix.io/axops/tool"
	"applatix.io/axops/usage"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/axops/volume"
	"applatix.io/axops/yaml"
	"applatix.io/common"
	"applatix.io/rediscl"
	"applatix.io/restcl"
	"github.com/gin-gonic/gin"
)

type MapType map[string]string

func Init(dbUrl, devopsUrl, workflowAdcUrl, axmonUrl, axnotifierUrl, fixmgrUrl, schedulerUrl string, artifactUrl string) {
	InitLoggers()

	if len(dbUrl) != 0 {
		Dbcl = axdbcl.NewAXDBClientWithTimeout(dbUrl, 5*time.Minute)
		utils.Dbcl = Dbcl
		axammutils.DbCl = Dbcl
	}

	if len(devopsUrl) != 0 {
		DevopsCl = restcl.NewRestClientWithTimeout(devopsUrl, 20*time.Minute)
		utils.DevopsCl = DevopsCl
	}

	if len(workflowAdcUrl) != 0 {
		WorkflowAdcCl = restcl.NewRestClientWithTimeout(workflowAdcUrl, 1*time.Minute)
		utils.WorkflowAdcCl = WorkflowAdcCl
	}

	if len(axmonUrl) != 0 {
		utils.AxmonCl = restcl.NewRestClientWithTimeout(axmonUrl, 5*time.Minute)
		axammutils.AxmonCl = utils.AxmonCl
	}

	if len(axnotifierUrl) != 0 {
		AxNotifierCl = restcl.NewRestClientWithTimeout(axnotifierUrl, 5*time.Minute)
		utils.AxNotifierCl = AxNotifierCl
		axammutils.AxNotifierCl = utils.AxNotifierCl
	}

	if len(fixmgrUrl) != 0 {
		utils.FixMgrCl = restcl.NewRestClientWithTimeout(fixmgrUrl, 5*time.Minute)
		axammutils.FixMgrCl = utils.FixMgrCl
	}

	if len(schedulerUrl) != 0 {
		utils.SchedulerCl = restcl.NewRestClientWithTimeout(schedulerUrl, 5*time.Minute)
	}

	if len(artifactUrl) != 0 {
		ArtifactCl = restcl.NewRestClientWithTimeout(artifactUrl, 5*time.Minute)
		utils.ArtifactCl = ArtifactCl
	}

	// Don't add the namespace axsys, otherwise the unit test might be connecting the redis in the axsys
	utils.RedisCacheCl = rediscl.NewRedisClient("redis:6379", "", utils.RedisCachingDatabase)
	utils.AxammCl = restcl.NewRestClientWithTimeout("http://axamm:8966/v1", 5*time.Minute)

	AX_NAMESPACE := common.GetAxNameSpace()
	AX_VERSION := common.GetAxVersion()

	if AX_NAMESPACE == "" {
		utils.ErrorLog.Printf("AX_NAMESAPCE is not available from environment variables. Abort.")
		os.Exit(1)
	}

	if AX_VERSION == "" {
		utils.ErrorLog.Printf("AX_VERSION is not available from environment variables. Abort.")
		os.Exit(1)
	}

	utils.InfoLog.Printf("axops AX_NAMESPACE:%s AX_VERSION:%s\n", AX_NAMESPACE, AX_VERSION)

	// Wait for axdb to be ready
	count := 0
	for {
		count++
		kvMap, dbErr := schema_internal.GetAppKeyValsByAppName(axdb.AXDBAppAXOPS, utils.Dbcl)
		if dbErr != nil {
			utils.InfoLog.Printf("waiting for axdb to be ready ...... count %v error %v", count, dbErr)
		} else {
			AX_NAMESPACE_DB, _ := kvMap["AX_NAMESPACE"]
			AX_VERSION_DB, _ := kvMap["AX_VERSION"]
			utils.InfoLog.Printf("axops schema AX_NAMESPACE:%s AX_VERSION:%s\n", AX_NAMESPACE_DB, AX_VERSION_DB)
			if AX_NAMESPACE_DB != AX_NAMESPACE || AX_VERSION_DB != AX_VERSION {
				utils.InfoLog.Printf("axops schema version from system and db don't match\n")
			} else {
				utils.InfoLog.Printf("axops schema version from system and db match\n")
				break
			}
		}

		if count > 300 {
			// Give up, marathon would restart this container
			utils.ErrorLog.Printf("axdb is not available, exited")
			os.Exit(1)
		}

		time.Sleep(1 * time.Second)
	}

	nativeScheme := &native.NativeScheme{&auth.BaseScheme{}}
	auth.RegisterScheme("native", nativeScheme)

	// resubmit all the jobs to adc
	go ResubmitServices()
}

func InitTest(dbUrl, devopsUrl, workflowAdcUrl, axmonUrl, axnotifierUrl, fixmgrUrl, schedulerUrl string) {
	InitLoggers()

	if len(dbUrl) != 0 {
		Dbcl = axdbcl.NewAXDBClientWithTimeout(dbUrl, 5*time.Minute)
		utils.Dbcl = Dbcl
	}

	if len(devopsUrl) != 0 {
		DevopsCl = restcl.NewRestClientWithTimeout(devopsUrl, 1*time.Minute)
		utils.DevopsCl = DevopsCl
	}

	if len(workflowAdcUrl) != 0 {
		WorkflowAdcCl = restcl.NewRestClientWithTimeout(workflowAdcUrl, 1*time.Minute)
		utils.WorkflowAdcCl = WorkflowAdcCl
	}

	if len(axmonUrl) != 0 {
		utils.AxmonCl = restcl.NewRestClientWithTimeout(axmonUrl, 5*time.Minute)
	}

	if len(axnotifierUrl) != 0 {
		AxNotifierCl = restcl.NewRestClientWithTimeout(axnotifierUrl, 5*time.Minute)
		utils.AxNotifierCl = AxNotifierCl
	}

	if len(fixmgrUrl) != 0 {
		utils.FixMgrCl = restcl.NewRestClientWithTimeout(fixmgrUrl, 5*time.Minute)
	}

	if len(schedulerUrl) != 0 {
		utils.SchedulerCl = restcl.NewRestClientWithTimeout(schedulerUrl, 5*time.Minute)
	}

	// Don't add the namespace axsys, otherwise the unit test might be connecting the redis in the axsys
	utils.RedisCacheCl = rediscl.NewRedisClient("redis:6379", "", utils.RedisCachingDatabase)

	// Wait for axdb to be ready
	count := 0
	for {
		count++
		var bodyArray []interface{}
		dbErr := Dbcl.Get("axdb", "version", nil, &bodyArray)
		if dbErr == nil {
			break
		} else {
			InfoLog.Printf("waiting for axdb to be ready ...... count %v error %v", count, dbErr)
			if count > 300 {
				// Give up, marathon would restart this container
				ErrorLog.Printf("axdb is not available, exited")
				os.Exit(1)
			}
		}
		time.Sleep(1 * time.Second)
	}

	nativeScheme := &native.NativeScheme{&auth.BaseScheme{}}
	auth.RegisterScheme("native", nativeScheme)
}

func queryParameter(c *gin.Context, name string) string {
	valueArray := c.Request.URL.Query()[name]
	if valueArray == nil {
		return ""
	}
	return valueArray[0]
}

// returns 0 on error. 0 is not a valid parameter. 0 also indicates that parameter is not set
func queryParameterInt(c *gin.Context, name string) int64 {
	valueArray := c.Request.URL.Query()[name]
	if valueArray == nil {
		return 0
	}

	value, err := strconv.ParseInt(valueArray[0], 10, 64)
	if err != nil {
		ErrorLog.Printf("expecting int64 got %v", valueArray[0])
		c.JSON(axdb.RestStatusInvalid, nullMap)
		return 0
	}
	return value
}

func getBodyString(c *gin.Context) ([]byte, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(c.Request.Body)
	if err != nil {
		return nil, err
	}
	body := buffer.Bytes()
	return body, nil
}

func GetContextParams(c *gin.Context, sFields []string, bFields []string, iFields []string, mField []string) (map[string]interface{}, *axerror.AXError) {
	return common.GetContextParams(c, sFields, bFields, iFields, mField)
}

func GetContextTimeParams(c *gin.Context, params map[string]interface{}) (map[string]interface{}, *axerror.AXError) {
	return common.GetContextTimeParams(c, params)
}

func getContextRawParams(c *gin.Context) (map[string]interface{}, *axerror.AXError) {
	params := make(map[string]interface{})

	for k, vs := range c.Request.URL.Query() {
		if len(vs) == 0 {
			params[k] = ""
		} else {
			params[k] = vs[0]
		}
	}

	return params, nil
}

func LoadCustomCert() {
	certs, axErr := tool.GetToolsByType(tool.TypeServer)
	if axErr != nil {
		utils.ErrorLog.Println("Failed to load the cert from DB:", axErr)
		return
	}

	var crt, key string
	if len(certs) == 0 {
		utils.InfoLog.Println("There is no cert in use. The server will start generating the self-signed certificate.")

		crt, key = utils.GenerateSelfSignedCert()

		base := &tool.ToolBase{
			Category: tool.CategoryCertificate,
			Type:     tool.TypeServer,
		}
		crtConfig := &tool.ServerCertConfig{base, crt, key}

		axErr, _ := tool.Create(crtConfig)
		if axErr != nil {
			panic(fmt.Sprintf("Failed to persisted the newly created self-signed certificate: %v", axErr))
		}

		utils.InfoLog.Println("Self-signed certificate is created successfully.")
	} else {
		crt = certs[0].(*tool.ServerCertConfig).PublicCert
		key = certs[0].(*tool.ServerCertConfig).PrivateKey
	}

	axErr = utils.WriteToFile(crt, utils.GetPublicCertPath())
	if axErr != nil {
		panic(fmt.Sprintln("Failed to write the public cert file:", axErr))
	}

	axErr = utils.WriteToFile(key, utils.GetPrivateKeyPath())
	if axErr != nil {
		panic(fmt.Sprintln("Failed to write the private key file:", axErr))
	}

	utils.InfoLog.Println("Certificate is loaded successfully.")
}

func CreateTable(table axdb.Table, c chan *axerror.AXError) {
	dbcl := utils.Dbcl.CopyClient()
	fmt.Printf("Starting updating table: %v.\n", table)
	startTime := time.Now().UnixNano() / 1e9
	var axErr *axerror.AXError
	for {
		currentTime := time.Now().UnixNano() / 1e9
		if currentTime-startTime > 900 {
			axErr = axerror.ERR_AX_TIMEOUT.NewWithMessage("Axops table creation failed due to timeout(15 minutes)")
		} else {
			_, axErr = dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
			if axErr == nil {
				fmt.Printf("Successfully updated table %v schema.\n", table.Name)
			} else {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		c <- axErr
		break
	}
}

var tables = []axdb.Table{
	// Always keep this one to be the first one
	schema_internal.AppSchema,

	schema_platform.ArtifactsSchema,
	schema_platform.ArtifactRetentionSchema,
	schema_platform.ArtifactMetaSchema,
	schema_platform.NodePortManagerSchema,

	schema_devops.WorkflowSchema,
	schema_devops.WorkflowLeafServiceSchema,
	schema_devops.WorkflowTimedEventsSchema,
	schema_devops.WorkflowKVSchema,
	schema_devops.WorkflowNodeEventsSchema,
	schema_devops.BranchSchema,
	schema_devops.ApprovalSchema,
	schema_devops.ApprovalResultSchema,
	schema_devops.JunitTestCaseResultSchema,
	schema_devops.ResourceSchema,

	application.GetLatestApplicationSchema(),
	application.GetHistoryApplicationSchema(),
	deployment.GetLatestDeploymentSchema(),
	deployment.GetHistoryDeploymentSchema(),

	axnc.CodeSchema,
	axnc.EventSchema,
	axnc.RuleSchema,

	service.GetRunningServiceSchema(),
	service.GetDoneServiceSchema(),
	service.TemplateSchema,
	policy.PolicySchema,
	tool.ToolSchema,
	commit.CommitSchema,
	fixture.FixtureTemplateSchema,
	fixture.FixtureClassSchema,
	fixture.FixtureInstanceSchema,
	fixture.FixtureRequestSchema,
	label.LabelSchema,
	user.UserSchema,
	user.SystemRequestSchema,
	user.GroupSchema,
	session.SessionSchema,
	auth.AuthRequestSchema,
	usage.ContainerUsageSchema,
	usage.HostUsageSchema,
	host.HostSchema,
	container.ContainerSchema,
	custom_view.CustomViewSchema,
	AuditTrailSchema,
	ProfileSchema,
	project.ProjectSchema,
	volume.StorageProviderSchema,
	volume.StorageClassSchema,
	volume.VolumeSchema,
	cluster.ClusterSettingSchema,
	index.SearchIndexSchema,
	jira.JiraSchema,
	configuration.ConfigurationSchema,
}

func CreateTables() *axerror.AXError {
	n := len(tables)
	batch := 1
	index := 0
	startTime := time.Now().UnixNano() / 1e9
	for index < n {
		currentTime := time.Now().UnixNano() / 1e9
		if currentTime-startTime > 900 {
			return axerror.ERR_AX_TIMEOUT.NewWithMessage("Axops table creation failed due to timeout(15 minutes)")
		}
		actualBatch := 0
		if index+batch >= n {
			actualBatch = n - index
		} else {
			actualBatch = batch
		}

		c := make(chan *axerror.AXError, 10*len(tables))
		for i := 0; i < actualBatch; i++ {
			go CreateTable(tables[i+index], c)
		}

		for i := 0; i < actualBatch; i++ {
			axErr := <-c
			if axErr != nil {
				fmt.Printf("got an error %d: %v", i, axErr)
				return axErr
			}
		}
		index += batch
	}

	axErr := loadUserLabels()
	if axErr != nil {
		return axErr
	}

	return nil
}

var userLabels = []string{label.UserLabelSubmitter, label.UserLabelAuthor, label.UserLabelSubmitter, label.UserLabelSCM, label.UserLabelFixtureManager}

func loadUserLabels() *axerror.AXError {
	for _, userLabel := range userLabels {
		lb := label.Label{
			Type:     label.LabelTypeUser,
			Key:      userLabel,
			Value:    "",
			Reserved: true,
			Ctime:    time.Now().UTC().Unix() * 1e6,
		}
		lb.ID = lb.GenerateID()

		axErr := lb.Update()
		if axErr != nil {
			fmt.Printf("failed to post user label, err: %v", axErr)
			return axErr
		}
	}
	fmt.Println("Loaded system user labels")
	return nil
}

func GCLables() {
	ticker := time.NewTicker(time.Hour * 12)
	go func() {
		for _ = range ticker.C {
			yaml.GarbageCollectLabels()
		}
	}()
}

var profileChan = make(chan *Profile, 200)

func ProfilerWorker() {
	for profile := range profileChan {
		utils.DebugLog.Printf("Profile buffer channel status: %v/%v\n", len(profileChan), cap(profileChan))
		username := profile.ApiCallerLogin
		profile.ApiCallerLogin = ""
		_, dbErr := Dbcl.Post(axopsPerfApp, ProfileTableName, profile)
		if dbErr != nil {
			utils.InfoLog.Printf("*** TEST: insert failure with error %v", dbErr)
		}

		profile.ApiCallerLogin = username
		_, dbErr = Dbcl.Post(axopsPerfApp, AuditTrailTableName, profile)
		if dbErr != nil {
			utils.InfoLog.Printf("*** TEST: insert failure with error %v", dbErr)
		}
	}
}

func SendToProfileChan(p *Profile) {
	if p == nil {
		return
	}

	select {
	case profileChan <- p:
	default:
		utils.DebugLog.Printf("Profile channel is full, data: %v\n", *p)
	}
}

func GCTemplatePolicyProjectFixture() {
	ticker := time.NewTicker(time.Hour * 24)
	go func() {
		for _ = range ticker.C {
			yaml.GarbageCollectTemplatePolicyProjectFixture()
		}
	}()
}

func RotateETagHourly() {

	// Rotate the ETAG hourly in case of some internal bugs
	secondsToHour := 3600 - time.Now().Unix()%3600
	utils.DebugLog.Printf("[ETAG]: sleep %v seconds to start etag rotation.\n", secondsToHour)
	time.Sleep(time.Duration(secondsToHour * int64(time.Second)))

	rotate := func() {
		service.UpdateServiceETag()
		service.UpdateTemplateETag()
		policy.UpdateETag()
		commit.UpdateETag()
		user.UpdateETag()
		UpdateSpendingETag()
		project.UpdateETag()
		fixture.UpdateETag()
		utils.DebugLog.Println("[ETAG]: etag rotated.")
	}

	rotate()

	ticker := time.NewTicker(time.Minute * 60)
	go func() {
		for _ = range ticker.C {
			rotate()
		}
	}()
}

func CreateDomainManagementTool() *axerror.AXError {
	if tools, axErr := tool.GetToolsByType(tool.TypeRoute53); axErr != nil {
		return axErr
	} else {
		if len(tools) != 0 {
			return nil
		} else {
			domain := &tool.DomainConfig{&tool.ToolBase{}, nil, nil}
			domain.Category = tool.CategoryDomainManagement
			domain.Type = tool.TypeRoute53
			axErr, _ := tool.Create(domain)
			if axErr != nil {
				return axErr
			}
		}
	}
	return nil
}

func RefreshScmToolScheduler() {

	// Rotate the ETAG hourly in case of some internal bugs
	secondsToHour := 1800 - time.Now().Unix()%1800
	utils.DebugLog.Printf("[SCM]: sleep %v seconds to start scm refreshment.\n", secondsToHour)
	time.Sleep(time.Duration(secondsToHour * int64(time.Second)))

	refresher := func() {
		UpdateScmTools()
		utils.DebugLog.Println("[SCM]: scm tool configurations updated.")
	}

	refresher()

	ticker := time.NewTicker(time.Minute * 30)
	go func() {
		for _ = range ticker.C {
			refresher()
		}
	}()
}

const (
	LimitSpendingExceed = "LimitSpendingExceed"
	LimitTimeExceed     = "LimitTimeExceed"
)

func ShouldTerminate(s *service.Service, cost float64, runTime float64) (bool, string) {
	if s == nil {
		return false, ""
	}

	if s.TerminationPolicy == nil {
		return false, ""
	}

	spendingLimitCents, err := strconv.ParseFloat(s.TerminationPolicy.SpendingCents, 64)
	if err != nil {
		utils.ErrorLog.Printf("[JobMonitor] Error parsing the spending limit in cents (%v): %v.\n", s.TerminationPolicy.SpendingCents, err)
	}

	timeLimitSeconds, err := strconv.ParseFloat(s.TerminationPolicy.TimeSeconds, 64)
	if err != nil {
		utils.ErrorLog.Printf("[JobMonitor] Error parsing the time limit in seconds (%v): %v.\n", s.TerminationPolicy.TimeSeconds, err)
	}

	utils.DebugLog.Printf("[JobMonitor] Limit(%v %v) Actual(%v, %v) Serivce(%v %v)", spendingLimitCents, timeLimitSeconds, cost, runTime, s.Name, s.Id)
	if spendingLimitCents > 0 && cost > spendingLimitCents {
		return true, LimitSpendingExceed
	}

	if timeLimitSeconds > 0 && runTime > timeLimitSeconds {
		return true, LimitTimeExceed
	}

	return false, ""
}

func SubmitJob(s *service.Service, once bool) *axerror.AXError {
	substitutedSvc, axErr := s.Preprocess()
	if axErr != nil {
		return axErr
	}

	data, _ := json.MarshalIndent(substitutedSvc, "", "    ")
	utils.InfoLog.Printf("[JOB] service object to be sent to workflow_adc:\n%s", string(data))

	// keep retrying until we succeed at submitting the job to adc
	for {
		axErr, code := utils.WorkflowAdcCl.Post2("workflows", nil, &substitutedSvc, nil)
		if axErr != nil {
			utils.ErrorLog.Printf("[JOB] Failed to submit job %v/%v, (%v)error: %v", s.Id, s.Name, code, axErr)
			if code >= 500 {
				if !once {
					time.Sleep(5 * time.Second)
					continue
				}
			} else if code >= 400 {

				detail := map[string]interface{}{
					"code":    axErr.Code,
					"message": axErr.Message,
					"detail":  axErr.Detail,
				}

				if service.ValidateStateTransition(s.Status, utils.ServiceStatusFailed) {
					return service.HandleServiceUpdate(s.Id, utils.ServiceStatusFailed, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
				}

				return nil
			}

		} else {
			return s.MarkSubmitted()
		}
		return axErr
	}
}
