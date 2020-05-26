package main

import "strings"

// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#specifying-a-structural-schema
var intOrString = obj{"x-kubernetes-int-or-string": true}
var any = obj{"x-kubernetes-preserve-unknown-fields": true}

type structuralSchemaContext struct {
	swagger obj
	simple  bool
}

func (c structuralSchemaContext) structuralSchemaByName(name string) obj {
	switch name {
	case "io.argoproj.workflow.v1alpha1.Item":
		return any
	case "io.argoproj.workflow.v1alpha1.Template":
		if c.simple {
			return any
		}
	case "io.k8s.apimachinery.pkg.api.resource.Quantity":
		return intOrString
	case "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":
		return obj{"type": "object"}
	}
	return c.swagger["definitions"].(obj)[name].(obj)
}

func (c structuralSchemaContext) structuralSchema(definition obj) obj {
	if ref, ok := definition["$ref"]; ok {
		return c.structuralSchema(c.structuralSchemaByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
	}
	if _, ok := definition["anyOf"]; ok {
		return any
	}
	scrubStructuralSchema(definition)
	if items, ok := definition["items"].(obj); ok {
		if _, ok := items["anyOf"]; ok {
			definition["items"] = any
		} else if ref, ok := items["$ref"]; ok {
			definition["items"] = c.structuralSchema(c.structuralSchemaByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
		}
	}
	if properties, ok := definition["properties"].(obj); ok {
		for name, value := range properties {
			if _, ok := value.(obj)["anyOf"]; ok {
				definition[name] = any
			} else {
				properties[name] = c.structuralSchema(value.(obj))
			}
		}
	}
	if properties, ok := definition["additionalProperties"].(obj); ok {
		definition["additionalProperties"] = c.structuralSchema(properties)
	}
	if format, ok := definition["format"]; ok && format == "int-or-string" {
		return intOrString
	}
	return definition
}

func scrubStructuralSchema(definition obj) {
	delete(definition, "description")
	delete(definition, "x-kubernetes-group-version-kind")
	delete(definition, "x-kubernetes-patch-merge-key")
	delete(definition, "x-kubernetes-patch-strategy")
	delete(definition, "x-kubernetes-list-map-keys")
	delete(definition, "x-kubernetes-list-type")
	properties, ok := definition["properties"].(obj)
	if ok {
		for _, v := range properties {
			scrubStructuralSchema(v.(obj))
		}
	}
}
