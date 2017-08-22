package deployment

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"applatix.io/axamm/heartbeat"
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"applatix.io/axops/service"
	"applatix.io/axops/tool"
	"applatix.io/axops/volume"
	"applatix.io/common"
	"applatix.io/lock"
	"applatix.io/template"
)

var DeployLockGroup lock.LockGroup

func InitLock() {
	DeployLockGroup.Name = "DeployLock"
	DeployLockGroup.Init()
}

func Init(app string) *axerror.AXError {

	InitLock()

	params := map[string]interface{}{}
	params[DeploymentAppName] = app

	deployments, axErr := GetLatestDeployments(params, false)
	if axErr != nil {
		return axErr
	}

	for _, d := range deployments {
		switch d.Status {
		case DeployStateTerminating:
		case DeployStateTerminated:
		default:
			heartbeat.RegisterHandler(d.HeartBeatKey(), GetHeartBeatHandler())
			// register previous deployment as well
			if len(d.PreviousDeploymentId) > 0 {
				prev_d, axErr := GetHistoryDeploymentByID(d.PreviousDeploymentId, false)
				if axErr != nil {
					return axErr
				}
				heartbeat.RegisterHandler(prev_d.HeartBeatKey(), GetHeartBeatHandler())
			}

		}
	}

	go monitorDeployments()

	return nil
}

type RedisDeploymentResult struct {
	Id              string                 `json:"id,omitempty"`
	Name            string                 `json:"name,omitempty"`
	ApplicationName string                 `json:"app_name,omitempty"`
	Status          string                 `json:"status,omitemtpy"`
	StatusDetail    map[string]interface{} `json:"status_detail,omitempty"`
}

func (r *RedisDeploymentResult) String() string {
	jsonBytes, _ := json.Marshal(r)
	return string(jsonBytes)
}

type Domain struct {
	Name string `json:"name"`
}

type DomainModel struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Type       string   `json:"type"`
	Password   string   `json:"password"`
	Domains    []Domain `json:"domains,omitempty"`
	AllDomains []string `json:"all_domains,omitempty"`
}

type DomainData struct {
	Data []*DomainModel `json:"data"`
}

func ValidateDeploymentName(name string) bool {
	re := regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?)$`)
	return re.MatchString(name)
}

func (d *Deployment) PostProcess() *axerror.AXError {

	if !ValidateDeploymentName(d.Template.DeploymentName) {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The deployment name %v is invalid: expect the format ^([a-z0-9]([-a-z0-9]*[a-z0-9])?)$.", d.Template.DeploymentName)
	}

	if len(d.Template.DeploymentName) > 63 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The deployment name %v can not be more than 63 characters.", d.Template.DeploymentName)
	}

	d.ApplicationName = d.Template.ApplicationName
	d.Name = d.Template.DeploymentName
	d.ApplicationID = common.GenerateUUIDv5(d.ApplicationName)
	d.DeploymentID = common.GenerateUUIDv5(d.ApplicationName + "$$" + d.Name)
	d.TerminationPolicy = d.Template.TerminationPolicy

	if d.Description == "" && d.Template != nil {
		d.Description = d.Template.Description
	}

	d.CostId = map[string]interface{}{
		"app":     d.ApplicationName,
		"service": d.Name,
		"user":    d.User,
	}

	for ctnName := range d.Template.Containers {
		d.Template.Containers[ctnName].CostId = d.CostId
	}

	if d.Template.InternalRoutes != nil {
		if axErr := d.Template.InternalRoutes.Validate(); axErr != nil {
			return axErr
		}
	}

	if d.Template.ExternalRoutes != nil {

		var domains DomainData
		if axErr := utils.AxopsCl.Get("tools", map[string]interface{}{"type": tool.TypeRoute53}, &domains); axErr != nil {
			return axErr
		}

		for _, route := range d.Template.ExternalRoutes {
			route.DNSPrefix = strings.TrimSpace(route.DNSPrefix)
			if route.DNSPrefix == "" {
				route.DNSPrefix = fmt.Sprintf("%v-%v", strings.ToLower(d.ApplicationName), strings.ToLower(d.Name))
			}

			if len(domains.Data) == 0 || len(domains.Data[0].Domains) == 0 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("No domain is available in cluster for external route, please configure the domain in configuration page.")
			}

			route.DNSDomain = strings.TrimSpace(route.DNSDomain)
			if route.DNSDomain == "" {
				// if user did not specify a DNSDomain in his template, we choose the first one returned in the list
				route.DNSDomain = domains.Data[0].Domains[0].Name
			} else {
				found := false
				common.DebugLog.Println("[Domain]", route.DNSDomain)
				for _, domain := range domains.Data[0].Domains {
					common.DebugLog.Println("[Domain]", domain)
					if domain.Name == route.DNSDomain || domain.Name == route.DNSDomain+"." {
						found = true
						break
					}
				}
				if !found {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("DNS domain %v is not configured in this cluster, please configure it in Domain Management or use another domain.", route.DNSDomain)
				}
			}
			route.DNSName = route.DNSPrefix + "." + route.DNSDomain
			d.Endpoints = append(d.Endpoints, route.DNSName)
		}

		if axErr := d.Template.ExternalRoutes.Validate(); axErr != nil {
			return axErr
		}
	}

	if d.Template.Volumes != nil {
		for name, vol := range d.Template.Volumes {
			if vol.Name == "" {
				// Anonymous volume
				if vol.StorageClass == "" {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Storage class can not be empty for annoymous volume.")
				}

				class, axErr := volume.GetStorageClassByName(vol.StorageClass)
				if axErr != nil {
					return axErr
				}

				if class == nil {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Cannot find annoymous volume %v storage class %v information.", name, vol.StorageClass)
				}

				//class.Axrn = fmt.Sprintf("vol:/anonymous/%v/%v/%v", d.ApplicationName, d.Name, name)
				// TODO: revisit -Jesse
				//vol.Details = class
			} else {
				// TODO: revisit -Jesse
				//vol.Axrn = fmt.Sprintf("vol:/%v", vol.Name)
			}
		}
	}

	if len(d.Template.Volumes) != 0 {
		if d.Template.Scale.Min > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Deployment with volumes to have more than one instance is not supported. Please check the scale setting.")
		}
	}

	if d.Template.Containers != nil {
		for _, ctn := range d.Template.Containers {
			inputs := ctn.Template.GetInputs()
			if inputs != nil {
				for _ = range inputs.Volumes {
					// TODO: revisit -Jesse
					// mount.Volume = strings.TrimSpace(mount.Volume)
					// mount.Volume = strings.TrimPrefix(mount.Volume, "%%")
					// mount.Volume = strings.TrimSuffix(mount.Volume, "%%")
					// if strings.HasPrefix(mount.Volume, "volumes") {
					// 	args := strings.Split(mount.Volume, ".")
					// 	mount.Volume = args[1]
					// }
				}
			}
		}
	}

	// rolling update is not supported for deployment with volumes or fixtures
	if d.Template.Strategy != nil && d.Template.Strategy.Type == template.StrategyRollingUpdate {
		if len(d.Template.Volumes) != 0 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Rolling Update is not supported for deployment with volumes")
		}
	}

	currentDeployment, axErr := GetLatestDeploymentByName(d.ApplicationName, d.Name, false)
	if axErr != nil {
		return axErr
	}
	if currentDeployment != nil {

		if !d.fixturesAreSame(currentDeployment) {
			utils.DebugLog.Printf("Fixture specs are different. old:%v\n new:%v\n", currentDeployment.Template.Fixtures, d.Template.Fixtures)
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Cannot update deployment since the fixture specs are different")
		}
		if !d.volumesAreSame(currentDeployment) {
			utils.DebugLog.Printf("Volume specs are different. old:%v\n new:%v\n", currentDeployment.Template.Volumes, d.Template.Volumes)
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Cannot update deployment since the volume specs are different")
		}
		if !reflect.DeepEqual(d.Template.ExternalRoutes, currentDeployment.Template.ExternalRoutes) {
			utils.DebugLog.Printf("External routes are different. old:%v\n new:%v\n", currentDeployment.Template.ExternalRoutes, d.Template.ExternalRoutes)
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Cannot update deployment since the external route specs are different")
		}
		if !reflect.DeepEqual(d.Template.InternalRoutes, currentDeployment.Template.InternalRoutes) {
			utils.DebugLog.Printf("Internal routes are different. old:%v\n new:%v\n", currentDeployment.Template.InternalRoutes, d.Template.InternalRoutes)
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Cannot update deployment since the internal route specs are different")
		}
	}

	return nil
}

func (d *Deployment) fixturesAreSame(o *Deployment) bool {
	if len(d.Template.Fixtures) != len(d.Template.Fixtures) {
		utils.DebugLog.Printf("unequal fixture lengths: %v, %v", len(d.Template.Fixtures), len(d.Template.Fixtures))
		return false
	}
	for i, parallelFixtures := range d.Template.Fixtures {
		if len(parallelFixtures) != len(o.Template.Fixtures[i]) {
			utils.DebugLog.Printf("unequal parallel fixture lengths: %v, %v", len(parallelFixtures), len(o.Template.Fixtures[i]))
			return false
		}
		for fixName, fixture := range parallelFixtures {
			vo, exists := o.Template.Fixtures[i][fixName]
			if !exists {
				utils.DebugLog.Printf("%s does not exist in previous request", fixName)
				return false
			}
			if !fixture.Equals(vo.FixtureRequirement) {
				utils.DebugLog.Printf("fixture requirements different: %v != %v", fixture, vo.FixtureRequirement)
				return false
			}
		}
	}
	return true
}

func (d *Deployment) volumesAreSame(o *Deployment) bool {
	if len(d.Template.Volumes) != len(d.Template.Volumes) {
		return false
	}
	for k, v := range d.Template.Volumes {
		if vo, exists := o.Template.Volumes[k]; exists {
			if !vo.Equals(*v) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (d *Deployment) Substitute() (*Deployment, *axerror.AXError) {
	// Substituted from top down
	dBytes, err := json.Marshal(d)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}
	dStr := string(dBytes)

	if d.Fixtures != nil {
		utils.DebugLog.Println("Before fixture substituted Deployment:", dStr)
		for name, fixes := range d.Fixtures {
			for key, val := range fixes {
				if val != nil {
					utils.DebugLog.Printf("Subsituting fixture parameter: %v %v %v", name, key, val)
					replaceVal := fmt.Sprintf("%v", val)
					dStr = strings.Replace(dStr, "%%fixtures."+name+"."+key+"%%", replaceVal, -1)
				}
			}
		}
		utils.DebugLog.Println("After fixture substituted Deployment:", dStr)
	}

	var copy Deployment
	err = json.Unmarshal([]byte(dStr), &copy)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	newTmpl, axErr := copy.Template.SubstituteArguments(copy.Arguments)
	if axErr != nil {
		return nil, axErr
	}
	copy.Template = newTmpl.(*service.EmbeddedDeploymentTemplate)
	return &copy, nil
}

func (d *Deployment) PreProcess() *axerror.AXError {

	d.Status = DeployStateInit
	d.StatusDetail = map[string]interface{}{}
	d.getMaxResources()

	if d.Id == "" {
		d.Id = common.GenerateUUIDv1()
		common.DebugLog.Printf("Generated ID %v for deployment %v/%v.\n", d.Id, d.ApplicationName, d.Name)
	}

	if d.Arguments == nil {
		d.Arguments = make(template.Arguments)
	}

	if d.Template.Labels == nil {
		d.Template.Labels = map[string]string{}
	}

	// TODO: revisit -Jesse
	// if d.Template.Annotations == nil {
	// 	d.Template.Annotations = map[string]string{}
	// }

	if d.Template.Scale == nil {
		d.Template.Scale = &template.Scale{
			Min: 1,
		}
	}

	/* TODO: revisit -Jesse
	for _, ctr := range d.Template.Containers {
		if ctr.Template.Labels == nil {
			ctr.Template.Labels = map[string]string{}
		}

		//if ctr.Template.Annotations == nil {
		//	ctr.Template.Annotations = map[string]string{}
		//}

		if ctr.Template.Inputs != nil && ctr.Template.Inputs.Parameters != nil {
			if ctr.Parameters == nil {
				ctr.Parameters = map[string]interface{}{}
			}

			for key := range ctr.Template.Inputs.Parameters {
				if val, ok := ctr.Parameters[key]; !ok || val.(string) == "" {
					ctr.Parameters[key] = "%%" + key + "%%"
				}
			}

		}
	}
	*/

	if d.Template.Labels != nil {
		d.Labels = d.Template.Labels
	}

	// TODO: revisit -Jesse
	// if d.Template.Annotations != nil {
	// 	d.Annotations = d.Template.Annotations
	// }

	d.CreateTime = 0
	d.LaunchTime = 0
	d.RunTime = 0
	d.WaitTime = 0
	d.EndTime = 0

	if d.Description == "" && d.Template != nil {
		d.Description = d.Template.Description
	}

	return nil
}
