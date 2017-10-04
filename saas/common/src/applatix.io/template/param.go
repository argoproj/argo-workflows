package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"applatix.io/axerror"
)

// types of parameters used in service templates
const (
	paramTypeString   = "string"
	paramTypeFixture  = "fixture"
	paramTypeArtifact = "artifact"
	paramTypeVolume   = "volume"
)

var (
	paramNameRegexStr       = "[-0-9A-Za-z_]+"
	paramNameRegex          = regexp.MustCompile("^[-0-9A-Za-z_]+$")
	ouputArtifactRegexp     = regexp.MustCompile(`^%%(steps|fixtures)\.` + paramNameRegexStr + `\.outputs\.artifacts\.` + paramNameRegexStr + `%%$`)
	VarRegex                = regexp.MustCompile("%%[-0-9A-Za-z_]+(\\.[-0-9A-Za-z_]+)*%%")
	varRegexExact           = regexp.MustCompile("^%%[-0-9A-Za-z_]+(\\.[-0-9A-Za-z_]+)*%%$")
	listExpansionParamRegex = regexp.MustCompile("\\$\\$\\[(.*)\\]\\$\\$")
	ConfigVarRegex          = regexp.MustCompile("%%config\\.([^ %,:]+)\\.([-0-9A-Za-z]+)\\.([-0-9A-Za-z]+)%%")
	ArtifactTagRegex        = regexp.MustCompile("%%artifacts\\.tag\\.[-0-9A-Za-z_]+\\.[-0-9A-Za-z_]+%%")
	ServiceOutputRegex      = regexp.MustCompile("%%service\\.[-0-9A-Za-z_]+\\.outputs\\.[-0-9A-Za-z_]+\\.[-0-9A-Za-z_]+%%")
)

// param is the type used to track parameter usages
type param struct {
	name       string
	paramType  string
	defaultVal *string
}

func (p *param) parts() []string {
	return strings.Split(strings.Replace(p.name, "%%", "", -1), ".")
}

type paramMap map[string]param

// IsParam returns whether or not the supplied string is a paramter, of the form %%param%%
func IsParam(s string) bool {
	return varRegexExact.MatchString(s)
}

// HasParam returns whether or not the supplied string contains a parameter, of the form %%param%%
func HasParam(s string) bool {
	return VarRegex.MatchString(s)
}

// HasGlobalScope returns if the parameter is accessible anywhere because it belongs in the global scope
func HasGlobalScope(s string) bool {
	if !strings.HasPrefix(s, "%%") {
		s = "%%" + s + "%%"
	}
	if ConfigVarRegex.MatchString(s) {
		return true
	}
	if ArtifactTagRegex.MatchString(s) {
		return true
	}
	if ServiceOutputRegex.MatchString(s) {
		return true
	}
	return false
}

// validateParams is a helper to iterates all %%parameter%% used in a template,
// and verify they are properly declared and of the same type as what was declared.
func validateParams(declaredParams, usedParams paramMap) *axerror.AXError {
	for _, p := range usedParams {
		if HasGlobalScope(p.name) {
			continue
		}
		declaredParam := declaredParams.resolveVariable(p.name)
		if declaredParam == nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("cannot resolve '%%%%%s%%%%'", p.name)
		}
		if declaredParam.paramType != p.paramType {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("parameter '%s' used as a %s but declared as a %s", p.name, p.paramType, declaredParam.paramType)
		}
	}
	return nil
}

// resolveVariable checks if the variable is resolvable in the parameter map, and returns the param
// Special handling is performed for acesssing fixture attributes (and eventually volumes), in which
// we will accept any attribute as resolvable, so long as the fixture exists in the param map
func (pm paramMap) resolveVariable(s string) *param {
	s = strings.Trim(s, "%")
	parts := strings.Split(s, ".")
	if strings.HasPrefix(s, "inputs.fixtures.") {
		if len(parts) >= 4 {
			// accessing attribute
			fixName := strings.Join(parts[0:3], ".")
			if _, ok := pm[fixName]; ok {
				return &param{
					name:      s,
					paramType: paramTypeString,
				}
			}
		}
	} else if strings.HasPrefix(s, "fixtures.") {
		if len(parts) >= 3 {
			// accessing attribute
			fixName := strings.Join(parts[0:2], ".")
			if _, ok := pm[fixName]; ok {
				return &param{
					name:      s,
					paramType: paramTypeString,
				}

			}
		}
	}
	if p, ok := pm[s]; ok {
		return &p
	}
	return nil
}

// extractUsedParams extracts any used parameters from some field of a service template (e.g. container.image) into the current paramMap
// it works by marshalling the structure it to a string, and performing a regexp search
func (pm paramMap) extractUsedParams(val interface{}, paramType string) *axerror.AXError {
	if val == nil {
		return nil
	}
	str, err := json.Marshal(val)
	if err != nil {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Could not marshal %v", val)
	}
	for _, match := range VarRegex.FindAll(str, -1) {
		pName := strings.Trim(string(match), "%")
		p := param{
			name:      pName,
			paramType: paramType,
		}

		if prev, exists := pm[p.name]; exists {
			if prev.paramType != p.paramType {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("parameter '%s' used as both a %s and %s", pName, prev.paramType, p.paramType)
			}
		} else {
			pm[p.name] = p
		}
	}
	return nil
}

func (pm paramMap) merge(other paramMap) *axerror.AXError {
	for pName, p := range other {
		if prev, exists := pm[pName]; exists {
			if prev.paramType != p.paramType {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("parameter '%s' used as both a %s and %s", pName, prev.paramType, p.paramType)
			}
		} else {
			pm[pName] = p
		}
	}
	return nil
}

func (pm paramMap) getParam(name string) *param {
	p, ok := pm["%%"+name+"%%"]
	if !ok {
		return nil
	}
	return &p
}

// validateReceiverParams validates that all of the template receiver's inputs can be satisfied
func validateReceiverParams(rcvrTemplateName string, rcvrInputs *Inputs, arguments Arguments, scopedParams paramMap) *axerror.AXError {
	unresolved, axErr := validateReceiverParamsPartial(rcvrTemplateName, rcvrInputs, arguments, scopedParams)
	if axErr != nil {
		return axErr
	}
	for rcvrInputName := range unresolved {
		return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("%s.%s parameter was not satisfied by caller", rcvrTemplateName, rcvrInputName)
	}
	return nil
}

// validateReceiverParamsPartial iterates each of the template receiver's inputs to see if it can satisfied by one of the following:
// 1. the parameters the sender is explicitly sending
// 2. a default value in the reciever's inputs
// 3. parameters from the sender's scope
// This function also verifies the sender's and receiver's parameters are of the same parameter type.
// It returns paramMap of unresolved parameters, or any errors it encountered
func validateReceiverParamsPartial(rcvrTemplateName string, rcvrInputs *Inputs, arguments Arguments, scopedParams paramMap) (paramMap, *axerror.AXError) {
	var rcvrInputsParamMap paramMap
	if rcvrInputs != nil {
		rcvrInputsParamMap = rcvrInputs.parameters(false)
	}

	// First check if caller is passing any extra parameters that the template does not accept
	for arg := range arguments {
		if rcvrInputs == nil {
			return nil, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("template '%s' does not accept any inputs but arguments were supplied", rcvrTemplateName)
		}
		if _, ok := rcvrInputsParamMap[fmt.Sprintf("inputs.%s", arg)]; !ok {
			return nil, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("template '%s' does not accept an input '%s'", rcvrTemplateName, arg)
		}
	}

	unresolved := make(paramMap)
	if rcvrInputs == nil {
		// Reciever has no params. Nothing to do
		return unresolved, nil
	}
	for _, rcvrInput := range rcvrInputsParamMap {
		// This loop ensure that all inputs to the receiver are satisfied by our rules:

		// Check #1: are we explicitly sending in a value to the template?
		argName := strings.TrimPrefix(rcvrInput.name, "inputs.")
		argVal, ok := arguments[argName]
		if ok && argVal != nil {
			// yes we are. is the passed value a "%%variable%%" ?
			if IsParam(*argVal) {
				if HasGlobalScope(*argVal) {
					// user is passing in something like "%%secrets.secretname%%". We consider this resolved
					continue
				}
				// yes it is. ensure that that parameter is resolvable in caller's scope
				p := scopedParams.resolveVariable(*argVal)
				if p == nil {
					return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("cannot resolve '%s'", *argVal)
				}
				// ensure the type we are sending matches the receiving type
				if p.paramType != rcvrInput.paramType {
					return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("%s expected parameter of type '%s' but received '%s' instead", rcvrInput.name, rcvrInput.paramType, p.paramType)
				}
			} else {
				// we are passing a string to the template. ensure the template is expecting it in parameters
				if rcvrInput.paramType != paramTypeString {
					return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("%s expected parameter of type '%s' but received '%s' instead", rcvrInput.name, rcvrInput.paramType, paramTypeString)
				}
			}
			continue
		}

		// Check #2: does the receiver has a default value
		if rcvrInput.defaultVal != nil {
			// also, is the default value usable? we consider %%session.commit%% and %%session.repo%%
			// as unusable defaults because for leaf templates, they would go unsubstituted to the container.
			if *rcvrInput.defaultVal != "%%session.commit%%" && *rcvrInput.defaultVal != "%%session.repo%%" {
				// yes. we are okay
				continue
			}
		}

		// If we get here, the sender did not explicity pass in a parameter value to the template for this input.
		// Additionally, the receiver does not have a default value.
		// e.g. they may have done:
		// - step-build
		//     template: build-template
		// Here, the expectation is that argument would be inferred from this template's own inputs.
		// We now need to see if we can resolve the parameter from the scope

		// Check #3: can we get get the value from the caller's scope? (i.e. inputs)
		p, ok := scopedParams[rcvrInput.name]
		if !ok {
			unresolved[rcvrInput.name] = rcvrInput
		} else {
			// param was satisfied by a parameter in sender's scope (i.e. inputs). ensure the type matches
			// NOTE: now that we switched fully qualified parameter names, this check may not be needed
			if p.paramType != rcvrInput.paramType {
				return nil, axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("%s expected parameter of type '%s' but received '%s' instead", rcvrInput.name, rcvrInput.paramType, p.paramType)
			}
		}
	}
	return unresolved, nil
}

// getParameterDeclarations is a helper function to return a paramMap for all parameters declared in the template scope.
// e.g. either as inputs, or implicitly in volumes, fixtures, sections
func getParameterDeclarations(inputs *Inputs, volumes VolumeRequirements, fixtures FixtureRequirements) paramMap {
	pMap := make(paramMap)
	if inputs != nil {
		pMap.merge(inputs.parameters(true))
	}
	for refName := range volumes {
		p := param{
			name:      fmt.Sprintf("volumes.%s", refName),
			paramType: paramTypeVolume,
		}
		pMap[p.name] = p
	}
	for _, fixReqMap := range fixtures {
		for refName := range fixReqMap {
			p := param{
				name:      fmt.Sprintf("fixtures.%s", refName),
				paramType: paramTypeFixture,
			}
			pMap[p.name] = p
		}
	}
	return pMap
}

// extractScopedParameters builds a paramMap of a datastructure (possibly nested) which is used to build up a scope
// of referenceable variables. (e.g. allow a user to do something like %%resources.mem_mib%%)
// NOTE: this is not yet used
func extractScopedParameters(from interface{}, prefix string, paramType string, skipKeys []string) paramMap {
	jsonBytes, err := json.Marshal(from)
	if err != nil {
		panic(err)
	}
	var tmplMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &tmplMap)
	if err != nil {
		panic(err)
	}
	for _, key := range skipKeys {
		delete(tmplMap, key)
	}
	flatMap := flatten(tmplMap)
	pMap := make(paramMap)
	for pName := range flatMap {
		p := param{
			name:      prefix + pName,
			paramType: paramType,
		}
		pMap[p.name] = p
	}
	return pMap
}

// flatten takes a map and returns a new one where nested maps are replaced by dot-delimited keys.
func flatten(m map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

// GetTemplateArguments returns a argument map for all arguments expected by the template
func GetTemplateArguments(st TemplateIf) Arguments {
	in := st.GetInputs()
	if in == nil {
		return nil
	}
	args := make(map[string]*string)
	for pName := range in.Parameters {
		args[fmt.Sprintf("parameters.%s", pName)] = nil
	}
	for pName := range in.Artifacts {
		args[fmt.Sprintf("artifacts.%s", pName)] = nil
	}
	for pName := range in.Volumes {
		args[fmt.Sprintf("volumes.%s", pName)] = nil
	}
	for pName := range in.Fixtures {
		args[fmt.Sprintf("fixtures.%s", pName)] = nil
	}
	return args
}
