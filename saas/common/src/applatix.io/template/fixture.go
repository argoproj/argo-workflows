package template

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"applatix.io/axerror"
)

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type FixtureTemplate struct {
	BaseTemplate `json:",squash"`
	Attributes   map[string]FixtureAttribute `json:"attributes,omitempty"`
	Actions      map[string]FixtureAction    `json:"actions,omitempty"`
}

type FixtureAttribute struct {
	Name    string        `json:"name,omitempty"`
	Type    string        `json:"type,omitempty"`
	Flags   string        `json:"flags,omitempty"`
	Options []interface{} `json:"options,omitempty"`
	Default interface{}   `json:"default,omitempty"`
}

type FixtureAction struct {
	TemplateRef `json:",squash"`
	OnSuccess   *string `json:"on_success,omitempty"`
	OnFailure   *string `json:"on_failure,omitempty"`
}

// FixtureRequirements is a list of fixture requirements needed by a workflow or deployment
type FixtureRequirements []map[string]FixtureTemplateRef

// FixtureTemplateRef can act as a reference to a container service template (dynamic fixture), a fixture requirement (managed fixture)
type FixtureTemplateRef struct {
	TemplateRef        `json:",squash"`
	FixtureRequirement `json:",squash"`
}

// FixtureRequirement is an individual request for a managed fixture
type FixtureRequirement struct {
	Class      string            `json:"class,omitempty"`
	Name       string            `json:"name,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

const (
	FixtureAttributeTypeString = "string"
	FixtureAttributeTypeInt    = "int"
	FixtureAttributeTypeBool   = "bool"
	FixtureAttributeTypeFloat  = "float"
)

var FixtureAttributeDataTypesMap = map[string]bool{
	FixtureAttributeTypeString: true,
	FixtureAttributeTypeInt:    true,
	FixtureAttributeTypeBool:   true,
	FixtureAttributeTypeFloat:  true,
}

const (
	FixtureAttributeFlagRequired = "required"
	FixtureAttributeFlagArray    = "array"
)

var FixtureAttributeFlagsMap = map[string]bool{
	FixtureAttributeFlagRequired: true,
	FixtureAttributeFlagArray:    true,
}

// FixtureAttributeReservedNamesMap is a set of reserved names that cannot be used as attribute
// names in a fixture template. These are prevented from use, since they would collide with the
// first-level attributes when fixturemanager flattens the attributes into a single level map.
var FixtureAttributeReservedNamesMap = map[string]bool{
	"id":          true,
	"name":        true,
	"description": true,
	"class":       true,
	"class_id":    true,
	"class_name":  true,
	"status":      true,
}

// FixtureAttributeRegexp matches valid attribute names.
// Valid attribute names must be lowercase, alpha-numeric or underscore, beginning with a letter.
// This enables fixturemanager a reliable way of querying attributes in its database (mongodb).
var FixtureAttributeRegexp = regexp.MustCompile("^[a-z][0-9a-z_]*$")

var FixtureActionEventValueMap = map[string]bool{
	"enable":  true,
	"disable": true,
}

func (fr FixtureRequirements) Validate() *axerror.AXError {
	for i, parallelFixtures := range fr {
		for refName, ftr := range parallelFixtures {
			axErr := ftr.Validate()
			if axErr != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("fixtures[%d].%s: %s", i, refName, axErr)
			}
		}
	}
	return nil
}

func (tmpl *FixtureTemplate) GetInputs() *Inputs {
	return nil
}

func (tmpl *FixtureTemplate) GetOutputs() *Outputs {
	return nil
}

func (tmpl *FixtureTemplate) Validate(preproc ...bool) *axerror.AXError {
	if axErr := tmpl.BaseTemplate.Validate(); axErr != nil {
		return axErr
	}
	if len(tmpl.Name) < 1 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("fixture name is empty")
	}

	for key, attrib := range tmpl.Attributes {
		isArray := false
		// attribute name cannot be reserved words
		if _, reserved := FixtureAttributeReservedNamesMap[key]; reserved {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute name %s is a reserved key word", key))
		}
		// attribute names must be lowercase, alpha-numeric or underscore, beginning with a letter
		if !FixtureAttributeRegexp.MatchString(key) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("invalid fixture attribute name %s: must be lowercase, alpha-numeric or underscore, beginning with a letter", key))
		}
		// validate data type
		if _, valid := FixtureAttributeDataTypesMap[attrib.Type]; !valid {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid type. type should be one of (string,int,bool,float).", key))
		}
		// validate flags
		if len(attrib.Flags) > 0 {
			flagsList := strings.Split(attrib.Flags, ",")
			for _, flag := range flagsList {
				if _, valid := FixtureAttributeFlagsMap[flag]; !valid {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid flags. valid flags are (required,array).", key))
				}
				if flag == FixtureAttributeFlagArray {
					isArray = true
				}
			}
		}
		// validate default
		if attrib.Default != nil {
			switch v := attrib.Default.(type) {
			default:
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type: %T.", key, v))
			case bool:
				if isArray || attrib.Type != FixtureAttributeTypeBool {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type of bool.", key))
				}
			case int:
				if isArray || attrib.Type != FixtureAttributeTypeInt {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type of int.", key))
				}
			case float64:
				if isArray || (attrib.Type != FixtureAttributeTypeFloat && attrib.Type != FixtureAttributeTypeInt) {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type of float.", key))
				}
				// json.Unmarshal will unmarshal ints as a float64. This checks if default value is really an int
				if attrib.Type == FixtureAttributeTypeInt {
					if attrib.Default.(float64) != float64(int64(attrib.Default.(float64))) {
						return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s is a float instead of int.", key))
					}
					attrib.Default = int64(attrib.Default.(float64))
				}
			case string:
				if isArray || attrib.Type != FixtureAttributeTypeString {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type of string", key))
				}
			case []interface{}:
				if !isArray {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("fixture attribute %s has invalid default type of array", key))
				}
			}
		}
	}

	if tmpl.Actions != nil {
		for actionName, action := range tmpl.Actions {
			if len(action.Template) == 0 {
				//action should have a template
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage(fmt.Sprintf("actions.%s.template is required", actionName))
			}
			// check on_success and on_failure value for action
			if action.OnSuccess != nil {
				if _, validValue := FixtureActionEventValueMap[*action.OnSuccess]; !validValue {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s.on_success invalid value '%s'. expected: enable, disable", actionName, *action.OnSuccess)
				}
			}
			if action.OnFailure != nil {
				if _, validValue := FixtureActionEventValueMap[*action.OnFailure]; !validValue {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s.on_failure invalid value '%s'. expected: enable, disable", actionName, *action.OnSuccess)
				}
			}
		}
	}
	return nil
}

// ValidateContext iterates the actions and performs context validation for the parameters in the template references
func (tmpl *FixtureTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	// for each action make sure template exists
	if tmpl.Actions != nil {
		// Build up a scope of fixture attributes which we will use to validate any %%attributes.name%% usages in the action parameters
		scopedParams := make(paramMap)
		attributeNames := map[string]bool{}
		for k := range FixtureAttributeReservedNamesMap {
			attributeNames[k] = true
			p := param{
				name:      fmt.Sprintf("%s.%s", KeywordAttributes, k),
				paramType: paramTypeString,
			}
			scopedParams[p.name] = p
		}
		for k := range tmpl.Attributes {
			attributeNames[k] = true
			p := param{
				name:      fmt.Sprintf("%s.%s", KeywordAttributes, k),
				paramType: paramTypeString,
			}
			scopedParams[p.name] = p
		}

		for actionName, action := range tmpl.Actions {
			st, exists := context.Templates[action.Template]
			if !exists {
				// template should exist
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s template %s does not exist", actionName, action.Template)
			}
			rcvrInputs := st.GetInputs()
			switch st.GetType() {
			case TemplateTypeWorkflow, TemplateTypeContainer:
				break
			default:
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s template %s must be of type container or workflow", actionName, action.Template)
			}
			var axErr *axerror.AXError
			if actionName == "create" || actionName == "delete" {
				// for create and delete actions, all reciever params need to be resolved because these actions cannot accept inputs
				// since we trigger these actions automatically on fixture create/delete.
				axErr = validateReceiverParams(st.GetName(), rcvrInputs, action.Arguments, scopedParams)
			} else {
				// otherwise, we just need to check the parameter type matches. unresolved attributes will become UI input parameters
				_, axErr = validateReceiverParamsPartial(st.GetName(), rcvrInputs, action.Arguments, scopedParams)
			}
			if axErr != nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s %v", actionName, axErr)
			}
		}
	}
	return nil
}

func (ftr *FixtureTemplateRef) IsDynamicFixture() bool {
	return ftr.Template != ""
}

func (ftr *FixtureTemplateRef) Validate() *axerror.AXError {
	if ftr.IsDynamicFixture() {
		return ftr.TemplateRef.Validate()
	}
	return nil
}

func (ftr FixtureRequirement) Equals(o FixtureRequirement) bool {
	return reflect.DeepEqual(ftr, o)
}

func (ftr FixtureRequirement) String() string {
	parts := []string{}
	if ftr.Name != "" {
		parts = append(parts, fmt.Sprintf("name=%s", ftr.Name))
	}
	if ftr.Class != "" {
		parts = append(parts, fmt.Sprintf("class=%s", ftr.Class))
	}
	for attrName, attrVal := range ftr.Attributes {
		parts = append(parts, fmt.Sprintf("attribute.%s=%s", attrName, attrVal))
	}
	return strings.Join(parts, ", ")
}
