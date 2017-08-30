package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/commit"
	"applatix.io/axops/container"
	"applatix.io/axops/host"
	"applatix.io/axops/label"
	"applatix.io/axops/notification"
	"applatix.io/axops/policy"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/notification_center"
	"applatix.io/restcl"
	"applatix.io/template"
	"github.com/Shopify/sarama"
	"github.com/gocql/gocql"
)

type ServiceContext struct {
	User             *user.User
	Root             *Service
	IsPartial        bool
	DependsNewCommit bool
	Arguments        template.Arguments // The arguments the user provided at the time of launching the service.
	services         ServiceMap         // for re-submit partial run job
	serviceDict      map[string]string  // the mapping of old service id to new service id
}

type ServiceStatusEvent struct {
	TaskId string `json:"task_id,omitempty"`
	Id     string `json:"id,omitempty"`
	Status int    `json:"status"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// Handler for iterator function. It will pass in a valid service object, its parent and the name of the service object in its parent's steps.
// Return an error to abort iteration
type ServiceIterator func(service *Service, parent *Service, sName string) *axerror.AXError

type Service struct {
	Id                string                      `json:"id,omitempty"`
	CostId            map[string]interface{}      `json:"costid,omitempty"`
	Template          EmbeddedTemplateIf          `json:"template,omitempty"`
	TemplateID        string                      `json:"template_id,omitempty"`
	Arguments         template.Arguments          `json:"arguments,omitempty"`
	Flags             map[string]interface{}      `json:"flags,omitempty"`
	Status            int                         `json:"status"`
	StatusString      string                      `json:"status_string"`
	StatusDetail      map[string]interface{}      `json:"status_detail,omitempty"`
	Name              string                      `json:"name,omitempty"`
	Description       string                      `json:"description,omitempty"`
	Mem               float64                     `json:"mem,omitempty"`
	CPU               float64                     `json:"cpu,omitempty"`
	User              string                      `json:"user,omitempty"`
	HostName          string                      `json:"host_name,omitempty"`
	HostId            string                      `json:"host_id,omitempty"`
	ContainerName     string                      `json:"container_name,omitempty"`
	ContainerId       string                      `json:"container_id,omitempty"`
	Cost              float64                     `json:"cost"`
	LogURL            string                      `json:"log_url,omitempty"`
	CreateTime        int64                       `json:"create_time"`
	LaunchTime        int64                       `json:"launch_time"`
	InitTime          int64                       `json:"init_time"`
	AverageInitTime   int64                       `json:"average_init_time"`
	EndTime           int64                       `json:"end_time"`
	WaitTime          int64                       `json:"wait_time"`
	RunTime           int64                       `json:"run_time"`
	AverageRunTime    int64                       `json:"average_runtime"`
	ArtifactTags      string                      `json:"artifact_tags"` // TODO: ask UI to allow omitempty
	Children          []*Service                  `json:"children,omitempty"`
	Parent            *Service                    `json:"-"`
	Notifications     []notification.Notification `json:"notifications,omitempty"`
	Commit            *commit.ApiCommit           `json:"commit,omitempty"`
	NewCommit         *commit.ApiCommit           `json:"newcommit,omitempty"`
	PolicyId          string                      `json:"policy_id,omitempty"`
	Labels            map[string]string           `json:"labels,omitempty"`
	Annotations       map[string]string           `json:"annotations,omitempty"`
	Endpoint          string                      `json:"endpoint,omitempty"`
	FailurePath       []interface{}               `json:"failure_path,omitempty"`
	TaskId            string                      `json:"task_id,omitempty"`
	TerminationPolicy *template.TerminationPolicy `json:"termination_policy,omitempty"`
	IsSubmitted       bool                        `json:"is_submitted"`
	JiraIssues        []string                    `json:"jira_issues,omitempty"`
	Fixtures          map[string]interface{}      `json:"fixtures,omitempty"`
	Repo              string                      `json:"repo,omitempty"`
	Branch            string                      `json:"branch,omitempty"`
}

func (s *Service) UnmarshalJSON(b []byte) error {
	// sAlias is an alias to Service used during unmarshalling so that the
	// custom unmarshaler will not get into a recursive loop
	type sAlias Service
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}
	// unmarshall everything but template field and rewrite b
	// template is an interface, and we use a custom function to unmarshal it
	templateRaw, ok := objMap["template"]
	if ok {
		delete(objMap, "template")
	}
	b, err = json.Marshal(objMap)
	if err != nil {
		return err
	}

	// unmarshall modifed b into the alias, and update self
	var newSvc sAlias
	err = json.Unmarshal(b, &newSvc)
	if err != nil {
		return err
	}
	newS := Service(newSvc)
	*s = newS

	// unmarshal the template using our UnmarshallEmbeddedTemplate function
	if templateRaw != nil {
		est, axErr := UnmarshalEmbeddedTemplate([]byte(*templateRaw))
		if axErr != nil {
			return fmt.Errorf(axErr.Error())
		}
		s.Template = est
	}
	return nil
}

func (s *Service) AggregateChildrenCost() {
	if s.Children == nil {
		s.Cost = 0.0
		return
	}

	var totalCost float64
	for _, child := range s.Children {
		if child != nil && child.Template != nil && child.Template.GetType() == template.TemplateTypeContainer {
			totalCost = totalCost + child.Cost
		}
	}

	s.Cost = totalCost
}

type ServiceSummary struct {
	Id             string `json:"ax_uuid,omitempty"`
	HostId         string `json:"host_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Status         int    `json:"status,omitempty"`
	User           string `json:"user_id,omitempty"`
	RunTime        int64  `json:"run_time,omitempty"`
	AverageRunTime int64  `json:"avg_run_time,omitempty"`
}

func (service *Service) Iterate(handler ServiceIterator, parent *Service, sName string) *axerror.AXError {
	// Do top down iteration. This is needed to pass parent parameter down to all the children.
	axErr := handler(service, parent, sName)
	if axErr != nil {
		return axErr
	}

	// if it's a leaf we are done
	if service.Template == nil {
		return nil
	}

	switch service.Template.GetType() {
	case template.TemplateTypeContainer:
		break
	case template.TemplateTypeDeployment:
		dt := service.Template.(*EmbeddedDeploymentTemplate)
		for name, templateRef := range dt.Containers {
			utils.DebugLog.Println("Processing child container:", name)
			axErr := templateRef.Iterate(handler, service, name)
			if axErr != nil {
				return axErr
			}
		}
	case template.TemplateTypeWorkflow:
		wft := service.Template.(*EmbeddedWorkflowTemplate)
		for _, parallelSteps := range wft.Fixtures {
			for name, fixRef := range parallelSteps {
				utils.DebugLog.Printf("Processing child fixture (dynamic=%v): %s", fixRef.IsDynamicFixture(), name)
				if fixRef.IsDynamicFixture() {
					axErr := fixRef.Iterate(handler, service, name)
					if axErr != nil {
						return axErr
					}
				}
			}
		}
		for _, parallelSteps := range wft.Steps {
			for name, step := range parallelSteps {
				utils.DebugLog.Println("Processing child step:", name)
				axErr := step.Iterate(handler, service, name)
				if axErr != nil {
					return axErr
				}
			}
		}
	}
	return nil
}

func (svc *Service) Copy() *Service {
	bytes, err := json.Marshal(svc)
	if err != nil {
		panic(fmt.Sprintf("Error copying service when marshaling %s, err %v", svc.Name, err))
	}
	var newService Service
	err = json.Unmarshal(bytes, &newService)
	if err != nil {
		panic(fmt.Sprintf("Error copying service when unmarshaling %s, err %v", svc.Name, err))
	}
	return &newService
}

// Copy the service and its subtree, return the copied service with the parent intact.
func (svc *Service) CopyTree(parent *Service) *Service {
	// This is not a frequent operation. We just do the simple thing that's not really the fastest.
	newService := svc.Copy()
	recreateIdHandler := func(s *Service, p *Service, name string) *axerror.AXError {
		s.Id = gocql.UUIDFromTime(time.Now()).String()
		s.Parent = parent
		return nil
	}
	newService.Iterate(recreateIdHandler, nil, "")
	return newService
}

// GetChildByName
// This code was preserved from legacy implementation and should be revisited.
func (service *Service) GetChildByName(name string) *Service {
	if service.Template == nil {
		return nil
	}

	switch service.Template.GetType() {
	case template.TemplateTypeContainer:
		break
	case template.TemplateTypeDeployment:
		dt := service.Template.(*EmbeddedDeploymentTemplate)
		if ctr, ok := dt.Containers[name]; ok {
			return ctr
		}
	case template.TemplateTypeWorkflow:
		wft := service.Template.(*EmbeddedWorkflowTemplate)
		for _, parallelSteps := range wft.Steps {
			if wfs, ok := parallelSteps[name]; ok {
				return wfs
			}
		}
		for _, parallelFixtures := range wft.Fixtures {
			if fix, ok := parallelFixtures[name]; ok {
				return fix.Service
			}
		}
	}
	return nil
}

func (service *Service) GetChildByID(id string) *Service {
	if service.Template == nil {
		return nil
	}
	switch service.Template.GetType() {
	case template.TemplateTypeContainer:
		break
	case template.TemplateTypeDeployment:
		dt := service.Template.(*EmbeddedDeploymentTemplate)
		for _, ctr := range dt.Containers {
			if ctr.Id == id {
				return ctr
			}
		}
	case template.TemplateTypeWorkflow:
		wft := service.Template.(*EmbeddedWorkflowTemplate)
		for _, parallelSteps := range wft.Steps {
			for _, wfs := range parallelSteps {
				if wfs.Id == id {
					return wfs
				}
			}
		}
		for _, parallelFixtures := range wft.Fixtures {
			for _, fix := range parallelFixtures {
				if fix.Id == id {
					return fix.Service
				}
			}
		}
	}
	return nil
}

func (service *Service) ReplaceEnvParameter(value interface{}, c *ServiceContext) interface{} {
	if value != nil {
		if reflect.TypeOf(value).Kind() == reflect.String {
			if id, exist := c.serviceDict[value.(string)]; exist {
				if s, exist := c.services[id]; exist && s.Status != utils.ServiceStatusSkipped {
					value = s.Id
					utils.InfoLog.Printf("[PartialRun]: old id: %v, use the new service id: %v since this step cannot be skipped.\n", value, s.Id)
				} else {
					utils.InfoLog.Printf("[PartialRun]: new id: %v, still use the old service id: %v since this step can be skipped.\n", s.Id, value)
				}
			}
		}
	}
	return value
}

// Resolve the environment parameters
func (service *Service) ResolveEnvParameter(value interface{}, c *ServiceContext) interface{} {
	if value == nil {
		return nil
	}

	if reflect.TypeOf(value).Kind() == reflect.String {
		str := value.(string)
		if strings.Contains(str, "%%") {
			replaced := strings.Replace(str, "%%", "", -1)
			tokens := strings.Split(replaced, ".")
			if len(tokens) < 2 {
				return nil
			}

			switch tokens[0] {
			case utils.EnvNSSession:
				if tokens[1] == utils.EnvKeyUser {
					if tokens[2] == "username" {
						return c.User.Username
					} else if tokens[2] == "password" {
						return c.User.Password
					}
				} else {
					return c.Arguments[replaced]
				}
			case utils.EnvNSService, utils.EnvNSStep, utils.EnvNSFixture, "step":
				// the format is service.sibling_servicename.id or service.child_servicename.id
				for {
					parent := service.Parent
					if parent != nil {
						foundService := parent.GetChildByName(tokens[1])
						if foundService != nil {
							if len(tokens) == 2 || tokens[2] == utils.EnvKeyId {
								return foundService.Id
							}
						}
					}
					foundService := service.GetChildByName(tokens[1])
					if foundService != nil {
						if len(tokens) == 2 || tokens[2] == utils.EnvKeyId {
							return foundService.Id
						}
					}
					if parent == nil {
						break
					} else {
						service = parent
					}
				}
			}
		}
	}
	return value
}

// Get the parameter value for the paramName. Return nil if we can't figure this param out.
func (service *Service) GetParameterValue(paramName string) *string {

	//utils.DebugLog.Printf("Resolve %v param - %v\n", service.Name, paramName)
	// the order matters below.
	if v, ok := service.Arguments[paramName]; ok {
		//utils.DebugLog.Printf("Resolve %v param - %v: %v found in service template\n", service.Name, paramName, v)
		return v
	}

	// first check the parent's parameters. Parameters passed down may be from the user.
	parent := service.Parent
	if parent != nil && parent.Arguments != nil && parent.Arguments[paramName] != nil {
		//utils.DebugLog.Printf("Resolve %v param - %v: %v found in parent service\n", service.Name, paramName, parent.Parameters[paramName])
		return parent.Arguments[paramName]
	}

	tempParam := service.Template.GetInputs().Parameters[paramName]
	// if there is a default value, use it.
	if tempParam != nil && tempParam.Default != nil {
		//utils.DebugLog.Printf("Resolve %v param - %v: %v found in default value\n", service.Name, paramName, tempParam.Default)
		return tempParam.Default
	}
	return nil
}

func GetServiceAverageTimes(templateName string) (int64, int64) {

	var fields []string
	fields = append(fields, axdb.AXDBUUIDColumnName)
	fields = append(fields, axdb.AXDBTimeColumnName)
	fields = append(fields, ServiceTaskId)
	fields = append(fields, ServiceIsTask)
	fields = append(fields, ServiceLaunchTime)
	fields = append(fields, ServiceRunTime)
	fields = append(fields, ServiceWaitTime)
	fields = append(fields, ServiceEndTime)
	fields = append(fields, ServiceAverageRunTime)
	fields = append(fields, ServiceAverageWaitTime)

	// we use the average of the recent runs for now. Later may switch to using the DB stat on run_time.
	params := map[string]interface{}{
		ServiceTemplateName:      templateName,
		ServiceStatus:            utils.ServiceStatusSuccess,
		axdb.AXDBQueryMaxEntries: 10,
		axdb.AXDBSelectColumns:   fields,
	}
	serviceArray, axErr := GetServicesFromTable(DoneServiceTable, false, params)
	if axErr != nil || len(serviceArray) == 0 {
		return 400, 60
	}
	var count int64
	var sumRuntime int64
	var sumInitTime int64
	for _, service := range serviceArray {
		count++
		sumRuntime = sumRuntime + service.RunTime
		sumInitTime = sumInitTime + service.InitTime
	}
	return sumRuntime / count, sumInitTime / count
}

// Get the max CPU and memory needed at any time for the service.
func (service *Service) GetMaxResources() (cpu float64, mem float64) {
	if service.CPU > 0.0 || service.Mem > 0.0 {
		return service.CPU, service.Mem
	}
	if service.Template == nil {
		service.CPU = 0.0
		service.Mem = 0.0
		return 0.0, 0.0
	}
	switch service.Template.GetType() {
	case template.TemplateTypeContainer:
		ct := service.Template.(*EmbeddedContainerTemplate)
		if ct.Resources != nil {
			cpuCores, _ := ct.Resources.CPUCoresValue()
			cpu += cpuCores
			memMiB, _ := ct.Resources.MemMiBValue()
			mem += memMiB
		}
		service.CPU = cpu
		service.Mem = mem
		return cpu, mem
	case template.TemplateTypeWorkflow:
		wft := service.Template.(*EmbeddedWorkflowTemplate)
		// fixtures are all running in parallel, sum up all their resources.
		for _, step := range wft.Fixtures {
			for _, fixture := range step {
				if fixture.IsDynamicFixture() {
					c, m := fixture.Service.GetMaxResources()
					cpu += c
					mem += m
				}
			}
		}
		maxStepCPU := 0.0
		maxStepMem := 0.0
		for _, step := range wft.Steps {
			stepCPU := 0.0
			stepMem := 0.0
			for _, s := range step {
				c, m := s.GetMaxResources()
				stepCPU += c
				stepMem += m
			}
			if stepCPU > maxStepCPU {
				maxStepCPU = stepCPU
			}
			if stepMem > maxStepMem {
				maxStepMem = stepMem
			}
		}
		cpu += maxStepCPU
		mem += maxStepMem
		service.Mem = mem
		service.CPU = cpu
		return cpu, mem
	case template.TemplateTypeDeployment:
		// deployment resource usage is ignored in workflow
		cpu = 0.0
		mem = 0.0
		service.CPU = cpu
		service.Mem = mem
		return cpu, mem
	default:
		return 0, 0
	}
}

/**
 * Estimate the number of parallel containers for the job
 */
func (service *Service) GetEstimatedParallelContainers() int {
	if service.Template == nil {
		return 0
	}
	switch service.Template.GetType() {
	case template.TemplateTypeContainer:
		return 1
	case template.TemplateTypeWorkflow:
		wft := service.Template.(*EmbeddedWorkflowTemplate)
		return int(len(service.Children) / len(wft.Steps))
	default:
		return 0
	}
}

func (service *Service) InitUUID(parent *Service, c *ServiceContext, children []*Service) *axerror.AXError {
	currentTime := time.Now()
	service.CreateTime = currentTime.Unix()
	service.LaunchTime = 0
	var oldServID string = ""
	utils.InfoLog.Printf("[PartialRun]: in initUUID(), service name = %s, service id = %s, len of children %d", service.Name, service.Id, len(children))
	for _, child := range children {
		utils.InfoLog.Printf("[PartialRun]: in initUUID(), service id = %s, child id = %s, service status = %d, child status = %d", service.Id, child.Id, service.Status, child.Status)
		if child.Id == service.Id {
			utils.InfoLog.Printf("[PartialRun]: in initUUID(), service name = %s, service status = %d, child status = %d", service.Name, service.Status, child.Status)
			service.Status = child.Status
			oldServID = child.Id
			break
		}
	}
	service.Id = gocql.UUIDFromTime(currentTime).String()
	c.services[service.Id] = service
	c.serviceDict[oldServID] = service.Id
	service.Parent = parent
	service.IsSubmitted = false
	return nil
}

func (service *Service) UpdateUser(c *ServiceContext) *axerror.AXError {

	if c.User != nil {
		service.User = c.User.Username
	}

	if service.CostId != nil {
		service.CostId["user"] = service.User
	}

	return nil
}

func (service *Service) MarkServiceStatus(status int) *axerror.AXError {
	utils.InfoLog.Printf("[PartialRun]: mark status for service %s", service.Name)
	if service != nil {
		service.Status = status
		if service.Status == utils.ServiceStatusInitiating {
			if service.Flags != nil {
				if b, ok := service.Flags["skipped"]; ok && b.(bool) == true {
					utils.InfoLog.Printf("[PartialRun]: flag of service %s was reset to non-skipped.\n", service.Name)
					service.Flags["skipped"] = false
					service.StatusDetail["code"] = ""
				}
			}
		}
	}

	return nil
}

func (service *Service) markFixtureStatus(status int) {
	utils.InfoLog.Printf("[PartialRun]: deal with fixture for service %s", service.Name)
	if service == nil || service.Template == nil {
		return
	}
	handlerMarkStatus := func(s *Service, p *Service, name string) *axerror.AXError {
		return s.MarkServiceStatus(status)
	}
	wft, ok := service.Template.(*EmbeddedWorkflowTemplate)
	if !ok {
		return
	}
	for _, parallelFixtures := range wft.Fixtures {
		for name, fix := range parallelFixtures {
			utils.InfoLog.Printf("[PartialRun]: deal with fixture %s at level %s for service %s", name, fix.Service.Name, service.Name)
			fix.Service.Iterate(handlerMarkStatus, service, name)
		}
	}
}

// this function label status of current service to be "Skipped", so that workflow executor will skip running this service
func (service *Service) SetInitialStatus(c *ServiceContext) *axerror.AXError {
	utils.InfoLog.Printf("[PartialRun]: before set service name: %s, status = %d", service.Name, service.Status)
	isPartial := c.IsPartial
	if !isPartial {
		service.Status = utils.ServiceStatusInitiating
	} else {
		// the step doesn't depend on new commit
		if !service.DependsOnNewCommit(c) {
			if service.Status != utils.ServiceStatusSuccess && service.Status != utils.ServiceStatusSkipped {
				service.Status = utils.ServiceStatusInitiating
				// if the service depends on fixture, all fixtures step should be marked as initiating.
				service.markFixtureStatus(utils.ServiceStatusInitiating)
			} else {
				service.Status = utils.ServiceStatusSkipped
			}
		} else {
			// the step depends on new commit; reset it to initiating regardless of its status in previous run
			preStatus := service.Status
			if preStatus != utils.ServiceStatusSuccess {
				utils.InfoLog.Printf("[PartialRun]: failed service %s was set to initiating first.\n", service.Name)
				service.Status = utils.ServiceStatusInitiating
				//service.ResetDependentChain()
			} else {
				service.Status = utils.ServiceStatusSkipped
			}
		}
	}

	utils.InfoLog.Printf("[PartialRun]: after set service name: %s, status = %d", service.Name, service.Status)
	return nil

}

// this function label status of current service to be "Skipped", so that workflow executor will skip running this service
func (service *Service) SetInitialStatus1(parent *Service, c *ServiceContext, isPartial bool, children []*Service) *axerror.AXError {
	utils.InfoLog.Printf("[PartialRun]: before set service name: %s, status = %d", service.Name, service.Status)
	// the following is a working version with different understanding on re-run successful steps
	var canSkip = false
	if isPartial {
		if service.Status == utils.ServiceStatusSuccess {
			if !service.DependsOnNewCommit(c) && !service.DependsOnNonSkippedService(c) {
				canSkip = true
			}
		}
	}

	if canSkip {
		service.Status = utils.ServiceStatusSkipped
	} else {
		service.Status = utils.ServiceStatusInitiating
	}

	utils.InfoLog.Printf("[PartialRun]: after set service name: %s, status = %d", service.Name, service.Status)
	return nil
}

func (service *Service) DependsOnNewCommit(c *ServiceContext) bool {
	utils.InfoLog.Printf("[PartialRun]: In dependsOnNewCommit, service = %s", service.Name)
	if service == c.Root && !c.DependsNewCommit {
		newCommit := service.NewCommit
		// the new commit isn't specified, we think it is a re-run against the old commit
		if newCommit == nil {
			c.DependsNewCommit = false
		} else {
			var oldCommit *commit.ApiCommit
			if c.Root != nil {
				oldCommit = c.Root.Commit
			}
			if oldCommit != nil {
				//new commit is the same as the old
				if oldCommit.Revision == newCommit.Revision && oldCommit.Repo == newCommit.Repo {
					c.DependsNewCommit = false
				} else {
					cmt, _ := commit.GetAPICommitByRevision(newCommit.Revision, newCommit.Repo)
					if cmt != nil {
						service.Commit = cmt
					} else {
						service.Commit = newCommit
					}
					c.DependsNewCommit = true
				}
			} else {
				c.DependsNewCommit = true
			}
		}
	}
	utils.InfoLog.Printf("[PartialRun]: service %s depends On NewCommit?: %t", service.Name, c.DependsNewCommit)
	return c.DependsNewCommit
}

func (service *Service) ResetDependentChain(c *ServiceContext) {
	inputs := service.Template.GetInputs()
	if inputs == nil || inputs.Artifacts == nil {
		return
	}
	/* TODO: re-implement partial re-run
	for _, artifact := range inputs.Artifacts {
		if len(artifact.ServiceId) != 0 {
			// the id of dependent service
			id := artifact.ServiceId
			utils.InfoLog.Printf("[PartialRun]: in ResetChain() service %s, depend id %s", service.Name, id)
			//backtrack to the dependent service, all services on the path are labled as "initiating"
			if serv := c.services[id]; serv != nil {
				utils.InfoLog.Printf("[ParitalRun]: ancestor %s was reset to initiating.\n", serv.Name)
				//serv.Status = utils.ServiceStatusInitiating
				serv.MarkServiceStatus(utils.ServiceStatusInitiating)
				serv.markFixtureStatus(utils.ServiceStatusInitiating)
				for {
					//service.Status = utils.ServiceStatusInitiating
					service.MarkServiceStatus(utils.ServiceStatusInitiating)
					service.markFixtureStatus(utils.ServiceStatusInitiating)
					parent := service.Parent
					if parent != nil {
						foundService := parent.GetChildByID(id)
						if foundService != nil && foundService == serv {
							break
						} else {
							service = parent
						}
					} else {
						break
					}
				}
			}
		}
	}
	*/
}

func (service *Service) DependsOnNonSkippedService(c *ServiceContext) bool {
	utils.InfoLog.Printf("[PartialRun]: In DependsOnNonSkippedService, service = %v", service)
	utils.InfoLog.Printf("[PartialRun]: input: %v", service.Template.GetInputs())
	/* TODO: re-implement partial re-run
	if service.Template.Inputs != nil && service.Template.Inputs.Artifacts != nil {
		for _, artifact := range service.Template.Inputs.Artifacts {
			if len(artifact.ServiceId) != 0 {
				// the id of dependent service
				id := artifact.ServiceId
				//backtrack to the dependent service, all services on the path are labled as "initiating"
				if serv := c.services[id]; serv != nil {
					if serv.Status != utils.ServiceStatusSkipped {
						for {
							service.Status = utils.ServiceStatusInitiating
							parent := service.Parent
							if parent != nil {
								foundService := parent.GetChildByID(id)
								if foundService != nil && foundService == serv {
									break
								}
							}

							if parent == nil {
								break
							} else {
								service = parent
							}
						}
						return true
					}
				}
			}
		}
	}
	*/
	return false
}

func (service *Service) FulfillParameters(parent *Service, c *ServiceContext) *axerror.AXError {
	service.Parent = parent
	if service.Arguments == nil {
		service.Arguments = make(map[string]*string)
	}
	inputs := service.Template.GetInputs()
	if inputs == nil || inputs.Parameters == nil {
		return nil
	}
	tempParams := inputs.Parameters
	if len(tempParams) > 0 && service.Arguments == nil {
		service.Arguments = make(map[string]*string)
	}

	for name, _ := range tempParams {
		value := service.Arguments[name]
		if value == nil {
			// this parameter is not set yet. Try to generate it based on default and parent.
			value = service.GetParameterValue(name)
			if value == nil {
				// something is wrong. This shouldn't be possible. The UI should have collected everything.
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("Parameter can't be determined: %s.%s", service.Name, name))
			}
		}
		v := service.ReplaceEnvParameter(service.ResolveEnvParameter(*value, c), c).(*string)
		service.Arguments[name] = v
		utils.InfoLog.Printf("%s.%s = %v", service.Template.GetName(), name, service.Arguments[name])
	}
	return nil
}

func (service *Service) Init(parent *Service, c *ServiceContext) *axerror.AXError {
	service.IsSubmitted = false

	service.Parent = parent

	service.SetInitialStatus(c)

	service.StatusDetail = map[string]interface{}{}
	if service.Status == utils.ServiceStatusSkipped {
		if service.Flags == nil {
			service.Flags = map[string]interface{}{}
		}
		service.Flags["skipped"] = true
		service.StatusDetail["code"] = "TASK_SKIPPED"
	}

	//if service.DependsOnNewCommit(c) {
	if c.IsPartial {
		// use the parameter value in servicecontext.parameters
		for name := range service.Arguments {
			if c.Arguments != nil {
				if v, ok := c.Arguments[name]; ok {
					service.Arguments[name] = v
				}
			}
		}
		//delete(service.Parameters, "commit")
		//delete(service.Parameters, "repo")
	}

	service.GetMaxResources()
	if c.User != nil {
		service.User = c.User.Username
	}

	if c != nil {
		service.TaskId = c.Root.Id
	}

	// If there is no template, it's a static fixture. Nothing else needs to be done in that case.
	if service.Template == nil {
		return nil
	}

	service.Labels = service.Template.GetLabels()

	// Only populate when template is available
	var wft *EmbeddedWorkflowTemplate
	var dt *EmbeddedDeploymentTemplate
	switch service.Template.GetType() {
	case template.TemplateTypeWorkflow:
		wft = service.Template.(*EmbeddedWorkflowTemplate)
	case template.TemplateTypeDeployment:
		dt = service.Template.(*EmbeddedDeploymentTemplate)
	}

	if service.Name == "" {
		service.Name = service.Template.GetName()
	}
	service.AverageRunTime, service.AverageInitTime = GetServiceAverageTimes(service.Template.GetName())
	service.CostId = map[string]interface{}{"user": service.User, "service": c.Root.Template.GetName(), "app": "axdevops"}

	// TODO: removing parameter fulfilment since this now done later -Jesse
	// axErr := service.FulfillParameters(parent, c)
	// if axErr != nil {
	// 	return axErr
	// }

	/* TODO: re-implement partial re-run
	// We allow the artifacts section to refer to services directly.
	if service.Template.Outputs != nil && service.Template.Outputs.Artifacts != nil {
		for _, artifact := range service.Template.Outputs.Artifacts {
			if len(artifact.ServiceId) != 0 && strings.Contains(artifact.ServiceId, "%%") {
				// Add to parameters but leave the service template alone
				paramName := strings.Replace(artifact.ServiceId, "%%", "", -1)
				if service.Parameters[paramName] == nil {
					service.Parameters[paramName] = service.ReplaceEnvParameter(service.ResolveEnvParameter(artifact.ServiceId, c), c).(string)
				}
			}
		}
	}
	if service.Template.Inputs != nil && service.Template.Inputs.Artifacts != nil {
		for _, artifact := range service.Template.Inputs.Artifacts {
			if !service.DependsOnNewCommit(c) && len(artifact.ServiceId) != 0 && !strings.Contains(artifact.ServiceId, "%%") {
				utils.InfoLog.Printf("[PartialRun]: service %s still use old artifact %s.\n", service.Name, artifact.ServiceId)
				continue
			}
			utils.InfoLog.Printf("[PartialRun]: service %s will use new artifact.\n", service.Name)
			if len(artifact.ServiceId) != 0 && strings.Contains(artifact.ServiceId, "%%") {
				utils.InfoLog.Printf("[PartialRun]: service %s figuring out new artifact with serviceId.\n", service.Name)
				// spcified by tag_id.step
				if len(artifact.Tag) != 0 {
					tagName := artifact.Tag

					utils.InfoLog.Printf("[Cross]: tagName = %s", tagName)
					s, err := GetServiceByArtifactTag(tagName)
					if err != nil || s == nil {
						return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("artifact tag %s can't be resolved", artifact.Tag)
					}

					artifact.ServiceId = s.Id
					utils.InfoLog.Printf("[Cross]: artifact service id: %s", s.Id)
				} else if len(artifact.WorkflowID) != 0 {
					utils.InfoLog.Printf("Cross]: artifact service id: %s", artifact.WorkflowID)
					artifact.ServiceId = artifact.WorkflowID
				} else {
					// Add to parameters but leave the service template alone
					paramName := strings.Replace(artifact.ServiceId, "%%", "", -1)
					if service.Parameters[paramName] == nil {
						service.Parameters[paramName] = service.ReplaceEnvParameter(service.ResolveEnvParameter(artifact.ServiceId, c), c).(string)
					}

				}
			} else if len(artifact.From) != 0 && strings.Contains(artifact.From, "%%") {
				utils.InfoLog.Printf("[PartialRun]: service %s figuring out new artifact with serviceId.\n", service.Name)
				paramName := strings.Replace(artifact.From, "%%", "", -1)

				if service.Parameters[paramName] == nil {
					value := service.GetParameterValue(paramName)
					if value == nil {
						return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("artifact from: %s can't be resolved", artifact.From)
					}
					service.Parameters[paramName] = service.ReplaceEnvParameter(service.ResolveEnvParameter(value, c), c)
				}
				ar := strings.Split(strings.Replace(service.Parameters[paramName].(string), "%%", "", -1), ".")
				if len(ar) != 3 && len(ar) != 4 {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("artifact from: %s can't be resolved", artifact.From)
				}
				if ar[0] == utils.EnvNSArtifact {
					if ar[1] == utils.EnvNSTag {
						utils.InfoLog.Printf("[Cross]: tagName = %s", ar[2])
						s, err := GetServiceByArtifactTag(ar[2])
						if err != nil || s == nil {
							return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("artifact tag %s can't be resolved", artifact.Tag)
						}
						artifact.ServiceId = s.Id
						artifact.Name = ar[3]
					} else if ar[1] == utils.EnvNSWorkflow {
						utils.InfoLog.Printf("[Cross]: serviceID = %s", ar[1])
						artifact.ServiceId = ar[2]
						artifact.Name = ar[3]
					}
				} else {
					artifact.ServiceId = service.ReplaceEnvParameter(service.ResolveEnvParameter("%%steps."+ar[1]+"%%", c), c).(string)
					artifact.Name = ar[2]
				}

			} else if len(artifact.From) != 0 {
				utils.InfoLog.Printf("[PartialRun]: service %s figuring out new artifact with serviceId, have info From: %s.\n", service.Name, artifact.From)
			}
		}
	}
	*/

	switch service.Template.GetType() {
	case template.TemplateTypeWorkflow:
		service.TerminationPolicy = wft.TerminationPolicy
	case template.TemplateTypeDeployment:
		service.TerminationPolicy = dt.TerminationPolicy
	}

	if parent == nil {
		// deal with the tags for workflow node.
		// if wft != nil && len(wft.ArtifactTags) > 0 {
		// 	tagsBytes, err := json.Marshal(wft.ArtifactTags)
		// 	if err != nil {
		// 		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("artifact tags (%v) isn't in valid format", wft.ArtifactTags)
		// 	}
		// 	service.ArtifactTags = string(tagsBytes[:])
		// }

		// populate the default 'failure' path
		path := []interface{}{}
		switch service.Template.GetType() {
		case template.TemplateTypeContainer:
			path = append(path, service.Name)
		case template.TemplateTypeDeployment:
			for name := range dt.Containers {
				path = append(path, name)
				break
			}
		case template.TemplateTypeWorkflow:
			for _, steps := range wft.Steps {
				if len(steps) != 0 {
					for name, _ := range steps {
						path = append(path, name)
						break
					}
				}
			}
		}
		service.FailurePath = path
	}

	if service.DependsOnNewCommit(c) && service.Status == utils.ServiceStatusInitiating {
		service.ResetDependentChain(c)
	}
	// Must fulfill the children's parameters before trying to expand. The expand parameter
	// logic depends on knowing the parameter values. For instance, for a workflow, if it has
	// a special parameter that's used by multiple children, we need to treat it differently.

	// Disabling call to FulfillParameters since substitution is handled later
	// if wft != nil {
	// 	for _, step := range wft.Steps {
	// 		for _, srv := range step {
	// 			utils.InfoLog.Printf("fulfilling service %s", srv.Template.GetName())
	// 			axErr := srv.FulfillParameters(service, c)
	// 			if axErr != nil {
	// 				return axErr
	// 			}
	// 		}
	// 	}
	// }

	// TODO: revisit -Jesse
	//return service.ExpandParameterArray(parent, c)
	return nil
}

func (self *Service) ShouldExpand(c *ServiceContext) bool {
	utils.InfoLog.Printf("checking should expand: %s", self.Template.GetName())

	// Only expand the nested workflow, we should restrict the case for now. If it is a single container job, simple
	// workflow job, expanding result should be a multiple steps job or simple just multiple jobs? Not sure.
	if self.Template.GetType() != template.TemplateTypeWorkflow {
		utils.InfoLog.Printf("Not workflow, return true")
		return true
	}
	wft := self.Template.(*EmbeddedWorkflowTemplate)

	// We only expand workflows if a dynamic parameter is needed for multiple steps. In this
	// case the most likely case is that we need to maintain the consistency of the parameter
	// passed to all the steps.
	for param, paramV := range self.Arguments {
		if paramV != nil && strings.Contains(*paramV, "$$[") {
			continue
		}
		count := 0
		for _, parallelSteps := range wft.Steps {
			for _, step := range parallelSteps {
				if step.Arguments[param] != nil {
					count++
				}
			}
		}
		for _, parallelFixtures := range wft.Fixtures {
			for _, fix := range parallelFixtures {
				if fix.Arguments[param] != nil {
					count++
				}
			}
		}
		//if count > 1 {
		//	utils.InfoLog.Printf("param %s is used multiple times, return true", param)
		//	return true
		//}
		if count > 0 {
			utils.InfoLog.Printf("param %s is used, return true", param)
			return true
		}
	}
	utils.InfoLog.Printf("return false")
	return false
}

func (self *Service) ExpandParameterArray(parent *Service, c *ServiceContext) *axerror.AXError {
	if self.Template.GetType() != template.TemplateTypeWorkflow {
		return nil
	}
	wft := self.Template.(*EmbeddedWorkflowTemplate)

	utils.InfoLog.Printf("\nExpandParameterArray: %s", self.Template.GetName())

	expandStep := func(parallelSteps map[string]*Service) bool {
		expanded := false
		for stepName, step := range parallelSteps {
			if expanded {
				break
			}

			if !step.ShouldExpand(c) {
				continue
			}

			for pName, pValue := range step.Arguments {
				utils.InfoLog.Printf("checking params %s %v", pName, pValue)
				if expanded {
					break
				}
				if pValue == nil {
					continue
				}

				pString := *pValue
				if strings.Contains(pString, "$$[") && strings.Contains(pString, "]$$") {
					utils.InfoLog.Printf("expanding")
					stripped := strings.Replace(pString, "$$[", "", -1)
					stripped = strings.Replace(stripped, "]$$", "", -1)
					array := strings.Split(stripped, ",")
					if len(array) != 0 {
						expanded = true
						delete(parallelSteps, stepName)
						for i, val := range array {
							stepPostfix := "_ax" + strconv.Itoa(i)
							if _, ok := parallelSteps[stepName+stepPostfix]; !ok {
								// Copy the whole suite instead of a simple step,
								// because there might be some step-step or step-fixture
								// reference, copying step only would break it, also
								// it is easier to reason about the end result.
								dup := step.CopyTree(self)
								dup.Arguments[pName] = &val
								dup.Flags = step.Flags
								parallelSteps[stepName+stepPostfix] = dup
							}
						}
					}
				}
			}
		}
		return expanded
	}

	// If the parameter is of the special format $$[...]$$ we need to replace the parameters.
	// Here we are keeping expanding the parameter, for each step, it might have multiple parameters to expand, so
	// after expanded, start over to check if anything needs to be expanded.
	for true {
		expanded := false
		for _, parallelSteps := range wft.Steps {
			if expanded {
				break
			}
			expanded = expandStep(parallelSteps)
		}
		if !expanded {
			break
		}
	}

	return nil
}

func (service *Service) InitAll(c *ServiceContext) *axerror.AXError {
	c.services = make(ServiceMap)
	c.serviceDict = make(map[string]string)
	children := service.Children
	handlerUUID := func(s *Service, p *Service, name string) *axerror.AXError {
		return s.InitUUID(p, c, children)
	}
	service.Iterate(handlerUUID, nil, "")

	utils.InfoLog.Printf("starting init")

	handler := func(s *Service, p *Service, name string) *axerror.AXError {
		return s.Init(p, c)
	}
	axErr := service.Iterate(handler, nil, "")
	utils.InfoLog.Printf("done init")
	return axErr
}

// Init a service object based on a map from DB
func (service *Service) InitFromMap(srvMap map[string]interface{}) *axerror.AXError {
	if srvMap[axdb.AXDBUUIDColumnName] != nil {
		service.Id = srvMap[axdb.AXDBUUIDColumnName].(string)
	}

	if srvMap[ServiceTemplateStr] != nil {
		tmplBytes := []byte(srvMap[ServiceTemplateStr].(string))
		// the static fixture service object has empty template string.
		if len(tmplBytes) > 0 {
			tmpl, axErr := UnmarshalEmbeddedTemplate(tmplBytes)
			if axErr != nil {
				utils.InfoLog.Printf("Failed to unmarshal template: %v", axErr)
				return axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal template from string: %s", srvMap[ServiceTemplateStr]))
			}
			service.Template = tmpl
		}
	}
	if srvMap[ServiceTemplateId] != nil {
		service.TemplateID = srvMap[ServiceTemplateId].(string)
	}

	if srvMap[ServiceArguments] != nil {
		argMap := srvMap[ServiceArguments].(map[string]interface{})
		service.Arguments = make(template.Arguments)
		for argName, argIf := range argMap {
			if argIf == nil {
				service.Arguments[argName] = nil
			} else {
				argStr := argIf.(string)
				service.Arguments[argName] = &argStr
			}
		}
	}

	if srvMap[ServiceFlags] != nil {
		service.Flags = srvMap[ServiceFlags].(map[string]interface{})
	}

	if srvMap[ServiceLabels] != nil {
		service.Labels = map[string]string{}
		if labelMap, ok := srvMap[ServiceLabels].(map[string]interface{}); ok {
			for key, value := range labelMap {
				service.Labels[key] = value.(string)
			}
		} else {
			if labelMap, ok := srvMap[ServiceLabels].(map[string]string); ok {
				service.Labels = labelMap
			}
		}
	}

	if srvMap[ServiceAnnotations] != nil {
		service.Annotations = map[string]string{}
		if annotationMap, ok := srvMap[ServiceAnnotations].(map[string]interface{}); ok {
			for key, value := range annotationMap {
				service.Annotations[key] = value.(string)
			}
		} else {
			if annotationMap, ok := srvMap[ServiceAnnotations].(map[string]string); ok {
				service.Annotations = annotationMap
			}
		}
	}

	if srvMap[ServiceStatus] != nil {
		service.Status = int(srvMap[ServiceStatus].(float64))
	}

	if srvMap[ServiceName] != nil {
		service.Name = srvMap[ServiceName].(string)
	}

	if srvMap[ServiceBranch] != nil {
		service.Branch = srvMap[ServiceBranch].(string)
	}

	if srvMap[ServiceRepo] != nil {
		service.Repo = srvMap[ServiceRepo].(string)
	}

	if srvMap[ServiceDescription] != nil {
		service.Description = srvMap[ServiceDescription].(string)
	}

	if srvMap[ServiceMem] != nil {
		service.Mem = srvMap[ServiceMem].(float64)
	}

	if srvMap[ServiceCPU] != nil {
		service.CPU = srvMap[ServiceCPU].(float64)
	}

	if srvMap[ServiceUserName] != nil {
		service.User = srvMap[ServiceUserName].(string)
	}

	if srvMap[ServiceEndpoint] != nil {
		service.Endpoint = srvMap[ServiceEndpoint].(string)
	}

	if tagStr, ok := srvMap[ServiceArtifactTags]; ok {
		service.ArtifactTags = tagStr.(string)
	}

	if jiraObjs, ok := srvMap[ServiceJiraIssues]; ok {
		for _, jiraObj := range jiraObjs.([]interface{}) {
			service.JiraIssues = append(service.JiraIssues, jiraObj.(string))
		}
	}

	if srvMap[ServiceTaskId] != nil {
		service.TaskId = srvMap[ServiceTaskId].(string)
	}

	if notificationStr, ok := srvMap[ServiceNotifications]; ok && notificationStr.(string) != "" {
		notifications := []notification.Notification{}
		if err := json.Unmarshal([]byte(notificationStr.(string)), &notifications); err != nil {
			errMsg := fmt.Sprintf("Failed to unmarshal the notifications string in service:%v", err)
			utils.ErrorLog.Println(errMsg)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
		}
		service.Notifications = notifications
	}

	if termPolicyStr, ok := srvMap[ServiceTerminationPolicy]; ok && termPolicyStr.(string) != "" {
		termPolicy := template.TerminationPolicy{}
		if err := json.Unmarshal([]byte(termPolicyStr.(string)), &termPolicy); err != nil {
			errMsg := fmt.Sprintf("Failed to unmarshal the termination policy string in service:%v", err)
			utils.ErrorLog.Println(errMsg)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
		}
		service.TerminationPolicy = &termPolicy
	}

	if statusDetailStr, ok := srvMap[ServiceStatusDetail]; ok && statusDetailStr.(string) != "" {
		statusDetail := map[string]interface{}{}
		if err := json.Unmarshal([]byte(statusDetailStr.(string)), &statusDetail); err != nil {
			errMsg := fmt.Sprintf("Failed to unmarshal the status detail string in service:%v", err)
			utils.ErrorLog.Println(errMsg)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
		}
		service.StatusDetail = statusDetail
	}

	if commitStr, ok := srvMap[ServiceCommit]; ok && commitStr.(string) != "" {
		commit := commit.ApiCommit{}
		if err := json.Unmarshal([]byte(commitStr.(string)), &commit); err != nil {
			errMsg := fmt.Sprintf("Failed to unmarshal the commit string in service:%v", err)
			utils.ErrorLog.Println(errMsg)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to unmarshal the commit string in service:%v", err))
		}
		commit.Jobs = nil
		service.Commit = &commit
	}

	if srvMap[ServicePolicyId] != nil {
		service.PolicyId = srvMap[ServicePolicyId].(string)
	}

	if srvMap[ServiceHostName] != nil {
		service.HostName = srvMap[ServiceHostName].(string)
	}

	if srvMap[ServiceHostId] != nil {
		service.HostId = srvMap[ServiceHostId].(string)
	}

	if srvMap[ServiceContainerName] != nil {
		service.ContainerName = srvMap[ServiceContainerName].(string)
	}
	if srvMap[ServiceContainerId] != nil {
		service.ContainerId = srvMap[ServiceContainerId].(string)
	}
	if srvMap[ServiceCostId] != nil {
		costid := srvMap[ServiceCostId].(map[string]interface{})
		service.CostId = costid
	}

	// time related
	if srvMap[axdb.AXDBTimeColumnName] != nil {
		service.CreateTime = int64(srvMap[axdb.AXDBTimeColumnName].(float64)) / 1e6
	}

	if srvMap[ServiceAverageRunTime] != nil {
		service.AverageRunTime = int64(srvMap[ServiceAverageRunTime].(float64)) / 1e6
	}

	if srvMap[ServiceAverageInitTime] != nil {
		service.AverageInitTime = int64(srvMap[ServiceAverageInitTime].(float64)) / 1e6
	}

	var launchTimeUs int64
	if srvMap[ServiceLaunchTime] != nil {
		launchTimeUs = int64(srvMap[ServiceLaunchTime].(float64))
		service.LaunchTime = launchTimeUs / 1e6
	}

	if srvMap[ServiceEndTime] != nil {
		endTimeUs := int64(srvMap[ServiceEndTime].(float64))
		service.EndTime = endTimeUs / 1e6
	}

	var waitTimeUs int64 = 0
	var runTimeUs int64 = 0

	switch service.Status {
	case utils.ServiceStatusInitiating:
		service.Cost = 0.0
	case utils.ServiceStatusWaiting:
		if launchTimeUs != 0 {
			waitTimeUs = time.Now().UnixNano()/1e3 - launchTimeUs
		}
		service.Cost = 0.0
	case utils.ServiceStatusRunning, utils.ServiceStatusCanceling:
		if _, ok := srvMap[ServiceWaitTime]; ok {
			waitTimeUs = int64(srvMap[ServiceWaitTime].(float64))
		}
		if launchTimeUs != 0 {
			runTimeUs = time.Now().UnixNano()/1e3 - launchTimeUs - waitTimeUs
		}
		service.Cost = GetSpendingCents(service.CPU, service.Mem, float64(runTimeUs))
	case utils.ServiceStatusSuccess, utils.ServiceStatusCancelled, utils.ServiceStatusFailed:
		if _, ok := srvMap[ServiceWaitTime]; ok {
			waitTimeUs = int64(srvMap[ServiceWaitTime].(float64))
		}
		if _, ok := srvMap[ServiceRunTime]; ok {
			runTimeUs = int64(srvMap[ServiceRunTime].(float64))
		}
		if srvMap[ServiceCost] != nil {
			service.Cost = srvMap[ServiceCost].(float64)
		}
	default:
		utils.DebugLog.Println("Expected service status:", service.Status)
	}

	if srvMap[ServiceStatusString] != nil {
		service.StatusString = srvMap[ServiceStatusString].(string)
	}

	service.RunTime = runTimeUs / 1e6
	service.WaitTime = waitTimeUs / 1e6

	if service.Status != utils.ServiceStatusInitiating {
		service.InitTime = service.LaunchTime - service.CreateTime
	} else {
		service.InitTime = time.Now().Unix() - service.CreateTime
	}

	if srvMap[ServiceFailurePath] != nil {
		service.FailurePath = srvMap[ServiceFailurePath].([]interface{})
	}

	if srvMap[ServiceIsSubmitted] != nil {
		service.IsSubmitted = srvMap[ServiceIsSubmitted].(bool)
	}

	if srvMap[ServiceFixtures] != nil {
		if fixtureMap, ok := srvMap[ServiceFixtures].(map[string]interface{}); ok {
			service.Fixtures = map[string]interface{}{}
			for instanceID, value := range fixtureMap {
				var instanceDoc interface{}
				if err := json.Unmarshal([]byte(value.(string)), &instanceDoc); err != nil {
					errMsg := fmt.Sprintf("Failed to unmarshal the fixture string in service:%v", err)
					utils.ErrorLog.Println(errMsg)
					return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
				}
				service.Fixtures[instanceID] = instanceDoc
			}
		}
	}

	return nil
}

func (service *Service) CreateServiceMap(c *ServiceContext) (map[string]interface{}, *axerror.AXError) {
	if len(service.Id) == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("service doesn't have an id")
	}
	uuid, err := gocql.ParseUUID(service.Id)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Invalid service uuid: " + service.Id)
	}

	srvMap := make(map[string]interface{})
	srvMap[axdb.AXDBUUIDColumnName] = service.Id
	srvMap[axdb.AXDBTimeColumnName] = uuid.Time().UnixNano() / 1e3
	srvMap[ServiceLaunchTime] = service.LaunchTime
	srvMap[ServiceRunTime] = service.RunTime
	srvMap[ServiceWaitTime] = service.WaitTime
	srvMap[ServiceEndTime] = service.EndTime

	if service.Template != nil {
		srvMap[ServiceTemplateName] = service.Template.GetName()
		srvMap[ServiceTemplateId] = service.Template.GetID()
		tempBytes, err := json.Marshal(service.Template)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Can't marshal service object to json")
		}
		srvMap[ServiceTemplateStr] = string(tempBytes)

		srvMap[ServiceRepo] = service.Template.GetRepo()
		srvMap[ServiceBranch] = service.Template.GetBranch()

		if service.Template.GetRepo() != "" && service.Template.GetBranch() != "" {
			srvMap[ServiceRepoBranch] = service.Template.GetRepo() + "_" + service.Template.GetBranch()
		}

	} else {
		srvMap[ServiceTemplateName] = ""
	}

	srvMap[ServiceArguments] = service.Arguments
	srvMap[ServiceFlags] = service.Flags
	//srvMap[ServiceArtifactTags] = service.ArtifactTags
	srvMap[ServiceStatus] = service.Status
	srvMap[ServiceStatusString] = StatusStringMap[service.Status]
	srvMap[ServiceName] = service.Name
	srvMap[ServiceDescription] = service.Description
	srvMap[ServiceMem] = service.Mem
	srvMap[ServiceCPU] = service.CPU
	srvMap[ServiceCostId] = service.CostId
	srvMap[ServiceUserName] = service.User
	if c != nil {
		srvMap[ServiceUserId] = c.User.ID
	}
	srvMap[ServiceEndpoint] = service.Endpoint

	if service.Commit != nil {

		srvMap[ServiceRevision] = service.Commit.Revision

		if _, ok := srvMap[ServiceRepoBranch]; !ok {
			srvMap[ServiceRepo] = service.Commit.Repo
			srvMap[ServiceBranch] = service.Commit.Branch

			if service.Commit.Repo != "" && service.Commit.Branch != "" {
				srvMap[ServiceRepoBranch] = service.Commit.Repo + "_" + service.Commit.Branch
			} else {
				srvMap[ServiceRepoBranch] = ""
			}
		}
	}

	if service.Parent == nil {
		srvMap[ServiceIsTask] = true
	} else {
		srvMap[ServiceIsTask] = false
		srvMap[ServiceParentId] = service.Parent.Id
	}

	if len(service.Notifications) != 0 {
		notificationsBytes, err := json.Marshal(service.Notifications)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the notifications object: %v", err))
		}
		srvMap[ServiceNotifications] = string(notificationsBytes)
	}

	if service.TerminationPolicy != nil {
		termPolicyBytes, err := json.Marshal(service.TerminationPolicy)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the termination policy object: %v", err))
		}
		srvMap[ServiceTerminationPolicy] = string(termPolicyBytes)
	}

	if service.StatusDetail != nil {
		statusDetailBytes, err := json.Marshal(service.StatusDetail)
		if err != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the status detail object: %v", err))
		}
		srvMap[ServiceStatusDetail] = string(statusDetailBytes)
	}

	commitBytes, err := json.Marshal(service.Commit)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the commit object: %v", err))
	}
	srvMap[ServiceCommit] = string(commitBytes)

	srvMap[ServicePolicyId] = service.PolicyId

	if c != nil {
		srvMap[ServiceTaskId] = c.Root.Id
	}

	srvMap[ServiceAverageWaitTime] = 10 * 1000 * 1000 // XXX convert to using DB average
	srvMap[ServiceAverageRunTime] = service.AverageRunTime * 1e6
	srvMap[ServiceAverageInitTime] = service.AverageInitTime * 1e6

	if service.Labels != nil {
		srvMap[ServiceLabels] = service.Labels
	} else {
		srvMap[ServiceLabels] = map[string]string{}
	}

	if service.Annotations != nil {
		srvMap[ServiceAnnotations] = service.Annotations
	} else {
		srvMap[ServiceAnnotations] = map[string]string{}
	}

	srvMap[ServiceIsSubmitted] = service.IsSubmitted

	if service.FailurePath != nil {
		srvMap[ServiceFailurePath] = service.FailurePath
	}

	if service.Fixtures != nil {
		fixturesMap := map[string]string{}
		for instanceID, instanceDoc := range service.Fixtures {
			instanceBytes, err := json.Marshal(instanceDoc)
			if err != nil {
				return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshall fixtures object: %v", err))
			}
			fixturesMap[instanceID] = string(instanceBytes)
		}
		srvMap[ServiceFixtures] = fixturesMap
	}

	return srvMap, nil
}

// save a new service to DB.
func (service *Service) saveNewServiceToDB(c *ServiceContext) *axerror.AXError {

	srvMap, axErr := service.CreateServiceMap(c)
	if axErr != nil {
		return axErr
	}

	_, axErr = updateServiceDB(RunningServiceTable, srvMap)

	return axErr
}

func (service *Service) LabelAll(c *ServiceContext, isPartial bool) *axerror.AXError {
	// the children contains the actual execution status of previous run
	children := service.Children
	// label the status of service to be "Skipped" if we re-submit a partial run
	// only successful steps can be skipped
	utils.InfoLog.Printf("[PartialRun]: In LabelAll(), isPartial = %t", isPartial)
	handlerStatus := func(s *Service, p *Service, name string) *axerror.AXError {
		// if the current service isn't successful, it must be re-run; as a result, its parent needs to re-run as well regardless the parent's status
		return s.SetInitialStatus1(p, c, isPartial, children)
	}
	return service.Iterate(handlerStatus, nil, "")
}

func (service *Service) PrintAll() *axerror.AXError {
	handler := func(s *Service, p *Service, n string) *axerror.AXError {
		utils.DebugLog.Printf("service: %s, id: %s, status: %d", s.Name, s.Id, s.Status)
		return nil
	}

	return service.Iterate(handler, nil, "")
}

func (service *Service) SaveAll(c *ServiceContext) *axerror.AXError {
	handler := func(s *Service, p *Service, n string) *axerror.AXError {

		if p != nil && p.Template != nil && p.Template.GetType() == template.TemplateTypeDeployment {
			// Don't persist the deployment children, it is not meaningful
			return nil
		}

		return s.saveNewServiceToDB(c)
	}
	return service.Iterate(handler, nil, "")
}

func (service *Service) UpdateUserAll(c *ServiceContext) *axerror.AXError {
	handler := func(s *Service, p *Service, n string) *axerror.AXError {
		return s.UpdateUser(c)
	}
	return service.Iterate(handler, nil, "")
}

//func (service *Service) Submit(once bool) *axerror.AXError {
//	// keep retrying until we succeed at submitting the job to adc
//	for {
//		utils.InfoLog.Printf("[JOB] service object to be sent to workflow_adc: %v", service)
//		axErr, code := utils.WorkflowAdcCl.Post2("workflows", nil, service, nil)
//		if axErr != nil {
//			utils.ErrorLog.Printf("[JOB] Failed to submit job %v/%v, (%v)error: %v", service.Id, service.Name, code, axErr)
//			if code >= 500 {
//				if !once {
//					time.Sleep(5 * time.Second)
//					continue
//				}
//			} else if code >= 400 {
//
//				detail := map[string]interface{}{
//					"code":    axErr.Code,
//					"message": axErr.Message,
//					"detail":  axErr.Detail,
//				}
//
//				if ValidateStateTransition(service.Status, utils.ServiceStatusFailed) {
//					return HandleServiceUpdate(service.Id, utils.ServiceStatusFailed, map[string]interface{}{"status_detail": detail}, event.AxEventProducer, utils.DevopsCl)
//				}
//
//				return nil
//			}
//
//		} else {
//			return service.MarkSubmitted()
//		}
//		return axErr
//	}
//}

func (service *Service) MarkSubmitted() *axerror.AXError {
	if len(service.Id) == 0 {
		return axerror.ERR_AX_INTERNAL.NewWithMessage("service doesn't have an id")
	}
	uuid, err := gocql.ParseUUID(service.Id)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage("Invalid service uuid: " + service.Id)
	}

	srvMap := make(map[string]interface{})
	srvMap[axdb.AXDBUUIDColumnName] = service.Id
	srvMap[axdb.AXDBTimeColumnName] = uuid.Time().UnixNano() / 1e3

	if service.Template != nil {
		srvMap[ServiceTemplateName] = service.Template.GetName()
	} else {
		srvMap[ServiceTemplateName] = ""
	}
	srvMap[ServiceIsSubmitted] = true

	_, axErr := updateServiceDB(RunningServiceTable, srvMap)
	if axErr != nil {
		utils.ErrorLog.Printf("Job Mark Terminated failed: %v.\n", axErr)
	}
	return axErr
}

func (service *Service) ContainsArtifactTag(tag string) bool {
	var tagList []string
	err := json.Unmarshal([]byte(service.ArtifactTags), &tagList)
	if err != nil {
		utils.InfoLog.Printf("[ARTIFACT]: unmarshal failed, err: %v", err)
		return false
	}
	utils.InfoLog.Printf("[ARTIFACT]: taglist after unmarshal %v\n", tagList)
	for _, candidate := range tagList {
		if candidate == tag {
			return true
		}
	}
	return false
}

func GetServiceByUUID(id string) (*Service, *axerror.AXError) {
	params := map[string]interface{}{
		axdb.AXDBUUIDColumnName: id,
	}

	tables := []string{RunningServiceTable, DoneServiceTable}
	var err *axerror.AXError
	var srvs []*Service
	for _, table := range tables {
		srvs, err = GetServicesFromTable(table, false, params)
		if err != nil {
			utils.ErrorLog.Printf("query table %s failure: %v", RunningServiceTable, err)
			break
		} else if len(srvs) == 1 {
			return srvs[0], nil
		} else if len(srvs) > 1 {
			utils.ErrorLog.Printf("found %d records with service_id: %s", len(srvs), id)
			break
		}
	}
	return nil, err
}

func GetServicesFromTable(
	tableName string, includeDetails bool, params map[string]interface{}) ([]*Service, *axerror.AXError) {

	var fields []string
	if params[axdb.AXDBSelectColumns] != nil {
		fields = params[axdb.AXDBSelectColumns].([]string)
		// UI still queries on 'parameters'. Change this to arguments
		// TODO: remove this when UI makes change
		for i, field := range fields {
			if field == "parameters" {
				fields[i] = "arguments"
			}
		}
		fields = append(fields, axdb.AXDBUUIDColumnName)
		fields = append(fields, axdb.AXDBTimeColumnName)
		fields = append(fields, ServiceTaskId)
		fields = append(fields, ServiceIsTask)
		fields = append(fields, ServiceLaunchTime)
		fields = append(fields, ServiceRunTime)
		fields = append(fields, ServiceWaitTime)
		fields = append(fields, ServiceEndTime)
		fields = append(fields, ServiceAverageRunTime)
		fields = append(fields, ServiceAverageWaitTime)
		fields = append(fields, ServiceStatus)
		fields = append(fields, ServiceStatusDetail)
		fields = append(fields, ServiceRepoBranch)
		fields = append(fields, ServiceJiraIssues)
		fields = DedupServiceFields(fields)
		params[axdb.AXDBSelectColumns] = fields
	}

	var tagList []string
	var needSort bool = true

	if params[axdb.AXDBQuerySearch] != nil {
		luceneSearch := params[axdb.AXDBQuerySearch].(*axdb.LuceneSearch)
		// we specified sort in lucene index search
		if luceneSearch.HasSort() {
			needSort = false
		}
	}

	copyParams := make(map[string]interface{})
	for k, v := range params {
		copyParams[k] = v
	}

	if copyParams[axdb.AXDBQueryExactSearch] != nil {
		tagList = params[axdb.AXDBQueryExactSearch].([]string)
		delete(copyParams, axdb.AXDBQueryExactSearch)
	}

	resultArray := []map[string]interface{}{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, tableName, copyParams, &resultArray)

	if axErr != nil {
		return nil, axErr
	}

	utils.InfoLog.Printf("[ARTIFACT]: num of rows %d before exact match\n", len(resultArray))
	serviceArray := []*Service{}
	for _, resultMap := range resultArray {
		var service Service
		service.InitFromMap(resultMap)
		if len(tagList) > 0 {
			for _, tag := range tagList {
				if service.ContainsArtifactTag(tag) {
					serviceArray = append(serviceArray, &service)
					break
				}
			}
		} else {
			serviceArray = append(serviceArray, &service)
		}
	}
	utils.InfoLog.Printf("[ARTIFACT]: num of rows %d after exact match\n", len(serviceArray))
	if includeDetails {
		for _, service := range serviceArray {
			details, axErr := GetServiceDetail(service.Id, fields)
			if axErr != nil {
				return nil, axErr
			}
			service.Children = details.Children
			service.AggregateChildrenCost()
		}
	}

	if needSort {
		sort.Sort(SerivceSorter(serviceArray))
	}

	return serviceArray, axErr
}

func GetServiceByArtifactTag(tag string) (*Service, *axerror.AXError) {
	utils.InfoLog.Printf("[Cross]: tag value = %s", tag)
	params := make(map[string]interface{})
	luceneSearch := axdb.NewLuceneSearch()
	params[axdb.AXDBSelectColumns] = []string{
		axdb.AXDBUUIDColumnName,
		axdb.AXDBTimeColumnName,
		ServiceTemplateName,
		ServiceTemplateStr,
		ServiceTemplateId,
		ServiceTaskId,
		ServiceIsTask,
		ServiceLaunchTime,
		ServiceRunTime,
		ServiceWaitTime,
		ServiceEndTime,
		ServiceArtifactTags,
		ServiceAverageRunTime,
		ServiceAverageWaitTime,
		ServiceStatus,
		ServiceStatusDetail,
	}
	//TODO: need exact match
	params[axdb.AXDBQueryExactSearch] = []string{tag}
	luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase(ServiceArtifactTags, "*"+tag+"*"))
	luceneSearch.AddSorter(axdb.NewLuceneSorterBase(ServiceLaunchTime, true))
	params[axdb.AXDBQuerySearch] = luceneSearch
	params[ServiceIsTask] = true
	liveServices, axErr := GetServicesFromTable(RunningServiceTable, false, params)
	if axErr != nil {
		utils.ErrorLog.Printf("failed to get services from table %s, err: %v", RunningServiceTable, axErr)
		return nil, axErr
	}
	if len(liveServices) >= 1 {
		return liveServices[0], nil
	}
	doneServices, axErr := GetServicesFromTable(DoneServiceTable, false, params)
	if axErr != nil {
		utils.ErrorLog.Printf("failed to get services from table %s, err: %v", DoneServiceTable, axErr)
		return nil, axErr
	}
	if len(doneServices) >= 1 {
		var foundService *Service = nil
		for _, doneService := range doneServices {
			// find the first workflow with status != failed && != skipped && != cancelled
			utils.InfoLog.Printf("[Cross]: launch_time %v", doneService.LaunchTime)
			if doneService.Status != utils.ServiceStatusSuccess {
				continue
			} else {
				utils.InfoLog.Printf("[Cross]: found service id %v", doneService.Id)
				foundService = doneService
				break
			}
		}
		return foundService, nil
	} else {
		utils.InfoLog.Printf("didn't find a service with the specified tag %s", tag)
		return nil, nil
	}
}

type SerivceSorter []*Service

// Len is part of sort.Interface.
func (s SerivceSorter) Len() int {
	return len(s)
}

// Swap is part of sort.Interface.
func (s SerivceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Swap is part of sort.Interface.
// DESC ordering
func (s SerivceSorter) Less(i, j int) bool {
	return s[i].CreateTime > s[j].CreateTime
}

func DedupServiceFields(old []string) []string {
	m := make(map[string]bool)

	for _, str := range old {
		str = strings.TrimSpace(str)
		m[str] = true
	}

	new := []string{}

	for k, _ := range m {
		if k != "id" {
			new = append(new, k)
		}
	}

	return new
}

type ServiceArray []*Service

func (a ServiceArray) Len() int {
	return len(a)
}
func (a ServiceArray) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ServiceArray) Less(i, j int) bool {
	// we know id is time based uuid.
	return strings.Compare(a[i].Id, a[j].Id) < 0
}

func GetServiceMapsFromDB(params map[string]interface{}) ([]map[string]interface{}, *axerror.AXError) {
	ch := make(chan interface{}, 2)
	getService := func(tableName string, c chan interface{}) {
		var resultArray []map[string]interface{}
		axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, tableName, params, &resultArray)
		if axErr != nil {
			c <- axErr
		} else {
			c <- resultArray
		}
	}

	go getService(RunningServiceTable, ch)
	go getService(DoneServiceTable, ch)

	var combinedArray []map[string]interface{}
	var axErr *axerror.AXError
	errType := reflect.TypeOf(axErr)
	for i := 0; i < 2; i++ {
		res := <-ch
		if reflect.TypeOf(res) == errType {
			return nil, res.(*axerror.AXError)
		}
		resultArray := res.([]map[string]interface{})
		if len(resultArray) == 0 {
			continue
		}
		if len(combinedArray) == 0 {
			combinedArray = resultArray
		} else {
			combinedArray = append(combinedArray, resultArray...)
		}
	}
	return combinedArray, nil
}

func GetServiceMapByID(serviceId string) (map[string]interface{}, *axerror.AXError) {

	params := map[string]interface{}{
		ServiceIsTask:           true,
		axdb.AXDBUUIDColumnName: serviceId,
	}

	if services, axErr := GetServiceMapsFromDB(params); axErr != nil {
		return nil, axErr
	} else {
		if len(services) != 0 {
			return services[0], nil
		}
	}

	return nil, nil
}

func GetServiceDetail(serviceId string, fields []string) (*Service, *axerror.AXError) {

	params := map[string]interface{}{ServiceTaskId: serviceId}

	if fields != nil {
		params[axdb.AXDBSelectColumns] = fields
	}

	var service *Service
	var children ServiceArray
	resultArray, axErr := GetServiceMapsFromDB(params)
	if axErr != nil {
		return nil, axErr
	}

	for _, resultMap := range resultArray {
		var s Service
		if resultMap[axdb.AXDBUUIDColumnName].(string) == serviceId {
			s.InitFromMap(resultMap)
			service = &s
		} else {
			s.InitFromMap(resultMap)
			children = append(children, &s)
		}
	}

	if service != nil && len(children) != 0 {
		sort.Sort(children)
		service.Children = children
		service.AggregateChildrenCost()
	}

	return service, nil
}

/*
 * Get a list of Jira associated with this specified job id
 */
func GetServiceJiraIDs(serviceId string) ([]string, *axerror.AXError) {
	var jiraIds []string
	params := map[string]interface{}{ServiceTaskId: serviceId,
		axdb.AXDBSelectColumns: []string{ServiceJiraIssues, axdb.AXDBUUIDColumnName},
	}
	resultArray, axErr := GetServiceMapsFromDB(params)
	if axErr != nil {
		return nil, axErr
	}
	for _, resultMap := range resultArray {
		if resultMap[axdb.AXDBUUIDColumnName].(string) == serviceId {
			if resultMap[ServiceJiraIssues] != nil {
				jiraIdObjs := resultMap[ServiceJiraIssues].([]interface{})
				for _, jiraIdObj := range jiraIdObjs {
					jiraIds = append(jiraIds, jiraIdObj.(string))
				}
			}
			return jiraIds, nil
		}
	}
	return nil, nil
}

// What we obtained from DB will be marshaled as float64. For those that are supposed to be integers we need to convert them.
var ConvertFloatToInt64 = func(serviceMap map[string]interface{}, fieldName string) {
	if v := serviceMap[fieldName]; v != nil {
		if _, ok := v.(float64); ok {
			serviceMap[fieldName] = int64(v.(float64))
		}
	}
}

var ConvertFloatToInt = func(serviceMap map[string]interface{}, fieldName string) {
	if v := serviceMap[fieldName]; v != nil {
		if _, ok := v.(float64); ok {
			serviceMap[fieldName] = int(v.(float64))
		}
	}
}

func updateService(currentTableName string, serviceMap map[string]interface{}, oldStatus int, axeventProducer sarama.SyncProducer, gatewayCl *restcl.RestClient) *axerror.AXError {
	if serviceMap[ServiceIsTask].(bool) && serviceMap[ServiceStatus].(int) < 0 {
		// if the job failed, we need to update the fail_path to be the path to the first failure.
		service, axErr := GetServiceDetail(serviceMap[axdb.AXDBUUIDColumnName].(string),
			[]string{axdb.AXDBUUIDColumnName, ServiceName, ServiceStatus, ServiceTemplateStr},
		)
		if axErr != nil {
			utils.ErrorLog.Printf("Getting service %v failed, error: %v", serviceMap[axdb.AXDBUUIDColumnName], axErr)
		} else {
			m := map[string]*Service{}
			for _, c := range service.Children {
				m[c.Id] = c
			}
			var path []string
			handler := func(s *Service, p *Service, name string) *axerror.AXError {
				if p == nil || m[s.Id] == nil {
					return nil
				}
				if m[s.Id].Status >= 0 {
					return nil
				}
				path = append(path, name)
				if s.Template != nil {
					if wt, ok := s.Template.(*EmbeddedWorkflowTemplate); ok && len(wt.Steps) == 0 {
						// this is bottom already, and not a static fixture, return error to break the iteration
						return axerror.ERR_AX_INTERNAL
					}
				}
				return nil
			}
			service.Iterate(handler, nil, "")
			serviceMap[ServiceFailurePath] = path
		}
	}

	axErr := UpdateServiceInDB(currentTableName, serviceMap)
	if axErr != nil {
		return axErr
	}

	handleServiceNotification(serviceMap, axeventProducer, gatewayCl)

	if serviceMap[ServiceIsTask].(bool) && serviceMap[ServiceCommit] != nil {
		var c commit.ApiCommit
		if err := json.Unmarshal([]byte(serviceMap[ServiceCommit].(string)), &c); err != nil {
			errMsg := fmt.Sprintf("Failed to unmarshal the commit string in service:%v", err)
			utils.ErrorLog.Println(errMsg)
			return axerror.ERR_AX_INTERNAL.NewWithMessage(errMsg)
		}
		if c.Revision != "" && c.Repo != "" {
			axErr := UpdateCommitJobHistory(c.Revision, c.Repo)
			if axErr != nil {
				utils.ErrorLog.Println("[CommitJobHistory]", axErr)
			}
		}
	}

	return nil
}

func UpdateServiceInDB(currentTableName string, serviceMap map[string]interface{}) *axerror.AXError {
	delete(serviceMap, axdb.AXDBWeekColumnName) // axdb really shouldn't this

	ConvertFloatToInt64(serviceMap, ServiceAverageRunTime)
	ConvertFloatToInt64(serviceMap, ServiceRunTime)
	ConvertFloatToInt64(serviceMap, ServiceAverageWaitTime)
	ConvertFloatToInt64(serviceMap, ServiceWaitTime)
	ConvertFloatToInt64(serviceMap, ServiceLaunchTime)
	ConvertFloatToInt64(serviceMap, ServiceEndTime)
	ConvertFloatToInt64(serviceMap, ServiceAverageInitTime)
	ConvertFloatToInt64(serviceMap, axdb.AXDBTimeColumnName)
	ConvertFloatToInt(serviceMap, ServiceStatus)

	newStatus := serviceMap[ServiceStatus].(int)
	if currentTableName == RunningServiceTable && newStatus <= 0 {
		// move to done table
		_, axErr := updateServiceDB(DoneServiceTable, serviceMap)
		if axErr != nil {
			utils.ErrorLog.Printf("DB request to %s table failed, err: %v", RunningServiceTable, axErr)
			return axErr
		}
		// Note a failure here can result in an entry in both running and done tables. The code that reconstructs
		// the tree is written to handle this case.
		_, axErr = utils.Dbcl.Delete(axdb.AXDBAppAXOPS, RunningServiceTable, []map[string]interface{}{serviceMap})
		if axErr != nil {
			utils.ErrorLog.Printf("DB request to %s table failed, err: %v", RunningServiceTable, axErr)
			return axErr
		}
	} else {
		// update in the current table
		_, axErr := updateServiceDB(currentTableName, serviceMap)
		if axErr != nil {
			utils.ErrorLog.Printf("DB request to %s table failed, err: %v", currentTableName, axErr)
			return axErr
		}
	}

	return nil
}

func handleServiceNotification(serviceMap map[string]interface{}, axeventProducer sarama.SyncProducer, gatewayCl *restcl.RestClient) *axerror.AXError {
	newStatus := serviceMap[ServiceStatus].(int)
	parentId := serviceMap[ServiceParentId]
	notifyStr := serviceMap[ServiceNotifications]
	if newStatus <= 0 || newStatus == utils.ServiceStatusRunning {
		if parentId == nil || parentId.(string) == axdb.AXDBNullUUID {
			if notifyStr != nil && notifyStr.(string) != "" {
				serviceID := serviceMap[axdb.AXDBUUIDColumnName].(string)
				serviceDetail, axErr := GetServiceDetail(serviceID, nil)
				if axErr != nil {
					return axErr
				}
				return serviceDetail.Notify(axeventProducer, gatewayCl)
			}

		}
	}
	return nil
}

// Allowed state transition map
var serviceStateMap map[int]map[int]int = map[int]map[int]int{
	utils.ServiceStatusInitiating: map[int]int{utils.ServiceStatusWaiting: 1, utils.ServiceStatusRunning: 1, utils.ServiceStatusCancelled: 1, utils.ServiceStatusFailed: 1, utils.ServiceStatusSuccess: 1, utils.ServiceStatusCanceling: 1},
	utils.ServiceStatusWaiting:    map[int]int{utils.ServiceStatusRunning: 1, utils.ServiceStatusCancelled: 1, utils.ServiceStatusFailed: 1, utils.ServiceStatusSuccess: 1, utils.ServiceStatusCanceling: 1},
	utils.ServiceStatusRunning:    map[int]int{utils.ServiceStatusCancelled: 1, utils.ServiceStatusFailed: 1, utils.ServiceStatusSuccess: 1, utils.ServiceStatusCanceling: 1},
	utils.ServiceStatusCanceling:  map[int]int{utils.ServiceStatusCancelled: 1, utils.ServiceStatusFailed: 1, utils.ServiceStatusSuccess: 1},
	utils.ServiceStatusCancelled:  map[int]int{},
	utils.ServiceStatusFailed:     map[int]int{},
	utils.ServiceStatusSuccess:    map[int]int{},
}

func ValidateStateTransition(old, new int) bool {
	return serviceStateMap[old][new] == 1
}

// returns whether the service has really been updated
func updateServiceInMem(serviceMap map[string]interface{}, newStatus int) bool {
	launchTime := serviceMap[ServiceLaunchTime].(float64)
	currentTime := float64(time.Now().UnixNano() / 1e3)
	if launchTime == 0 {
		launchTime = currentTime
	}
	diffTime := currentTime - launchTime
	if diffTime < 0 {
		diffTime = 0
	}
	oldStatus := serviceMap[ServiceStatus].(int)
	if serviceStateMap[oldStatus][newStatus] != 1 {
		utils.ErrorLog.Printf("invalid state change service id %v, old status %d new status %d", serviceMap[axdb.AXDBUUIDColumnName], oldStatus, newStatus)
		return false
	}

	serviceMap[ServiceStatus] = newStatus
	serviceMap[ServiceStatusString] = StatusStringMap[newStatus]
	if oldStatus == utils.ServiceStatusInitiating {
		serviceMap[ServiceLaunchTime] = currentTime
	} else if oldStatus == utils.ServiceStatusWaiting {
		serviceMap[ServiceWaitTime] = diffTime
	} else if oldStatus == utils.ServiceStatusRunning {
		serviceMap[ServiceRunTime] = diffTime - serviceMap[ServiceWaitTime].(float64)
		serviceMap[ServiceEndTime] = currentTime
	}
	return true
}

var serviceStatusMutex sync.Mutex

// Service id to status channels mapping. Each service id can have multiple channels (multiple clients interested in this service id).
// This is why we have another map that maps client context to the channel.
var serviceStatusIdChannels map[string](map[interface{}]chan *ServiceStatusEvent) = map[string](map[interface{}]chan *ServiceStatusEvent){}
var serviceStatusBranchChannels map[interface{}]*ServiceStatusBranchesChannel = map[interface{}]*ServiceStatusBranchesChannel{}

type ServiceStatusBranchesChannel struct {
	Ch     chan *ServiceStatusEvent
	Filter map[string]interface{}
}

func (c *ServiceStatusBranchesChannel) Match(repoBranch string) bool {
	if c.Filter == nil {
		return true
	}

	if repoBranch == "_" {
		return true
	}

	_, ok := c.Filter[strings.ToLower(repoBranch)]
	return ok
}

func (c *ServiceStatusBranchesChannel) AddFilter(branches []string) {
	if branches == nil || len(branches) == 0 {
		return
	}

	filter := map[string]interface{}{}
	for _, branch := range branches {
		filter[strings.ToLower(branch)] = nil
	}

	c.Filter = filter
}

/*
 * Get a channel of reading events for all tasks under the specified job id.
 */
func GetServiceStatusServiceIdChannel(ctx interface{}, serviceId string) (<-chan *ServiceStatusEvent, *axerror.AXError) {
	service, axErr := GetServiceDetail(serviceId, nil)
	if axErr != nil || service == nil {
		return nil, axErr
	}
	ch := make(chan *ServiceStatusEvent, 10)
	serviceStatusMutex.Lock()
	defer serviceStatusMutex.Unlock()

	setChannel := func(sid string) {
		utils.DebugLog.Printf("[STREAM] GetServiceStatusChannel id %s ctx %v channel %v", sid, ctx, ch)
		if serviceStatusIdChannels[sid] == nil {
			serviceStatusIdChannels[sid] = make(map[interface{}]chan *ServiceStatusEvent)
		}
		serviceStatusIdChannels[sid][ctx] = ch
	}
	setChannel(serviceId)
	for _, s := range service.Children {
		setChannel(s.Id)
	}
	utils.DebugLog.Printf("[STREAM] channel map size: %d", len(serviceStatusIdChannels))

	return ch, nil
}

func ClearServiceStatusIdChannel(ctx interface{}, serviceId string) *axerror.AXError {
	service, axErr := GetServiceDetail(serviceId, nil)
	if axErr != nil {
		return axErr
	}

	serviceStatusMutex.Lock()
	defer serviceStatusMutex.Unlock()

	clearChannel := func(sid string) {
		if serviceStatusIdChannels[sid] == nil {
			return
		}
		if serviceStatusIdChannels[sid][ctx] == nil {
			return
		}
		utils.DebugLog.Printf("[STREAM] ClearServiceStatusChannel id %s ctx %v channel %v", sid, ctx, serviceStatusIdChannels[sid][ctx])
		delete(serviceStatusIdChannels[sid], ctx)
		if len(serviceStatusIdChannels[sid]) == 0 {
			delete(serviceStatusIdChannels, sid)
		}
	}
	clearChannel(serviceId)
	if service != nil {
		for _, s := range service.Children {
			clearChannel(s.Id)
		}
	}
	utils.DebugLog.Printf("[STREAM] channel map size: %d", len(serviceStatusIdChannels))
	return nil
}

func GetServiceStatusServiceBranchChannel(ctx interface{}, branches []string) <-chan *ServiceStatusEvent {

	ch := make(chan *ServiceStatusEvent, 10)
	channel := &ServiceStatusBranchesChannel{}
	channel.Ch = ch
	channel.AddFilter(branches)

	serviceStatusMutex.Lock()
	defer serviceStatusMutex.Unlock()

	utils.DebugLog.Printf("[STREAM] GetServiceStatusBranchChannel branches %v ctx %v channel %v", branches, ctx, ch)
	serviceStatusBranchChannels[ctx] = channel

	utils.DebugLog.Printf("[STREAM] channel branch map size: %d", len(serviceStatusBranchChannels))

	return ch
}

func ClearServiceStatusBranchChannel(ctx interface{}) *axerror.AXError {
	serviceStatusMutex.Lock()
	defer serviceStatusMutex.Unlock()

	if serviceStatusBranchChannels[ctx] == nil {
		return nil
	}

	utils.DebugLog.Printf("[STREAM] ClearServiceStatusBranchChannel ctx %v channel %v", ctx, serviceStatusBranchChannels[ctx])
	delete(serviceStatusBranchChannels, ctx)

	utils.DebugLog.Printf("[STREAM] channel branch map size: %d", len(serviceStatusBranchChannels))
	return nil
}

func PostServiceStatusEvent(taskId, serviceId, repo, branch string, newStatus int) {
	event := ServiceStatusEvent{
		TaskId: taskId,
		Id:     serviceId,
		Status: newStatus,
		Repo:   repo,
		Branch: branch,
	}

	retryCount := 0
	for {
		retry := false
		serviceStatusMutex.Lock()

		for _, ch := range serviceStatusIdChannels[serviceId] {
			select {
			case ch <- &event:
				utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent task_id %v id %s repo %v branch %v status %d to channel %v", taskId, serviceId, repo, branch, newStatus, ch)
			default:
				utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent task_id %v id %s repo %v branch %v status %d to channel %v, operation failed", taskId, serviceId, repo, branch, newStatus, ch)
				retry = true
				break
			}
		}
		serviceStatusMutex.Unlock()

		if retry && retryCount < 300 {
			time.Sleep(100 * time.Millisecond)
			retryCount++
		} else {
			break
		}
	}

	retryCount = 0
	for {
		retry := false
		serviceStatusMutex.Lock()

		for _, ch := range serviceStatusBranchChannels {
			if ch != nil && ch.Match(event.Repo+"_"+event.Branch) {
				if ch.Ch != nil {
					select {
					case ch.Ch <- &event:
						utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent task_id %v id %s repo %v branch %v status %d to channel %v", taskId, serviceId, repo, branch, newStatus, ch)
					default:
						utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent task_id %v id %s repo %v branch %v status %d to channel %v, operation failed", taskId, serviceId, repo, branch, newStatus, ch)
						retry = true
						break
					}
				}
			}
		}
		serviceStatusMutex.Unlock()

		if retry && retryCount < 300 {
			time.Sleep(100 * time.Millisecond)
			retryCount++
		} else {
			break
		}
	}

	utils.DebugLog.Printf("[STREAM] channel service id map size: %d", len(serviceStatusIdChannels))
	utils.DebugLog.Printf("[STREAM] channel branch list size: %d", len(serviceStatusBranchChannels))
}

// we don't have axevent queue anymore, so notification has to post message to kafka
func HandleServiceUpdate(serviceId string, newStatus int, statusPayload map[string]interface{}, axeventProducer sarama.SyncProducer, gatewayCl *restcl.RestClient) *axerror.AXError {
	utils.InfoLog.Printf("UpdateServiceStatus id %v new status %v", serviceId, newStatus)
	var statusDetail map[string]interface{}
	if statusPayload["status_detail"] != nil {
		statusDetail = statusPayload["status_detail"].(map[string]interface{})
	}
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceId}
	currentTableName := RunningServiceTable

	// Most updates should happen on the running table. Optimize for that case
	var resultArray []map[string]interface{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, currentTableName, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", currentTableName, axErr)
		return axErr
	}
	if len(resultArray) == 0 {
		currentTableName = DoneServiceTable
		axErr = utils.Dbcl.Get(axdb.AXDBAppAXOPS, currentTableName, params, &resultArray)
		if axErr != nil {
			utils.ErrorLog.Printf("DB request to %s table failed, err: %v", currentTableName, axErr)
			return axErr
		}
	}

	if len(resultArray) == 0 {
		return nil
	}
	serviceMap := resultArray[0]
	ConvertFloatToInt(serviceMap, ServiceStatus)

	var taskId string
	if _, ok := serviceMap[ServiceTaskId]; ok {
		taskId = serviceMap[ServiceTaskId].(string)
	}

	var repo string
	if _, ok := serviceMap[ServiceRepo]; ok {
		repo = serviceMap[ServiceRepo].(string)
	}

	var branch string
	if _, ok := serviceMap[ServiceBranch]; ok {
		branch = serviceMap[ServiceBranch].(string)
	}

	statusDetailBytes, err := json.Marshal(statusDetail)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Failed to marshal the status detail object: %v", err))
	}
	serviceMap[ServiceStatusDetail] = string(statusDetailBytes)

	oldStatus := serviceMap[ServiceStatus].(int)
	changed := updateServiceInMem(serviceMap, newStatus)
	if changed {
		// update cost if needed.
		if newStatus == utils.ServiceStatusSuccess || newStatus < 0 {
			cost, axErr := GetServiceCost(serviceMap)
			if axErr == nil {
				serviceMap[ServiceCost] = cost
			}

			if serviceMap[ServiceIsTask].(bool) {
				go UpdateTemplateCost(serviceMap[ServiceTemplateId].(string), cost)
			}
		}

		axErr = updateService(currentTableName, serviceMap, oldStatus, axeventProducer, gatewayCl)
		if axErr != nil {
			return axErr
		}

		if serviceMap[ServiceIsTask].(bool) {
			axErr = UpdateTemplateJobCounts(serviceMap[ServiceTemplateId].(string), oldStatus, newStatus)
			if axErr != nil {
				return axErr
			}
		}
	} else {
		axErr = UpdateServiceInDB(currentTableName, serviceMap)
		if axErr != nil {
			return axErr
		}
	}

	PostServiceStatusEvent(taskId, serviceId, repo, branch, newStatus)
	UpdateServiceETag()

	return nil
}

func HandleServiceContainerInfoUpdate(ctn map[string]interface{}) *axerror.AXError {
	updateBytes, _ := json.Marshal(ctn)
	utils.DebugLog.Printf("Received service container update: %s", string(updateBytes))
	if val, ok := ctn[container.ContainerServiceId]; !ok || val == nil {
		utils.DebugLog.Printf("Skip update the container information to service, service ID is missing.")
		return nil
	}

	currentTable := RunningServiceTable
	serviceId := ctn[container.ContainerServiceId].(string)
	ctnId := ctn[container.ContainerId].(string)

	if msg, axErr := utils.RedisCacheCl.GetString(fmt.Sprintf(RedisServiceCtnKey, serviceId, ctnId)); axErr == nil {
		if msg == "processed" {
			utils.DebugLog.Printf("[Cache] cache hit for service with id %v, skip update the container %v information to service.\n", serviceId, ctnId)
			return nil
		}
	}

	serviceMap, axErr := getServicePKById(serviceId, currentTable)
	if serviceMap == nil {
		currentTable = DoneServiceTable
		serviceMap, axErr = getServicePKById(serviceId, currentTable)
		if serviceMap == nil {
			utils.DebugLog.Printf("Skip update the container information to service, failed to retriving service id %v, error %v", serviceId, axErr)
			return axErr
		}
	}

	if liveLog, liveLogFound := ctn[container.ContainerLogLive]; liveLogFound {
		serviceMap[ServiceLogLive] = liveLog
	}
	if doneLog, doneLogFound := ctn[container.ContainerLogDone]; doneLogFound {
		serviceMap[ServiceLogDone] = doneLog
	}
	if endpoint, ok := ctn[container.ContainerEndpoint]; ok {
		serviceMap[ServiceEndpoint] = endpoint
	}

	hostId := ctn[container.ContainerHostId].(string)
	hostName := ctn[container.ContainerHostName].(string)
	ctnId = ctn[container.ContainerId].(string)
	ctnName := ctn[container.ContainerName].(string)

	serviceMap[ServiceContainerName] = ctnName
	serviceMap[ServiceContainerId] = ctnId
	serviceMap[ServiceHostId] = hostId
	serviceMap[ServiceHostName] = hostName

	// update in the current table
	_, axErr = updateServiceDB(currentTable, serviceMap)
	if axErr != nil {
		utils.ErrorLog.Printf("DB update to %s table failed, err: %v", currentTable, axErr)
		return axErr
	}

	if currentTable == RunningServiceTable {
		serviceMapToCheck, axErr := getServiceById(serviceId, currentTable)
		if axErr != nil {
			utils.ErrorLog.Printf("DB request to %s table failed, err: %v", currentTable, axErr)
			return axErr
		}

		if serviceMapToCheck != nil {
			if val, ok := serviceMapToCheck["user_id"]; !ok || val == nil || val.(string) == "" {
				// This update created a new entry without user_id, it means it has moved to the done table.
				updateServiceDB(DoneServiceTable, serviceMap)
				if axErr = deleteServiceById(serviceId, RunningServiceTable); axErr != nil {
					return axErr
				}
			}
		}
	}

	utils.RedisCacheCl.SetWithTTL(fmt.Sprintf(RedisServiceCtnKey, serviceId, ctnId), "processed", 12*time.Hour)
	UpdateServiceETag()

	return nil
}

func getServiceById(serviceId string, serviceTable string) (map[string]interface{}, *axerror.AXError) {
	params := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceId}
	var resultArray []map[string]interface{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, serviceTable, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", serviceTable, axErr)
		return nil, axErr
	}

	if len(resultArray) != 0 {
		return resultArray[0], nil
	} else {
		return nil, nil
	}
}

func getServicePKById(serviceId string, serviceTable string) (map[string]interface{}, *axerror.AXError) {
	params := map[string]interface{}{
		axdb.AXDBUUIDColumnName: serviceId,
		axdb.AXDBSelectColumns:  []string{axdb.AXDBUUIDColumnName, ServiceTemplateName},
	}
	var resultArray []map[string]interface{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, serviceTable, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", serviceTable, axErr)
		return nil, axErr
	}

	if len(resultArray) != 0 {
		return resultArray[0], nil
	} else {
		return nil, nil
	}
}

func GetServicesSummaryFromTable(serviceTable string) ([]ServiceSummary, *axerror.AXError) {
	params := map[string]interface{}{
		axdb.AXDBSelectColumns: []string{axdb.AXDBUUIDColumnName, ServiceHostId, ServiceName, ServiceStatus, ServiceUserName, ServiceRunTime, ServiceAverageRunTime},
	}

	resultArray := []ServiceSummary{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, serviceTable, params, &resultArray)

	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s, table failed, err: %v", serviceTable, axErr)
		return nil, axErr

	}

	return resultArray, nil
}

func getServicesByHost(hostId string, serviceTable string) ([]ServiceSummary, *axerror.AXError) {
	params := map[string]interface{}{
		ServiceHostId:          hostId,
		axdb.AXDBSelectColumns: []string{axdb.AXDBUUIDColumnName, ServiceName, ServiceStatus, ServiceUserName, ServiceRunTime, ServiceAverageRunTime},
	}

	resultArray := []ServiceSummary{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, serviceTable, params, &resultArray)
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s, table failed, err: %v", serviceTable, axErr)
		return nil, axErr

	}

	return resultArray, nil
}

func deleteServiceById(serviceId string, serviceTable string) *axerror.AXError {
	serviceMap := map[string]interface{}{axdb.AXDBUUIDColumnName: serviceId}
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, serviceTable, []map[string]interface{}{serviceMap})
	if axErr != nil {
		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", serviceTable, axErr)
		return axErr
	}
	return nil
}

func getTaskCost(taskId string) (float64, *axerror.AXError) {
	params := map[string]interface{}{ServiceTaskId: taskId}
	resultArray, axErr := GetServiceMapsFromDB(params)
	if axErr != nil {
		return 0.0, axErr
	}

	var totalCost float64 = 0.0
	for _, resultMap := range resultArray {
		cost, _ := resultMap[ServiceCost].(float64)
		totalCost += cost
	}
	utils.DebugLog.Printf("task cost is %f ", totalCost)
	return totalCost, nil
}

// Get the service cost in dollars
func GetServiceCost(serviceMap map[string]interface{}) (float64, *axerror.AXError) {
	serviceId, ok := serviceMap[axdb.AXDBUUIDColumnName].(string)
	if !ok {
		return 0.0, axerror.ERR_AX_INTERNAL.NewWithMessage("can't get service cost with nil Id")
	}
	if serviceMap[ServiceIsTask].(bool) {
		return getTaskCost(serviceId)
	}

	tmpl, axErr := UnmarshalEmbeddedTemplate([]byte(serviceMap[ServiceTemplateStr].(string)))
	if axErr != nil {
		return 0.0, axerror.ERR_AXDB_INTERNAL.NewWithMessage(fmt.Sprintf("Can't unmarshal template from string: %s", serviceMap[ServiceTemplateStr]))
	}

	if tmpl.GetType() == template.TemplateTypeContainer {
		ct := tmpl.(*EmbeddedContainerTemplate)
		//modelPrice, axErr := host.GetAveragePrice(false)
		//if modelPrice == nil {
		//	return 0.0, axErr
		//}

		cpuCores := 0.0
		memMiB := 0.0
		if ct.Resources != nil {
			cpuCores, _ = ct.Resources.CPUCoresValue()
			memMiB, _ = ct.Resources.MemMiBValue()
		}

		var runTime float64
		if serviceMap[ServiceRunTime] != nil {
			runTime = serviceMap[ServiceRunTime].(float64)
		}

		//} else {
		//	utils.DebugLog.Printf("Runtime is missing for service %v, estimating based on container usage entries.\n", serviceId)
		//	// we query using cost_id and the time the service started running. The problem right now is that we don't
		//	// have a table view that uses service_id as partition key.
		//	_ = "breakpoint"
		//	costId := serviceMap[ServiceCostId]
		//	if costId == nil {
		//		return 0.0, nil
		//	}
		//	launchTime := serviceMap[ServiceLaunchTime].(float64)
		//	var waitTime float64 = 0
		//	if w := serviceMap[ServiceWaitTime]; w != nil {
		//		waitTime = w.(float64)
		//	}
		//	startTime := launchTime + waitTime
		//
		//	params := map[string]interface{}{
		//		usage.ContainerUsageCostId: costId,
		//		axdb.AXDBQueryMinTime:      int64(startTime),
		//	}
		//	var resultArray []map[string]interface{}
		//	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, usage.ContainerUsageTable, params, &resultArray)
		//	if axErr != nil {
		//		utils.ErrorLog.Printf("DB request to %s table failed, err: %v", usage.ContainerUsageTable, axErr)
		//		return 0.0, axErr
		//	}
		//
		//	var count float64
		//	for _, usageMap := range resultArray {
		//		if usageMap[usage.ContainerUsageServiceId] == serviceId {
		//			count++
		//		}
		//	}
		//
		//	runTime = count * utils.SpendingInterval
		//}

		cost := GetSpendingCents(cpuCores, memMiB, runTime)
		return cost, nil
	}

	return 0, nil
}

func GetSpendingCents(cpuCores, memMiB, runTime float64) float64 {
	modelPrice, axErr := host.GetAveragePrice(true)
	if axErr != nil || modelPrice == nil {
		return 0.0
	}
	cpuCost := cpuCores * runTime / 1e6 * modelPrice.CoreCost
	memCost := memMiB * runTime / 1e6 * modelPrice.MemCost
	if memCost > cpuCost {
		return memCost
	}
	return cpuCost
}

type Summary struct {
	Cluster       string
	Status        string
	Code          string
	Message       string
	Detail        string
	Name          string
	TemplateName  string
	Owner         string
	ShortRepo     string
	Repo          string
	Branch        string
	Revision      string
	Description   string
	ShortRevision string
	Committer     string
	Author        string
	Submitter     string
	Runtime       string
	Link          string
	StatusNum     int
}

var PolicyHistory map[string]int = map[string]int{}

func (s *Service) Notify(axeventProducer sarama.SyncProducer, gatewayCl *restcl.RestClient) *axerror.AXError {

	if len(s.Notifications) == 0 {
		return nil
	}

	newStatus := s.Status

	//serviceInfo := s.Summary()
	//serviceDetailBytes, err := json.MarshalIndent(serviceInfo, "", "  ")
	//if err != nil {
	//	return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Marshal service object failed:%v", err))
	//}
	//
	//serviceDetailStr := string(serviceDetailBytes)

	summary := Summary{
		Cluster:      common.GetPublicDNS(),
		Name:         s.Name,
		Submitter:    s.User,
		Link:         "https://" + common.GetPublicDNS() + "/app/jobs/job-details/" + s.Id,
		Runtime:      strconv.Itoa(int(s.RunTime)/60) + " minutes " + strconv.Itoa(int(s.RunTime)%60) + " seconds",
		Status:       StatusStringMap[newStatus],
		TemplateName: s.Template.GetName(),
		StatusNum:    newStatus,
	}

	if s.PolicyId != "" {
		if p, _ := policy.GetPolicyByID(s.PolicyId); p != nil {
			summary.Submitter = "https://" + common.GetPublicDNS() + "/app/policies/details/" + p.ID
		}
	}

	if s.Commit != nil {
		utils.DebugLog.Println(s.Commit)
		owner, repo := utils.ParseRepoURL(s.Commit.Repo)
		summary.Owner = owner
		summary.ShortRepo = repo
		summary.Repo = s.Commit.Repo
		summary.Branch = s.Commit.Branch
		summary.Author = s.Commit.Author
		summary.Committer = s.Commit.Committer
		summary.Revision = s.Commit.Revision
		summary.Description = s.Commit.Description
		if len(s.Commit.Revision) > 7 {
			summary.ShortRevision = s.Commit.Revision[:7]
		} else {
			summary.ShortRevision = s.Commit.Revision
		}
	} else {
		summary.Owner = "N/A"
		summary.ShortRepo = "N/A"
		summary.Repo = "N/A"
		summary.Branch = "N/A"
		summary.Author = "N/A"
		summary.Committer = "N/A"
		summary.Revision = "N/A"
		summary.Description = "N/A"
		summary.ShortRevision = "N/A"
	}

	if s.StatusDetail != nil {
		if val, ok := s.StatusDetail["code"]; ok {
			summary.Code = val.(string)
		}

		if val, ok := s.StatusDetail["message"]; ok {
			summary.Message = val.(string)
		}

		if val, ok := s.StatusDetail["detail"]; ok {
			summary.Detail = val.(string)
		}
	}

	for _, n := range s.Notifications {
		notify := false
		for _, when := range n.When {
			switch strings.ToLower(when) {
			case utils.ServiceEventOnFailure:
				if newStatus == utils.ServiceStatusFailed {
					notify = true
				}
			case utils.ServiceEventOnSuccess:
				if newStatus == utils.ServiceStatusSuccess {
					notify = true
				}
			case utils.ServiceEventOnStart:
				if newStatus == utils.ServiceStatusRunning {
					notify = true
				}
			case utils.ServiceEventOnCompletion:
				if newStatus <= 0 {
					notify = true
				}
			case utils.ServiceEventOnChange:
				if s.PolicyId != "" && s.Commit != nil {
					switch newStatus {
					case utils.ServiceStatusSuccess, utils.ServiceStatusFailed:
						key := s.PolicyId + s.Commit.Repo + s.Commit.Branch
						if oldStatus, ok := PolicyHistory[key]; ok {
							if oldStatus != newStatus {
								if newStatus == utils.ServiceStatusSuccess {
									notify = true
								}
								if newStatus == utils.ServiceStatusFailed {
									notify = true
								}
							}
						}
						PolicyHistory[key] = newStatus
					}
				}
			default:
				panic(fmt.Sprintf("unexpected event type %s", when))
			}
		}

		if notify {
			newWhom := []string{}
			whom := n.Whom
			for _, who := range whom {
				if utils.ValidateEmail(who) {
					newWhom = append(newWhom, who)
				} else {
					who := strings.ToLower(who)
					switch who {
					case label.UserLabelAuthor:
						common.InfoLog.Println("Has Author")
						if s.Commit != nil && s.Commit.Author != "" {
							common.InfoLog.Printf("Author:%v\n", s.Commit.Author)
							email := utils.GetUserEmail(s.Commit.Author)
							if email != "" {
								newWhom = append(newWhom, email)
							} else {
								utils.InfoLog.Printf("Cannot get valid email from author attribute: %s.", s.Commit.Author)
							}
						}
					case label.UserLabelCommitter:
						common.InfoLog.Println("Has Committer")
						if s.Commit != nil && s.Commit.Committer != "" {
							common.InfoLog.Printf("Committer:%v\n", s.Commit.Committer)
							email := utils.GetUserEmail(s.Commit.Committer)
							if email != "" {
								newWhom = append(newWhom, email)
							} else {
								utils.InfoLog.Printf("Cannot get valid email from committer attribute: %s.", s.Commit.Committer)
							}
						}
					case label.UserLabelSubmitter:
						if s.User != "" && s.User != "admin@internal" && s.User != "system" {
							if utils.ValidateEmail(s.User) {
								newWhom = append(newWhom, s.User)
							} else {
								utils.InfoLog.Printf("Cannot get valid email from user attribute: %s.", s.User)
							}
						}
					case label.UserLabelSCM:
						newWhom = append(newWhom, label.UserLabelSCM)
					case label.UserLabelFixtureManager:
						newWhom = append(newWhom, label.UserLabelFixtureManager)
					default:
						if strings.HasSuffix(who, "@slack") {
							newWhom = append(newWhom, who)
						} else {
							users, axErr := user.GetUsersByLabel(who)
							if axErr != nil {
								utils.ErrorLog.Printf("Failed to load users with label %v: %v.\n", who, axErr)
								continue
							}

							utils.DebugLog.Printf("Find %v users with label %v.\n", len(users), who)
							if len(users) != 0 {
								for _, u := range users {
									newWhom = append(newWhom, u.Username)
								}
							}
						}
					}
				}
			}

			n.Whom = utils.DedupStringList(newWhom)
			if len(n.Whom) == 1 && n.Whom[0] == label.UserLabelSCM {
				payload := map[string]interface{}{
					"id":     s.Id,
					"status": s.Status,
				}

				_, axErr := gatewayCl.Post("scm/reports", payload)
				if axErr != nil {
					utils.ErrorLog.Println("Failed to report to SCM:", axErr)
				}
			} else {
				fixtureManager, newWhom := popElement(label.UserLabelFixtureManager, n.Whom)
				if fixtureManager != nil {
					payload := map[string]interface{}{
						"id":          s.Id,
						"name":        s.Name,
						"user":        s.User,
						"status":      s.Status,
						"annotations": s.Annotations,
					}
					utils.InfoLog.Printf("Notifying fixturemanager of job %s (name: %s, status: %d, annotations: %s)", s.Id, s.Name, s.Status, s.Annotations)
					_, axErr := fixMgrCl.Post("v1/fixture/action_result", payload)
					if axErr != nil {
						utils.ErrorLog.Println("Failed to report to fixturemanager:", axErr)
					}
					n.Whom = newWhom
				}
				// send notification using notification center
				sendJobStatusToNotificationCenter(&summary, n.Whom)
			}
		}
	}
	return nil
}

// popElement searches a slice of strings for a item, and returns the found element and a new slice with the element removed
func popElement(needle string, haystack []string) (*string, []string) {
	var found *string
	newHaystack := make([]string, 0)
	for _, item := range haystack {
		if item == needle {
			found = &item
		} else {
			newHaystack = append(newHaystack, item)
		}
	}
	return found, newHaystack
}

func sendJobStatusToNotificationCenter(summary *Summary, recipients []string) {

	common.InfoLog.Printf("Recipients:%v\n", recipients)

	var code string

	detail := map[string]interface{}{}

	detail["Job name"] = summary.Name
	detail["Status"] = summary.Status
	detail["Job Detail"] = summary.Link
	detail["Triggered by"] = summary.Submitter
	detail["Repo:Branch"] = summary.ShortRepo + ":" + summary.Branch
	detail["Committer"] = summary.Committer

	switch summary.StatusNum {
	case utils.ServiceStatusRunning:
		code = notification_center.CodeJobStatusStarted
	case utils.ServiceStatusSuccess:
		code = notification_center.CodeJobStatusSuccess
	case utils.ServiceStatusFailed:
		code = notification_center.CodeJobStatusFailed
	default:
		utils.ErrorLog.Println("Skip notification, unexpected status:", summary.StatusNum)
		return
	}

	notification_center.Producer.SendMessage(code, "", recipients, detail)

}

func (s *Service) Summary() map[string]interface{} {
	summary := map[string]interface{}{}

	summary["id"] = s.Id
	summary["name"] = s.Name
	summary["description"] = s.Description
	summary["commit"] = s.Commit
	summary["ctime"] = s.CreateTime
	summary["runtime"] = strconv.Itoa(int(s.RunTime)/60) + " minutes " + strconv.Itoa(int(s.RunTime)%60) + " seconds"
	summary["status"] = StatusStringMap[s.Status]
	summary["link"] = "https://" + common.GetPublicDNS() + "/app/jobs/job-details/" + s.Id + ";tab=workflow"

	if len(s.Children) != 0 {
		subTasks := map[string]interface{}{}
		for _, child := range s.Children {
			if child != nil {
				stepName := s.GetStepNameByID(child.Id)
				if stepName != "" {
					subTasks[stepName] = StatusStringMap[child.Status]
				}
			}
		}
		summary["subtasks"] = subTasks
	}

	return summary
}

func (s *Service) JobSummary() *commit.JobSummary {
	job := &commit.JobSummary{
		Name:           s.Name,
		Status:         s.Status,
		StartTime:      s.CreateTime,
		CreateTime:     s.CreateTime,
		RunTime:        s.RunTime,
		AverageRunTime: s.AverageRunTime,
		LaunchTime:     s.LaunchTime,
		EndTime:        s.EndTime,
		WaitTime:       s.WaitTime,
	}
	return job
}

// This might be expensive, we may add the step name to the service object
func (s *Service) GetStepNameByID(id string) string {
	if s.Template == nil {
		return ""
	}
	wt, ok := s.Template.(*EmbeddedWorkflowTemplate)
	if !ok {
		return ""
	}
	for _, parallelSteps := range wt.Steps {
		for name, step := range parallelSteps {
			if step.Id == id {
				return name
			}

			if stepName := step.GetStepNameByID(id); stepName != "" {
				return stepName
			}
		}
	}
	return ""
}

func (s *Service) GetStepServiceByPath(path []string, isRoot bool) *Service {
	if s == nil {
		return nil
	} else if len(path) == 0 {
		if isRoot {
			return s
		}
		return nil
	} else if s.Template == nil {
		return nil
	}
	wt, ok := s.Template.(*EmbeddedWorkflowTemplate)
	if !ok {
		return nil
	}
	step := path[0]
	for _, parallelSteps := range wt.Steps {
		for name, svc := range parallelSteps {
			if name == step {
				return svc.GetStepServiceByPath(path[1:], false)
			}
		}
	}

	for _, parallelFixtures := range wt.Fixtures {
		for name, fix := range parallelFixtures {
			if name == step {
				return fix.Service.GetStepServiceByPath(path[1:], false)
			}
		}
	}
	return nil
}
