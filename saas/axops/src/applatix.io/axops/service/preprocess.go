package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"applatix.io/axerror"
	"applatix.io/axops/configuration"
	"applatix.io/axops/utils"
	"applatix.io/axops/volume"
	"applatix.io/template"
)

func (s *Service) Preprocess() (*Service, *axerror.AXError) {
	// First, make a copy of the service and arguments to return
	substitutedSvc := s.Copy()
	substitutedArguments := s.Arguments.Copy()

	// Next substitute all the global variables, i.e. configurations, cross-referenced artifacts
	newTemplate, axErr := SubstituteGlobalVariables(substitutedSvc.Template, *substitutedArguments)
	if axErr != nil {
		return nil, axErr
	}
	substitutedSvc.Template = newTemplate

	// Then substitute arguments to the template
	substitutedTmpl, axErr := substitutedSvc.Template.SubstituteArguments(*substitutedArguments)
	if axErr != nil {
		return nil, axErr
	}
	substitutedSvc.Template = substitutedTmpl

	// Set any default values which were not specified in the template
	setDefaults := func(s *Service, p *Service, name string) *axerror.AXError {
		if s.Template != nil && s.Template.GetType() == template.TemplateTypeContainer {
			ct := s.Template.(*EmbeddedContainerTemplate)
			if ct.Resources == nil {
				utils.DebugLog.Printf("Defaulting CPU and memory")
				ct.Resources = &template.ContainerResources{
					MemMiB:   "512",
					CPUCores: "0.25",
				}
			}
		}
		return nil
	}
	axErr = substitutedSvc.Iterate(setDefaults, nil, "")
	if axErr != nil {
		return nil, axErr
	}

	axErr = substitutedSvc.Iterate(revalidate, nil, "")
	if axErr != nil {
		return nil, axErr
	}
	utils.DebugLog.Printf("Preprocessing successful")
	return substitutedSvc, nil

}

// revalidate is a service template iterator function called to revalidate the template after substitution.
// It additionally checkes the template to ensure any volumes and fixtures requested in the template exist
// and are available for use.
func revalidate(s *Service, p *Service, name string) *axerror.AXError {
	if s.Template == nil {
		return nil
	}
	var axErr *axerror.AXError
	fixReqs := make([]template.FixtureRequirement, 0)

	switch s.Template.GetType() {
	case template.TemplateTypeContainer:
		ct := s.Template.(*EmbeddedContainerTemplate)
		axErr = ct.Validate(true)
	case template.TemplateTypeWorkflow:
		wt := s.Template.(*EmbeddedWorkflowTemplate)
		axErr = wt.WorkflowTemplate.Validate(true)
		if axErr == nil {
			axErr = verifyVolumeExistence(wt.Volumes, false)
			for _, parallelFixtures := range wt.Fixtures {
				for _, fixReq := range parallelFixtures {
					if !fixReq.IsDynamicFixture() {
						fixReqs = append(fixReqs, *fixReq.FixtureRequirement)
					}
				}
			}
		}
	case template.TemplateTypeDeployment:
		dt := s.Template.(*EmbeddedDeploymentTemplate)
		axErr = dt.DeploymentTemplate.Validate(true)
		if axErr == nil {
			axErr = verifyVolumeExistence(dt.Volumes, true)
			for _, parallelFixtures := range dt.Fixtures {
				for _, fixReq := range parallelFixtures {
					fixReqs = append(fixReqs, fixReq.FixtureRequirement)
				}
			}
		}
	}
	if axErr == nil && len(fixReqs) > 0 {
		axErr = verifyFixtureExistence(fixReqs)
	}
	if axErr != nil {
		axErr = axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Validation of %s failed: %v", s.Name, axErr)
		utils.DebugLog.Println(axErr)
		return axErr
	}
	return nil
}

// verifyVolumeExistence verifies a volume exists matching the name
// It optionally verifies that the volume is 'active'. This check is
// disabled for workflows, because it is acceptable for jobs to request
// volumes asynchronously (as opposed to deployments)
func verifyVolumeExistence(volReqs template.VolumeRequirements, verifyActive bool) *axerror.AXError {
	for _, volReq := range volReqs {
		if volReq.Name != "" {
			vol, axErr := volume.GetVolumeByAXRN(fmt.Sprintf("vol:/%s", volReq.Name))
			if axErr != nil {
				return axErr
			}
			if vol == nil {
				return axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Volume with name '%s' does not exist", volReq.Name)
			}
			if verifyActive && vol.Status != volume.VolumeStatusActive {
				return axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Volume '%s' is not '%s'. Current status: %s", volReq.Name, volume.VolumeStatusActive, vol.Status)
			}
		}
	}
	return nil
}

// verifyFixtureExistence verifies a fixture exists matching the requirement
func verifyFixtureExistence(fixReqs []template.FixtureRequirement) *axerror.AXError {
	utils.DebugLog.Printf("Verifying fixture requests %v", fixReqs)
	for _, fixReq := range fixReqs {
		instances, axErr := matchRequirements(fixReq)
		if axErr != nil {
			return axErr
		}
		totalInstances := len(instances)
		utils.DebugLog.Printf("Found %d fixture instances matching %s", totalInstances, fixReq)
		if totalInstances == 0 {
			return axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("No fixture instances exist matching: %s", fixReq)
		}
		// verifyAvailable check is disabled for now because when we upgrade a deployment,
		// it is possible for 0 fixtures matching the requirement to be available. A proper
		// check would take into consideration the deployment_id of the submitted service,
		// and if one of the fixtures is being reserved using the same deployment_id, then
		// we can proceed.
		//if verifyAvailable {
		//	instances, _ := matchRequirements(fixReq, true)
		//	if len(instances) == 0 {
		//		return axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("0 of %d fixture instances available matching: %s", totalInstances, fixReq)
		//	}
		//}
	}
	return nil
}

// matchRequirements returns a list of fixture instances matching a fixture requirement
func matchRequirements(req template.FixtureRequirement) ([]map[string]interface{}, *axerror.AXError) {
	params := make(map[string]interface{})
	params["deleted"] = "false"
	if req.Name != "" {
		params["name"] = req.Name
	}
	if req.Class != "" {
		params["class_name"] = req.Class
	}
	for attrName, attrVal := range req.Attributes {
		params[fmt.Sprintf("attributes.%s", attrName)] = attrVal
	}
	//if verifyAvailable {
	//	params["available"] = "true"
	//}
	type instanceData struct {
		Data []map[string]interface{} `json:"data"`
	}
	var instances instanceData
	axErr := fixMgrCl.Get("v1/fixture/instances", params, &instances)
	if axErr != nil {
		return nil, axErr
	}
	return instances.Data, nil
}

// buildReplaceMap looks at a template's inputs, the supplied arguments to the template, and formulates a map to perform string replacement
func buildReplaceMap(tmpl EmbeddedTemplateIf, arguments template.Arguments) (map[string]*string, *axerror.AXError) {
	// First build a map of all the expected inputs needed by a template. If it has a default value, supply the default.
	// If there is not a default, set the value to nil. This will get overridden, if the caller explicitly supplies the
	// argument. We later look for any nil values in the replaceMap to detect if there were any unsatisfied arguments
	// to the template.
	replaceMap := make(map[string]*string)
	in := tmpl.GetInputs()
	if in != nil {
		for refName, sp := range in.Parameters {
			if sp != nil {
				replaceMap[fmt.Sprintf("%%%%inputs.parameters.%s%%%%", refName)] = sp.Default
			} else {
				replaceMap[fmt.Sprintf("%%%%inputs.parameters.%s%%%%", refName)] = nil
			}
		}
		for refName, art := range in.Artifacts {
			// Check if .from is already set. This will happen in the case of global artifact references
			// In this case we set the replacement map to the from value
			if art != nil && template.HasGlobalScope(art.From) {
				fromVal := art.From
				replaceMap[fmt.Sprintf("%%%%inputs.artifacts.%s%%%%", refName)] = &fromVal
			} else {
				replaceMap[fmt.Sprintf("%%%%inputs.artifacts.%s%%%%", refName)] = nil
			}

		}
		for refName := range in.Volumes {
			replaceMap[fmt.Sprintf("%%%%inputs.volumes.%s%%%%", refName)] = nil
		}
		for refName := range in.Fixtures {
			replaceMap[fmt.Sprintf("%%%%inputs.fixtures.%s%%%%", refName)] = nil
		}
	}

	// For any immediate children of this template, replace usages of %%steps.STEPNAME.outputs.artifacts.ARTNAME%% or
	// %%fixtures.FIXNAME.outputs.artifacts.ARTNAME%% with concrete services IDs. This is only needed with workflows,
	// since containers don't have children, and deployments do not use outputs from child containers
	if tmpl.GetType() == template.TemplateTypeWorkflow {
		wft := tmpl.(*EmbeddedWorkflowTemplate)
		for _, parallelFixtures := range wft.Fixtures {
			for fixRefName, ftr := range parallelFixtures {
				if !ftr.IsDynamicFixture() {
					continue
				}
				outputs := ftr.Template.GetOutputs()
				if outputs == nil {
					continue
				}
				// The following converts %%fixtures.STEP1.outputs.artifacts.ARTNAME%% to %%service.service_id.outputs.artifacts.ARTNAME%%
				for artName := range outputs.Artifacts {
					serviceOutputRef := fmt.Sprintf("%%%%service.%s.outputs.artifacts.%s%%%%", ftr.Id, artName)
					replaceMap[fmt.Sprintf("%%%%fixtures.%s.outputs.artifacts.%s%%%%", fixRefName, artName)] = &serviceOutputRef
				}
			}
		}
		for _, parallelSteps := range wft.Steps {
			for stepName, step := range parallelSteps {
				if step.Template == nil {
					continue
				}
				outputs := step.Template.GetOutputs()
				if outputs == nil {
					continue
				}
				// The following converts %%steps.STEP1.outputs.artifacts.ARTNAME%% to %%service.service_id.outputs.artifacts.ARTNAME%%
				for artName := range outputs.Artifacts {
					serviceOutputRef := fmt.Sprintf("%%%%service.%s.outputs.artifacts.%s%%%%", step.Id, artName)
					replaceMap[fmt.Sprintf("%%%%steps.%s.outputs.artifacts.%s%%%%", stepName, artName)] = &serviceOutputRef
				}
				// If we support other types of outputs, add replacement here
			}
		}
	}

	// Now update the replacement map with the argument value that was explicitly supplied
	for argName, argVal := range arguments {
		if strings.HasPrefix(argName, "session.") {
			// UI will pass session.commit and session.repo as arguments, which we ignore
			continue
		}
		varName := fmt.Sprintf("%%%%inputs.%s%%%%", argName)
		if _, ok := replaceMap[varName]; !ok {
			// If we get here, caller supplied a argument that is not accepted by the child. Construct a useful error message.
			errMsg := fmt.Sprintf("%s is not an input parameter to ", argName)
			if tmpl.GetName() != "" {
				errMsg += fmt.Sprintf("'%s'", tmpl.GetName())
			} else {
				errMsg += fmt.Sprintf("child template")
			}
			varNameParts := strings.Split(varName, ".")
			if len(varNameParts) == 2 {
				errMsg += ". Do you mean 'parameters." + argName + "'?"
			}
			return nil, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessage(errMsg)
		}
		replaceMap[varName] = argVal
	}

	// Check to see if there were any unsatisfied values. If any value in the replacement map is nil, it indicates the
	// caller did not supply a required input to the template.
	for inputName, val := range replaceMap {
		if val == nil {
			return nil, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("%s '%s' was not satisfied by caller", tmpl.GetName(), strings.Trim(inputName, "%"))
		}
		unresolved := template.VarRegex.FindAllString(*val, -1)
		for _, varName := range unresolved {
			if template.HasGlobalScope(varName) {
				utils.DebugLog.Printf("Found unresolved global variable: %s", varName)
			} else {
				utils.DebugLog.Printf("Found unresolved variable: %s", varName)
				// If we are still seeing a %%param%%, it was unresolved by arguments, and it is not a global variable
				// It could be because we are referencing fixtures, or other inputs (when this is eventually supported)
				// TODO: distinguish from this and raise an error indicating there was unresolved variables.
				// NOTE: This may not be necessary since yaml validate should have caught this.
				//return nil, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("%s was not satisfied by caller", strings.Trim(inputName, "%"))
			}
		}
	}
	return replaceMap, nil
}

// SubstituteArguments will recursively substitute the arguments into a new copy of the service template.
// This is achieved in the following steps:
// 1. First make a copy the template with every occurence of the arg value substituted. This will
//    also replace occurrences in our children, which we don't want, but we will correct this in the next step.
// 2. For each of our children, restore the original templates since we only want to substitute arguments
//    in the immediate scope, and not in the scope of our children.
// 3. Finally iterate the children again, and make a recursive call to substitute arguments within child's scope

func (tmpl *EmbeddedContainerTemplate) SubstituteArguments(arguments template.Arguments) (EmbeddedTemplateIf, *axerror.AXError) {
	replaceMap, axErr := buildReplaceMap(tmpl, arguments)
	if axErr != nil {
		return nil, axErr
	}
	var sub EmbeddedContainerTemplate
	axErr = replaceStringMap(tmpl, &sub, replaceMap)
	if axErr != nil {
		return nil, axErr
	}
	axErr = setInputArtifactSource(sub.Inputs, arguments)
	if axErr != nil {
		return nil, axErr
	}
	return &sub, nil
}

func (tmpl *EmbeddedDeploymentTemplate) SubstituteArguments(arguments template.Arguments) (EmbeddedTemplateIf, *axerror.AXError) {
	replaceMap, axErr := buildReplaceMap(tmpl, arguments)
	if axErr != nil {
		return nil, axErr
	}
	// Step 1 - replace all
	var sub EmbeddedDeploymentTemplate
	axErr = replaceStringMap(tmpl, &sub, replaceMap)
	if axErr != nil {
		return nil, axErr
	}
	axErr = setInputArtifactSource(sub.Inputs, arguments)
	if axErr != nil {
		return nil, axErr
	}

	// Step 2 - recurse children
	for name, ctrRef := range tmpl.Containers {
		// NOTE: when recursing our children, we use the original template, but pass
		// substituted arguments since the user may have had variable usages in arguments
		childTmpl, axErr := ctrRef.Template.SubstituteArguments(sub.Containers[name].Arguments)
		if axErr != nil {
			return nil, axErr
		}
		subRef := sub.Containers[name]
		subRef.Template = childTmpl.(*EmbeddedContainerTemplate)
	}
	return &sub, nil
}

func (tmpl *EmbeddedWorkflowTemplate) SubstituteArguments(arguments template.Arguments) (EmbeddedTemplateIf, *axerror.AXError) {
	replaceMap, axErr := buildReplaceMap(tmpl, arguments)
	if axErr != nil {
		return nil, axErr
	}
	// Step 1 - replace all
	var sub EmbeddedWorkflowTemplate
	axErr = replaceStringMap(tmpl, &sub, replaceMap)
	if axErr != nil {
		return nil, axErr
	}
	axErr = setInputArtifactSource(sub.Inputs, arguments)
	if axErr != nil {
		return nil, axErr
	}

	// Step 2 - restore
	for i, parallelSteps := range tmpl.Steps {
		for name, wfs := range parallelSteps {
			// NOTE: when recursing our children, we use the original template, but pass
			// substituted arguments since the user may have had variable usages in arguments
			childTmpl, axErr := wfs.Template.SubstituteArguments(sub.Steps[i][name].Arguments)
			if axErr != nil {
				return nil, axErr
			}
			subRef := sub.Steps[i][name]
			subRef.Template = childTmpl
		}
	}
	for i, parallelFixtures := range tmpl.Fixtures {
		for name, fix := range parallelFixtures {
			if !fix.IsDynamicFixture() {
				// If a managed fixture, substitution already performed
				continue
			}
			// NOTE: we use the original template, but pass substituted arguments
			// since the user may have had variable usages in arguments
			childTmpl, axErr := fix.Template.SubstituteArguments(sub.Fixtures[i][name].Arguments)
			if axErr != nil {
				return nil, axErr
			}
			subRef := sub.Fixtures[i][name]
			subRef.Template = childTmpl.(*EmbeddedContainerTemplate)
		}
	}
	return &sub, nil
}

// SubstituteGlobalVariables replaces all the global variables in the service template.
// This includes %%config.*%%, %%artifact.tag.*%%, and %%session.*%% variables.
func SubstituteGlobalVariables(tmpl EmbeddedTemplateIf, arguments template.Arguments) (EmbeddedTemplateIf, *axerror.AXError) {
	replaceMap := make(map[string]*string)

	// Find all configuration variables in templates
	bytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Global variable substitution failed (marshaling template): %v", err)
	}
	byteStr := string(bytes)
	configVars := template.ConfigVarRegex.FindAllString(byteStr, -1)
	// Find all artifact tag references (e.g. %%artifacts.tag.)
	artTagVars := template.ArtifactTagRegex.FindAllString(byteStr, -1)

	// Find all configuration variables in arguments
	bytes, err = json.Marshal(arguments)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Global variable substitution failed (marshaling arguments): %v", err)
	}
	byteStr = string(bytes)
	configVarsArgs := template.ConfigVarRegex.FindAllString(byteStr, -1)
	configVars = append(configVars, configVarsArgs...)
	artTagVarsArgs := template.ArtifactTagRegex.FindAllString(byteStr, -1)
	artTagVars = append(artTagVars, artTagVarsArgs...)

	// Create a replace map with all the configuration variables
	for _, varName := range configVars {
		if _, ok := replaceMap[varName]; !ok {
			configVal, axErr := configuration.ProcessConfigurationStr(varName)
			if axErr != nil {
				return nil, axErr
			}
			if configVal == nil {
				// nil value indicates config is a secret
				// skip, since secret substitution is handled at platform
				continue
			}
			replaceMap[varName] = configVal
		}
	}

	// Add artifact tags to replacement map
	for _, artVar := range artTagVars {
		artVarParts := strings.Split(artVar, ".")
		tagName := artVarParts[2]
		if _, ok := replaceMap[tagName]; !ok {
			svc, axErr := GetServiceByArtifactTag(tagName)
			if axErr != nil {
				return nil, axErr
			}
			if svc == nil {
				return nil, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("Artifact tag '%s' not found", tagName)
			}
			artName := artVarParts[len(artVarParts)-1]
			svcArtID := fmt.Sprintf("%%%%service.%s.outputs.artifacts.%s%%%%", svc.Id, artName)
			replaceMap[artVar] = &svcArtID
		}
	}

	// Adding all the global session variables into the replace map
	for argName, argVal := range arguments {
		if strings.HasPrefix(argName, "session.") {
			varName := fmt.Sprintf("%%%%%s%%%%", argName)
			replaceMap[varName] = argVal
		}
	}

	// First replace the global variables in the arguments value
	for argName, argVal := range arguments {
		if configVal, ok := replaceMap[*argVal]; ok {
			arguments[argName] = configVal
		}
	}

	// Find right type of object to perform substitution
	switch tmpl.GetType() {
	case template.TemplateTypeContainer:
		var sub EmbeddedContainerTemplate
		axErr := replaceStringMap(tmpl, &sub, replaceMap)
		if axErr != nil {
			return nil, axErr
		}
		return &sub, nil
	case template.TemplateTypeWorkflow:
		var sub EmbeddedWorkflowTemplate
		axErr := replaceStringMap(tmpl, &sub, replaceMap)
		if axErr != nil {
			return nil, axErr
		}
		return &sub, nil
	case template.TemplateTypeDeployment:
		var sub EmbeddedDeploymentTemplate
		axErr := replaceStringMap(tmpl, &sub, replaceMap)
		if axErr != nil {
			return nil, axErr
		}
		return &sub, nil
	}
	return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Cannot find right type of the template", tmpl.GetType())
}

// replaceStringMap will replace all occurrences of the keys in replacements, with the values in replacements,
// and copies the resulting doc into the target interface
// TODO: continue to do this until it cannot find any replacements
func replaceStringMap(from interface{}, to interface{}, replacements map[string]*string) *axerror.AXError {
	bytes, err := json.Marshal(from)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Argument substitution failed (marshaling): %v", err)
	}
	byteStr := string(bytes)
	for fromVal, toVal := range replacements {
		//log.Printf("Replacing %s to %s\n", fromVal, *toVal)
		// The following escapes any special characters (e.g. newlines, tabs, etc...) in preparation for substitution
		escToVal := strconv.Quote(*toVal)
		escToVal = escToVal[1 : len(escToVal)-1]
		byteStr = strings.Replace(byteStr, fromVal, escToVal, -1)
	}
	err = json.Unmarshal([]byte(byteStr), to)
	if err != nil {
		utils.DebugLog.Printf("Argument substitution failed (unmarshaling): %v\nBefore:\n%s\nAfter\n%s", err, string(bytes), byteStr)
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Argument substitution failed (unmarshaling): %v", err)
	}
	return nil
}

// setInputArtifactSource is called during preprocessing of a service object, and sets the 'from'
// field of an input artifact to the one supplied as an argument (typically:
// %%service.<id>.outputs.artifacts.ARTNAME%%) The purpose of this is to provide the lower layers
// (platform/WFE) a convenient location to look up the artifact
func setInputArtifactSource(inputs *template.Inputs, arguments template.Arguments) *axerror.AXError {
	if inputs == nil {
		return nil
	}
	for argName, argVal := range arguments {
		parts := strings.Split(argName, ".")
		if len(parts) < 2 {
			return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Invalid argument synatx '%s'. Expected <input_type>.<input_name> (e.g. 'parameters.%s')", argName, argName)
		}
		switch parts[0] {
		case template.KeywordArtifacts:
			if art, ok := inputs.Artifacts[parts[1]]; ok && argVal != nil {
				if art == nil {
					inputs.Artifacts[parts[1]] = &template.InputArtifact{From: *argVal}
				} else {
					art.From = *argVal
				}
			}
		}
	}
	return nil
}
