package main

import (
	"fmt"
	"strings"
)

func hasOptionalAttributes(attributes *allowedAttributeList) bool {
	for _, attr := range *attributes {
		if attr.Optional {
			return true
		}
	}
	return false
}

func buildParameterList(baseParams []string, c *common, optionType string, attributes *attributesList) []string {
	params := baseParams
	requiredAttrs := getRequiredAttributes(&c.Attributes)

	for _, attr := range requiredAttrs {
		attribDef := getAttribByName(attr.Name, attributes)
		paramName := toLowerCamelCase(attr.Name)
		paramType := attribDef.attrType()
		params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
	}

	if hasOptionalAttributes(&c.Attributes) {
		optionTypeName := fmt.Sprintf("%s%sOption", c.Name, optionType)
		params = append(params, fmt.Sprintf("opts ...%s", optionTypeName))
	}

	return params
}

func getRequiredAttributes(a *allowedAttributeList) []allowedAttribute {
	var required []allowedAttribute
	for _, attr := range *a {
		if !attr.Optional {
			required = append(required, attr)
		}
	}
	return required
}

func toLowerCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = rune(strings.ToLower(string(runes[0]))[0])
	return string(runes)
}
