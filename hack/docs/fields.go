//go:build !fields

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

func getDescFromField(field map[string]any) string {
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

func getKeyValueFieldTypes(field map[string]any) (string, string) {
	keyType, valType := "string", "string"
	addProps := field["additionalProperties"].(map[string]any)
	if val, ok := addProps["type"]; ok {
		keyType = val.(string)
	}
	if val, ok := addProps["format"]; ok {
		valType = val.(string)
	}
	return keyType, valType
}

func getObjectType(field map[string]any, addToQueue func(string)) string {
	objTypeRaw := field["type"].(string)
	if objTypeRaw == "array" {
		if ref, ok := field["items"].(map[string]any)["$ref"]; ok {
			refString := ref.(string)[14:]
			addToQueue(refString)

			name := getNameFromFullName(refString)
			if refString == "io.argoproj.workflow.v1alpha1.ParallelSteps" {
				return fmt.Sprintf("`Array<Array<`%s`>>`", link("`WorkflowStep`", "#"+strings.ToLower("WorkflowStep")))
			}
			return fmt.Sprintf("`Array<`%s`>`", link(fmt.Sprintf("`%s`", name), "#"+strings.ToLower(name)))
		}
		fullName := field["items"].(map[string]any)["type"].(string)
		return fmt.Sprintf("`Array< %s >`", getNameFromFullName(fullName))
	} else if objTypeRaw == "object" {
		if ref, ok := field["additionalProperties"].(map[string]any)["$ref"]; ok {
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

func sortedMapInterfaceKeys(in map[string]any) []string {
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
	defs       map[string]any
}

type Set map[string]bool

func (c *DocGeneratorContext) addToIndex(indexName, fileName string) {
	if set, ok := c.index[indexName]; ok {
		set[fileName] = true
	} else {
		c.index[indexName] = make(Set)
		c.index[indexName][fileName] = true
	}
}

func NewDocGeneratorContext() *DocGeneratorContext {
	return &DocGeneratorContext{
		doneFields: make(Set),
		queue: []string{
			"io.argoproj.workflow.v1alpha1.Workflow", "io.argoproj.workflow.v1alpha1.CronWorkflow",
			"io.argoproj.workflow.v1alpha1.WorkflowTemplate", "io.argoproj.workflow.v1alpha1.WorkflowEventBinding",
			"io.argoproj.workflow.v1alpha1.InfoResponse",
		},
		external: []string{},
		index:    make(map[string]Set),
		jsonName: make(map[string]string),
		defs:     make(map[string]any),
	}
}

func (c *DocGeneratorContext) loadFiles() {
	bytes, err := os.ReadFile("api/openapi-spec/swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := make(map[string]any)
	err = json.Unmarshal(bytes, &swagger)
	if err != nil {
		panic(err)
	}
	c.defs = swagger["definitions"].(map[string]any)

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
			case "ClusterWorkflowTemplate", "CronWorkflow", "Workflow", "WorkflowTemplate", "WorkflowEventBinding", "InfoResponse":
			default:
				continue FILES
			}
			c.addToIndex(kind, fileName)
		}

		r = regexp.MustCompile(`([a-zA-Z]+?):`)
		finds := r.FindAllStringSubmatch(string(bytes), -1)
		for _, find := range finds {
			c.addToIndex(find[1], fileName)
		}

		// Index by type name for specific patterns where field name matching is too broad.
		// MetricLabel is used in prometheus metrics config - match files with both "prometheus:" and "labels:".
		if _, hasPrometheus := c.index["prometheus"][fileName]; hasPrometheus {
			if _, hasLabels := c.index["labels"][fileName]; hasLabels {
				c.addToIndex("MetricLabel", fileName)
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
	obj, ok := c.defs[key].(map[string]any)
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
			// HACK: The "spec" field usually refers to a WorkflowSpec, but other CRDs
			// have different definitions, and the examples with "spec" aren't applicable.
			// Similarly, "labels" appears in metadata.labels for every workflow, but we
			// only want examples that actually use the field (e.g., MetricLabel in prometheus.labels).
			showExamples := true
			if jsonName == "spec" && name != "WorkflowSpec" && name != "CronWorkflowSpec" {
				showExamples = false
			}
			if jsonName == "labels" && name != "ObjectMeta" {
				showExamples = false
			}
			if showExamples {
				out += getExamples(set, "Examples with this field")
			}
		}
	}

	var properties map[string]any
	def, ok := c.defs[key]
	if !ok {
		return out
	}
	if props, ok := def.(map[string]any)["properties"]; ok {
		properties = props.(map[string]any)
	} else {
		return out
	}

	out += fieldTableHeader
	for _, jsonFieldName := range sortedMapInterfaceKeys(properties) {
		field := properties[jsonFieldName].(map[string]any)
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

func generateFieldsDocs() {
	println("generating docs/fields.md")
	c := NewDocGeneratorContext()
	err := os.WriteFile("docs/fields.md", []byte(c.generate()), 0o600)
	if err != nil {
		panic(err)
	}
}
