//go:build !fields
// +build !fields

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const sectionHeader = `

# %s
`

const fieldHeader = `

## %s

%s`

const fieldTableHeader = `

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|`

const tableRow = `
|` + "`%s`" + `|%s|%s|`

const depTableRow = `
|~~` + "`%s`" + `~~|~~%s~~|%s|`

// `markdown` attribute for MD in HTML: https://squidfunk.github.io/mkdocs-material/setup/extensions/python-markdown/#markdown-in-html
const dropdownOpener = `

<details markdown>
<summary>%s (click to open)</summary>`

const listElement = `

- %s`

const dropdownCloser = `
</details>`

func cleanTitle(title string) string {
	if index := strings.Index(title, "+g"); index != -1 {
		return title[:index]
	}
	return title
}

func cleanDesc(desc string) string {
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.ReplaceAll(desc, "  ", " ") // reduce multiple spaces to a single space
	dep := ""
	if index := strings.Index(desc, "DEPRECATED"); index != -1 {
		dep = " " + desc[:index]
	}

	if index := strings.Index(desc, "+patch"); index != -1 {
		desc = desc[:index]
	}
	if index := strings.Index(desc, "+proto"); index != -1 {
		desc = desc[:index]
	}
	if index := strings.Index(desc, "+option"); index != -1 {
		desc = desc[:index]
	}

	if dep != "" && !strings.Contains(desc, "DEPRECATED") {
		desc += dep
	}
	return desc
}

func getRow(name, objType, desc string) string {
	if index := strings.Index(desc, "DEPRECATED"); index != -1 {
		return fmt.Sprintf(depTableRow, name, objType, "~~"+desc[:index-1]+"~~ "+desc[index:])
	}
	return fmt.Sprintf(tableRow, name, objType, desc)
}

func getNameFromFullName(fullName string) string {
	split := strings.Split(fullName, ".")
	return split[len(split)-1]
}

func link(text, linkTo string) string {
	return fmt.Sprintf("[%s](%s)", text, linkTo)
}

func getDescFromField(field map[string]interface{}) string {
	if val, ok := field["description"]; ok {
		return cleanDesc(val.(string))
	} else if val, ok := field["title"]; ok {
		return cleanDesc(cleanTitle(val.(string)))
	}
	return "_No description available_"
}

func getExamples(examples Set, summary string) string {
	out := fmt.Sprintf(dropdownOpener, summary)
	for _, example := range sortedSetKeys(examples) {
		split := strings.Split(example, "/")
		name := split[len(split)-1]
		out += fmt.Sprintf(listElement, link(fmt.Sprintf("`%s`", name), "https://github.com/argoproj/argo-workflows/blob/main/"+example))
	}
	out += dropdownCloser
	return out
}

func getKeyValueFieldTypes(field map[string]interface{}) (string, string) {
	keyType, valType := "string", "string"
	addProps := field["additionalProperties"].(map[string]interface{})
	if val, ok := addProps["type"]; ok {
		keyType = val.(string)
	}
	if val, ok := addProps["format"]; ok {
		valType = val.(string)
	}
	return keyType, valType
}

func getObjectType(field map[string]interface{}, addToQueue func(string)) string {
	objTypeRaw := field["type"].(string)
	if objTypeRaw == "array" {
		if ref, ok := field["items"].(map[string]interface{})["$ref"]; ok {
			refString := ref.(string)[14:]
			addToQueue(refString)

			name := getNameFromFullName(refString)
			if refString == "io.argoproj.workflow.v1alpha1.ParallelSteps" {
				return fmt.Sprintf("`Array<Array<`%s`>>`", link(fmt.Sprintf("`%s`", "WorkflowStep"), fmt.Sprintf("#"+strings.ToLower("WorkflowStep"))))
			}
			return fmt.Sprintf("`Array<`%s`>`", link(fmt.Sprintf("`%s`", name), fmt.Sprintf("#"+strings.ToLower(name))))
		}
		fullName := field["items"].(map[string]interface{})["type"].(string)
		return fmt.Sprintf("`Array< %s >`", getNameFromFullName(fullName))
	} else if objTypeRaw == "object" {
		if ref, ok := field["additionalProperties"].(map[string]interface{})["$ref"]; ok {
			refString := ref.(string)[14:]
			addToQueue(refString)
			name := getNameFromFullName(refString)
			return link(fmt.Sprintf("`%s`", name), "#"+strings.ToLower(name))
		}
		key, val := getKeyValueFieldTypes(field)
		return fmt.Sprintf("`Map< %s , %s >`", key, val)
	} else if format, ok := field["format"].(string); ok {
		return fmt.Sprintf("`%s`", format)
	}
	return fmt.Sprintf("`%s`", field["type"].(string))
}

func glob(dir string, ext string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func sortedMapInterfaceKeys(in map[string]interface{}) []string {
	var stringList []string
	for key := range in {
		stringList = append(stringList, key)
	}
	sort.Strings(stringList)
	return stringList
}

func sortedSetKeys(in Set) []string {
	var stringList []string
	for key := range in {
		stringList = append(stringList, key)
	}
	sort.Strings(stringList)
	return stringList
}

type DocGeneratorContext struct {
	doneFields Set
	queue      []string
	external   []string
	index      map[string]Set
	jsonName   map[string]string
	defs       map[string]interface{}
}

type Set map[string]bool

func NewDocGeneratorContext() *DocGeneratorContext {
	return &DocGeneratorContext{
		doneFields: make(Set),
		queue: []string{
			"io.argoproj.workflow.v1alpha1.Workflow", "io.argoproj.workflow.v1alpha1.CronWorkflow",
			"io.argoproj.workflow.v1alpha1.WorkflowTemplate",
		},
		external: []string{},
		index:    make(map[string]Set),
		jsonName: make(map[string]string),
		defs:     make(map[string]interface{}),
	}
}

func (c *DocGeneratorContext) loadFiles() {
	bytes, err := os.ReadFile("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := make(map[string]interface{})
	err = json.Unmarshal(bytes, &swagger)
	if err != nil {
		panic(err)
	}
	c.defs = swagger["definitions"].(map[string]interface{})

	files, err := glob("examples/", ".yaml")
	if err != nil {
		panic(err)
	}
FILES:
	for _, fileName := range files {
		bytes, err := os.ReadFile(filepath.Clean(fileName))
		if err != nil {
			panic(err)
		}

		r := regexp.MustCompile(`kind: ([a-zA-Z]+)`)
		matches := r.FindAllStringSubmatch(string(bytes), -1)
		for _, m := range matches {
			kind := m[1]
			switch kind {
			case "ClusterWorkflowTemplate", "CronWorkflow", "Workflow", "WorkflowTemplate":
			default:
				continue FILES
			}
			if set, ok := c.index[kind]; ok {
				set[fileName] = true
			} else {
				c.index[kind] = make(Set)
				c.index[kind][fileName] = true
			}
		}

		r = regexp.MustCompile(`([a-zA-Z]+?):`)
		finds := r.FindAllStringSubmatch(string(bytes), -1)
		for _, find := range finds {
			if set, ok := c.index[find[1]]; ok {
				set[fileName] = true
			} else {
				c.index[find[1]] = make(Set)
				c.index[find[1]][fileName] = true
			}
		}
	}
}

func (c *DocGeneratorContext) addToQueue(ref, jsonFieldName string) {
	if ref == "io.argoproj.workflow.v1alpha1.ParallelSteps" {
		ref = "io.argoproj.workflow.v1alpha1.WorkflowStep"
	}
	if _, ok := c.doneFields[ref]; !ok {
		c.doneFields[ref] = true
		c.jsonName[ref] = jsonFieldName
		if strings.Contains(ref, "io.argoproj.workflow") {
			c.queue = append(c.queue, ref)
		} else {
			c.external = append(c.external, ref)
		}
	}
}

func (c *DocGeneratorContext) getDesc(key string) string {
	obj, ok := c.defs[key].(map[string]interface{})
	if !ok {
		return "_No description available_"
	}
	if val, ok := obj["description"]; ok {
		return cleanDesc(val.(string))
	} else if val, ok := obj["title"]; ok {
		return cleanDesc(cleanTitle(val.(string)))
	}
	return "_No description available_"
}

func (c *DocGeneratorContext) getTemplate(key string) string {
	name := getNameFromFullName(key)
	out := fmt.Sprintf(fieldHeader, name, c.getDesc(key))

	if set, ok := c.index[name]; ok {
		out += getExamples(set, "Examples")
	}
	if jsonName, ok := c.jsonName[key]; ok {
		if set, ok := c.index[jsonName]; ok {
			out += getExamples(set, "Examples with this field")
		}
	}

	var properties map[string]interface{}
	def, ok := c.defs[key]
	if !ok {
		return out
	}
	if props, ok := def.(map[string]interface{})["properties"]; ok {
		properties = props.(map[string]interface{})
	} else {
		return out
	}

	out += fieldTableHeader
	for _, jsonFieldName := range sortedMapInterfaceKeys(properties) {
		field := properties[jsonFieldName].(map[string]interface{})
		if ref, ok := field["$ref"]; ok {
			refString := ref.(string)[14:]
			c.addToQueue(refString, jsonFieldName)

			desc := getDescFromField(field)
			refName := getNameFromFullName(refString)
			out += getRow(jsonFieldName, link(fmt.Sprintf("`%s`", refName), "#"+strings.ToLower(refName)), cleanDesc(desc))
		} else {
			objType := getObjectType(field, func(ref string) { c.addToQueue(ref, jsonFieldName) })
			desc := getDescFromField(field)
			out += getRow(jsonFieldName, objType, cleanDesc(desc))
		}
	}
	return out
}

func (c *DocGeneratorContext) generate() string {
	c.loadFiles()

	out := "# Field Reference"
	for len(c.queue) > 0 {
		var temp string
		temp, c.queue = c.queue[0], c.queue[1:]
		out += c.getTemplate(temp)
	}

	out += fmt.Sprintf(sectionHeader, "External Fields")
	for len(c.external) > 0 {
		var temp string
		temp, c.external = c.external[0], c.external[1:]
		out += c.getTemplate(temp)
	}

	out += "\n"
	return out
}

func generateDocs() {
	println("generating docs/fields.md")
	c := NewDocGeneratorContext()
	err := os.WriteFile("docs/fields.md", []byte(c.generate()), 0o600)
	if err != nil {
		panic(err)
	}
}
