package template

import (
	"strconv"
	"strings"

	"applatix.io/axerror"
	"applatix.io/common"
)

const (
	// External route options
	ExternalRouteVisibilityWorld = "world"
	ExternalRouteVisibilityOrg   = "organization"
	// Upgrade Strategies
	StrategyRollingUpdate = "rolling_update"
	StrategyRecreate      = "recreate"
)

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type DeploymentTemplate struct {
	BaseTemplate      `json:",squash"`
	Inputs            *Inputs                               `json:"inputs,omitempty"`
	ApplicationName   string                                `json:"application_name,omitempty"`
	DeploymentName    string                                `json:"deployment_name,omitempty"`
	Scale             *Scale                                `json:"scale,omitempty"`
	ExternalRoutes    ExternalRoutes                        `json:"external_routes,omitempty"`
	InternalRoutes    InternalRoutes                        `json:"internal_routes,omitempty"`
	Containers        map[string]InlineContainerTemplateRef `json:"containers,omitempty"`
	Fixtures          FixtureRequirements                   `json:"fixtures,omitempty"`
	Volumes           VolumeRequirements                    `json:"volumes,omitempty"`
	TerminationPolicy *TerminationPolicy                    `json:"termination_policy,omitempty"`
	MinReadySeconds   int                                   `json:"min_ready_seconds,omitempty"`
	Strategy          *Strategy                             `json:"strategy,omitempty"`
}

type Scale struct {
	Min int `json:"min,omitempty"`
	Max int `json:"max,omitempty"`
}

type ExternalRoutes []*ExternalRoute

type ExternalRoute struct {
	DNSPrefix   string   `json:"dns_prefix,omitempty"`
	DNSDomain   string   `json:"dns_domain,omitempty"`
	DNSName     string   `json:"dns_name,omitempty"`
	TargetPort  string   `json:"target_port,omitempty"`
	IPWhiteList []string `json:"ip_white_list,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
}

type InternalRoutes []*InternalRoute

type InternalRoute struct {
	Name  string  `json:"name,omitempty"`
	Ports []*Port `json:"ports,omitempty"`
}

type Port struct {
	Port       string `json:"port,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}

type Strategy struct {
	Type          string                 `json:"type,omitempty"`
	RollingUpdate *RollingUpdateStrategy `json:"rolling_update,omitempty"`
}

type RollingUpdateStrategy struct {
	MaxSurge       string `json:"max_surge"`
	MaxUnavailable string `json:"max_unavailable"`
}

func (tmpl *DeploymentTemplate) GetInputs() *Inputs {
	return tmpl.Inputs
}

func (tmpl *DeploymentTemplate) GetOutputs() *Outputs {
	return nil
}

func (tmpl *DeploymentTemplate) Validate(preproc ...bool) *axerror.AXError {
	preprocessing := len(preproc) > 0 && preproc[0]
	if tmpl.Name == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("'name' field missing or empty")
	}
	// TODO: validate kubernetes name
	tmpl.ApplicationName = strings.TrimSpace(tmpl.ApplicationName)
	if tmpl.ApplicationName == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("'application' field missing or empty")
	}
	tmpl.DeploymentName = strings.TrimSpace(tmpl.DeploymentName)
	if tmpl.DeploymentName == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("'deployment' field missing or empty")
	}
	axErr := tmpl.Inputs.Validate(false)
	if axErr != nil {
		return axErr
	}
	axErr = tmpl.Fixtures.Validate()
	if axErr != nil {
		return axErr
	}
	if tmpl.Scale == nil {
		tmpl.Scale = &Scale{
			Min: 1,
		}
	}
	if tmpl.Volumes != nil {
		if len(tmpl.Volumes) > 0 && tmpl.Scale.Min > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("%s.scale: deployment requiring volumes cannot scale past 1 instance", tmpl.Name)
		}
		axErr := tmpl.Volumes.Validate()
		if axErr != nil {
			return axErr
		}
	}
	if tmpl.InternalRoutes != nil {
		axErr := tmpl.InternalRoutes.Validate()
		if axErr != nil {
			return axErr
		}
	}
	if tmpl.ExternalRoutes != nil {
		axErr := tmpl.ExternalRoutes.Validate()
		if axErr != nil {
			return axErr
		}
	}
	if tmpl.Fixtures != nil {
		for i, parallelFixtures := range tmpl.Fixtures {
			for fixRefName, ftr := range parallelFixtures {
				if ftr.IsDynamicFixture() {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("fixtures[%d].%s: deployments cannot use dynamic fixtures", i, fixRefName)
				}
			}
		}
	}
	if tmpl.Strategy != nil && tmpl.Strategy.Type == StrategyRollingUpdate {
		if len(tmpl.Volumes) > 0 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy type \"rolling_update\" is not supported for deployments with volume")
		}
	}
	// run Validate against any inlined containers
	for refName, ctRef := range tmpl.Containers {
		if ctRef.Inlined() {
			axErr := ctRef.Validate()
			if axErr != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("containers.%s: %v", refName, axErr)
			}
			_, _, axErr = ctRef.ReverseInline("temp")
			if axErr != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("containers.%s: %v", refName, axErr)
			}
		}
	}

	if !preprocessing {
		// Container checking is skipped if we are validating during preprocessing, because
		// in the context of a EmbeddedDeploymentTemplate, the containers field will be null.
		// This happens because containers field is "overwritten" by the child and becomes
		// dropped during marshalling/unmarshalling
		if len(tmpl.Containers) == 0 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("'containers' field missing or empty")
		}
		if len(tmpl.Containers) > 1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Only one container per deployment is currently supported")
		}

		axErr := tmpl.ValidateParameterScope()
		if axErr != nil {
			return axErr
		}
	}

	return nil
}

// ValidateParameterScope checks that all used parameters within the scope of the template, are declared and of the same type
func (tmpl *DeploymentTemplate) ValidateParameterScope() *axerror.AXError {
	declaredParams := tmpl.parametersInScope()
	usedParams, axErr := tmpl.usedParameters()
	if axErr != nil {
		return axErr
	}
	axErr = validateParams(declaredParams, usedParams)
	if axErr != nil {
		return axErr
	}
	return nil
}

func (tmpl *DeploymentTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	// This loop iterates the containers and checks if it is a template reference (as opposed to inlined container).
	// If it is, ensures we satisfy all parameters to the template, and ensure they are of the same type
	for refName, ctRef := range tmpl.Containers {
		if ctRef.Inlined() {
			continue
		}
		// Case where we are referencing another template
		st, ok := context.Templates[ctRef.Template]
		if !ok {
			return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("containers.%s.template: '%s' does not exist", refName, ctRef.Template)
		}
		if st.GetType() != TemplateTypeContainer {
			return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("containers.%s.template: '%s' must be of type container", refName, ctRef.Template)
		}

		ct := st.(*ContainerTemplate)
		scopedParams := tmpl.parametersInScope()
		axErr := validateReceiverParams(ct.Name, ct.Inputs, ctRef.Arguments, scopedParams)
		if axErr != nil {
			return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("containers.%s: %v", refName, axErr)
		}
	}
	return nil
}

func (tmpl *DeploymentTemplate) parametersInScope() paramMap {
	return getParameterDeclarations(tmpl.Inputs, tmpl.Volumes, tmpl.Fixtures)
}

// usedParameters detects all the parameters used in various parts of a deployment template and returns it in a paramMap.
// For deployments, check inline container's command, args, env, image
func (tmpl *DeploymentTemplate) usedParameters() (paramMap, *axerror.AXError) {
	pMap := make(paramMap)
	axErr := pMap.extractUsedParams(tmpl.ApplicationName, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.DeploymentName, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.ExternalRoutes, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.InternalRoutes, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Fixtures, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Volumes, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.TerminationPolicy, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	// merge any used parameters in inlined containers
	for refName, ct := range tmpl.Containers {
		ctParams, axErr := ct.usedParameters()
		if axErr != nil {
			return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("containers.%s: %v", refName, axErr)
		}
		axErr = pMap.merge(ctParams)
		if axErr != nil {
			return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("containers.%s: %v", refName, axErr)
		}
	}
	return pMap, nil
}

func (irs InternalRoutes) Validate() *axerror.AXError {
	set := map[string]bool{}
	for i, route := range irs {
		if axErr := route.Validate(); axErr != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("internal_routes[%d]: %v", i, axErr)
		}
		if _, ok := set[route.Name]; ok {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("internal_routes[%d]: duplicated internal route name: %s", i, route.Name)
		}
		set[route.Name] = true
	}
	return nil
}

func (ir *InternalRoute) Validate() *axerror.AXError {
	if ir.Name == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("'name' is required")
	}

	if !common.ValidateKubeObjName(ir.Name) {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'name' %v is not valid", ir.Name)
	}

	if len(ir.Ports) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("no 'ports' provided")
	}

	for _, port := range ir.Ports {
		if _, err := strconv.ParseInt(port.Port, 0, 64); err != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'port' %v is not valid: %v", port.Port, err.Error())
		}

		if _, err := strconv.ParseInt(port.TargetPort, 0, 64); err != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'target_port' %v is not valid: %v", port.TargetPort, err.Error())
		}
	}

	return nil
}

func (ers ExternalRoutes) Validate() *axerror.AXError {
	dupPorts := make(map[string]bool)
	for i, route := range ers {
		if axErr := route.Validate(); axErr != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("external_routes[%d]: %v", i, axErr)
		}
		if _, ok := dupPorts[route.TargetPort]; ok {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("external_routes[%d]: duplicated port: %s", i, route.TargetPort)
		}
		if len(route.DNSName) > 2000 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("external_routes[%d]: dns_name '%s' can not be longer than 2000 characters")
		}
		dupPorts[route.TargetPort] = true
	}
	return nil
}

func (er *ExternalRoute) Validate() *axerror.AXError {
	if _, err := strconv.ParseInt(er.TargetPort, 0, 64); err != nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'port' %v is not valid: %v", er.TargetPort, err.Error())
	}

	if er.IPWhiteList == nil || len(er.IPWhiteList) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'ip_white_list' is required")
	}

	list := []string{}
	for _, ip := range er.IPWhiteList {
		list = append(list, strings.TrimSpace(ip))
	}

	er.IPWhiteList = list
	for _, ip := range er.IPWhiteList {
		if !common.ValidateCIDR(ip) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'ip_white_list' (%v) doesn't comply with IPv4 CIDR format", ip)
		}
	}

	if len(er.Visibility) == 0 {
		er.Visibility = ExternalRouteVisibilityWorld
	} else {
		if er.Visibility != ExternalRouteVisibilityWorld && er.Visibility != ExternalRouteVisibilityOrg {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("invalid external route visibility: '%s'. Valid values are: world, organization", er.Visibility)
		}
	}
	return nil
}

func (s *Strategy) Validate() *axerror.AXError {
	// validation for update strategy
	if s == nil {
		return nil
	}
	if s.Type != StrategyRollingUpdate && s.Type != StrategyRecreate {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy.type can only be one of [recreate, rolling_update]")
	}
	if s.Type == StrategyRecreate && s.RollingUpdate != nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy type \"recreate\" cannot include configuration for rolling_update")
	}

	if s.RollingUpdate != nil {
		maxUnavailable := 1
		maxSurge := 1
		var err error
		if len(s.RollingUpdate.MaxUnavailable) > 0 {
			if maxUnavailable, err = strconv.Atoi(s.RollingUpdate.MaxUnavailable); err != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy.rolling_update.max_unavailable should be a number")
			}
		}
		if len(s.RollingUpdate.MaxSurge) > 0 {
			if maxSurge, err = strconv.Atoi(s.RollingUpdate.MaxSurge); err != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy.rolling_update.max_surge should be a number")
			}
		}
		if maxUnavailable == 0 && maxSurge == 0 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("strategy.rolling_update.max_unavailable & strategy.rolling_update.max_surge cannot be 0 at the same time")
		}
	}

	return nil
}
