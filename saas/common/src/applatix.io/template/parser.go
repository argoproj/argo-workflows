package template

import (
	"reflect"
	"strconv"
	"strings"

	"applatix.io/axerror"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
)

// weakParseNumberToString allows mapstructure to accept an number as a string
func weakParseNumberToString(from reflect.Kind, to reflect.Kind, data interface{}) (interface{}, error) {
	switch to {
	case reflect.String:
		switch from {
		case reflect.Float32, reflect.Float64:
			return strconv.FormatFloat(data.(float64), 'f', -1, 64), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(data.(int64), 10), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return strconv.FormatUint(data.(uint64), 10), nil
		}
	}
	return data, nil
}

// Unmarshal is a wraper around yaml.Unarshal and mapstructure.Decode to unmarshal the object into the supplied interface.
// It provides useful error messages upon any errors, including the ability to detect and prevent unknown fields.
func Unmarshal(templateBytes []byte, template interface{}, strict bool) *axerror.AXError {
	var axErr *axerror.AXError
	err := yaml.Unmarshal(templateBytes, &template)
	if err != nil {
		axErr = axerror.ERR_API_INVALID_PARAM.NewWithMessage(err.Error())
	}
	if axErr != nil || strict {
		// If we have a yaml unmarshalling error, we want to use mapstructure to unmarshal because it gives better errors about what went wrong.
		// Secondly, if we are running in strict mode, we want to fail if we detect unknown fields supplied by the user, which we can get using
		// mapstructure. Both of these are best effort, so if we get any unexpected errors in this block, return the original error.

		// First unmarshal to a simple map
		var yamlMap map[string]interface{}
		err = yaml.Unmarshal(templateBytes, &yamlMap)
		if err != nil {
			// Best effort. return original error
			return axErr
		}

		// Use mapstructure to unmarshal from yamlMap and enable ErrorUnused if we are running in strict mode
		config := &mapstructure.DecoderConfig{
			DecodeHook:  weakParseNumberToString,
			ErrorUnused: strict,
			Result:      template,
			TagName:     "json",
		}
		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			// Best effort. return original error
			return axErr
		}
		err = decoder.Decode(yamlMap)
		if err != nil {
			// mapstructure provided a more useful error message. Return that error
			// it can also return multiple errors, but just choose one, for display purposes
			msErr := err.(*mapstructure.Error)
			// make it a little more human readable for invalid keys at the root layer
			errMsg := strings.TrimPrefix(msErr.Errors[0], "'' has ")
			axErr = axerror.ERR_API_INVALID_PARAM.NewWithMessage(errMsg)
			// TODO: we can further give a better error message to understand when user is using a %%param%% (string)
			// in place of a numeric/bool field, and give a message "field cannot be parameterized"
		}
	}
	return axErr
}

// UnmarshalTemplate parses a byte array and returns a corresponding ServiceTemplateIf
// Will return the partially parsed service template even if there is an error
func UnmarshalTemplate(data []byte, strict bool) (TemplateIf, *axerror.AXError) {
	// First determine the type of the template
	var tmpl BaseTemplate
	err := yaml.Unmarshal(data, &tmpl)
	if err != nil {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid yaml: %s", string(data))
	}
	if strings.TrimSpace(tmpl.Name) == "" {
		return &tmpl, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Template has no name")
	}

	switch tmpl.Type {
	case TemplateTypeContainer:
		var tmpl ContainerTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	case TemplateTypeWorkflow:
		var tmpl WorkflowTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	case TemplateTypeDeployment:
		var tmpl DeploymentTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	case TemplateTypeFixture:
		var tmpl FixtureTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	case TemplateTypePolicy:
		var tmpl PolicyTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	case TemplateTypeProject:
		var tmpl ProjectTemplate
		axErr := Unmarshal(data, &tmpl, strict)
		return &tmpl, axErr
	default:
		return &tmpl, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Unknown service template type: %s", tmpl.Type)
	}
}
