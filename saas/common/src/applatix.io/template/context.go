package template

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"applatix.io/axerror"
	"applatix.io/common"
)

// TemplateBuildContext is a context for template building
type TemplateBuildContext struct {
	Templates      map[string]TemplateIf       // name to template mapping
	PathToTemplate map[string][]TemplateIf     // maps template path to a list of service templates in that file
	Results        map[string]ValidationResult // map of processed templates
	IgnoreErrors   bool
	Strict         bool
	Repo           string
	Branch         string
	Revision       string

	depth          int               // used to detect infinite recursion
	firstErr       *ValidationResult // store the first error that occurred
	templateToPath map[string]string // map a template name to its file path
}

type ValidationResult struct {
	Template TemplateIf
	AXErr    *axerror.AXError
}

func NewTemplateBuildContext() *TemplateBuildContext {
	context := TemplateBuildContext{
		Templates:      map[string]TemplateIf{},
		PathToTemplate: map[string][]TemplateIf{},
		Results:        map[string]ValidationResult{},
		IgnoreErrors:   true,
		Strict:         true,
		templateToPath: map[string]string{},
	}
	return &context
}

// AddToContext adds the service template to the context during parsing (not yet validation).
// If the template had an error (such as a parse error or duplicate template name)
// will immediately mark it as processed, so it does not get processed again
func (ctx *TemplateBuildContext) AddToContext(st TemplateIf, filePath string, axErr *axerror.AXError) {
	templateName := st.GetName()
	ctx.Templates[templateName] = st
	if _, exists := ctx.PathToTemplate[filePath]; !exists {
		ctx.PathToTemplate[filePath] = []TemplateIf{}
	}
	ctx.PathToTemplate[filePath] = append(ctx.PathToTemplate[filePath], st)
	ctx.templateToPath[templateName] = filePath
	if axErr != nil {
		ctx.MarkProcessed(st, axErr)
	}
}

// Validate will iterate the templates and validate each within the context of other templates
func (ctx *TemplateBuildContext) Validate() *axerror.AXError {
	// The order here matters. We need to process the most composable templates first, before moving to the higher layers
	// 1. containers
	// 2. deployments
	// 3. workflows
	// 4. policy | project | fixture

	validateOrder := []string{
		TemplateTypeContainer,
		TemplateTypeDeployment,
		TemplateTypeWorkflow,
		TemplateTypeFixture,
		TemplateTypePolicy,
		TemplateTypeProject,
	}

	for _, templateType := range validateOrder {
		for _, st := range ctx.GetTemplates([]string{templateType}, false) {
			ctx.depth = 0
			axErr := st.Validate()
			if axErr == nil {
				axErr = st.ValidateContext(ctx)
			}
			st.setRepoInfo(ctx.Repo, ctx.Branch, ctx.Revision)
			ctx.MarkProcessed(st, axErr)
			if axErr != nil {
				if !ctx.IgnoreErrors {
					return axErr
				}
				log.Printf("Template %s had error: %v", st.GetName(), axErr)
			}
		}
	}
	return nil
}

// MarkProcessed marks a template as processed so we do not process it a second time
func (ctx *TemplateBuildContext) MarkProcessed(st TemplateIf, axErr *axerror.AXError) {
	stName := st.GetName()
	_, exists := ctx.Results[stName]
	if !exists || axErr != nil {
		// we have never processed this template. or we have processed it already but
		// now want to mark it as error. This can happen when we have duplicate template names
		result := ValidationResult{
			Template: st,
			AXErr:    axErr,
		}
		ctx.Results[stName] = result
		if axErr != nil && ctx.firstErr == nil {
			ctx.firstErr = &result
		}
	}
}

// FirstError returns the first error that occurred during parsing/validation along with its file path
func (ctx *TemplateBuildContext) FirstError() (*ValidationResult, string) {
	if ctx.firstErr == nil {
		return nil, ""
	}
	return ctx.firstErr, ctx.templateToPath[ctx.firstErr.Template.GetName()]
}

// Processed returns whether or not the template name was already processed with a result
func (ctx *TemplateBuildContext) Processed(templateName string) bool {
	_, ok := ctx.Results[templateName]
	return ok
}

// ParseDirectory walks a directory of yaml files and adds the contents of each into the context
func (ctx *TemplateBuildContext) ParseDirectory(dirPath string) *axerror.AXError {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		fileExt := filepath.Ext(info.Name())
		if fileExt != ".yaml" && fileExt != ".yml" {
			return nil
		}
		body, err := ioutil.ReadFile(path)
		if err != nil && !ctx.IgnoreErrors {
			err := fmt.Errorf("Can't read from file: %s, err: %s\n", path, err)
			return err
		}
		axErr := ctx.ParseFile(body, path)
		if axErr != nil && !ctx.IgnoreErrors {
			return axErr
		}
		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil && !ctx.IgnoreErrors {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef(err.Error())
	}
	return nil
}

var (
	yamlSeparator = regexp.MustCompile("\\n---")
)

// ParseFile parses the contents of a .yaml file (which may contain multiple service templates)
// and adds each template to the context. If parsing encountered any errors, will still add to the
// context, but mark the template as processed (with error).
func (ctx *TemplateBuildContext) ParseFile(body []byte, filePath string) *axerror.AXError {
	//objByteArray := bytes.Split(body, []byte("---"))
	yamlArray := yamlSeparator.Split(string(body), -1)
	for _, yamlStr := range yamlArray {
		if strings.TrimSpace(yamlStr) == "" {
			continue
		}
		tmpl, axParseErr := UnmarshalTemplate([]byte(yamlStr), ctx.Strict)
		if tmpl == nil {
			errMsg := fmt.Sprintf("Failed to parse template body in %s", filePath)
			if axParseErr != nil {
				errMsg += fmt.Sprintf(": %v", axParseErr)
			}
			axErr := axerror.ERR_API_INVALID_PARAM.NewWithMessagef(errMsg)
			var st TemplateIf = &BaseTemplate{}
			ctx.AddToContext(st, filePath, axParseErr)
			if !ctx.IgnoreErrors {
				return axErr
			}
			if common.DebugLog != nil {
				common.DebugLog.Println(axErr)
			}
			continue
		}
		templateName := tmpl.GetName()
		if _, exists := ctx.Templates[templateName]; exists {
			axErr := axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Duplicate template name: %s", templateName)
			ctx.MarkProcessed(tmpl, axErr)
			if !ctx.IgnoreErrors {
				return axErr
			}
			if common.DebugLog != nil {
				common.DebugLog.Println(axErr)
			}
			continue
		}
		ctx.AddToContext(tmpl, filePath, axParseErr)
		if axParseErr != nil {
			if !ctx.IgnoreErrors {
				return axParseErr
			}
			if common.DebugLog != nil {
				common.DebugLog.Println(axParseErr)
			}
		}
	}
	return nil
}

// GetTemplates returns a slice of templates of the given type. Optonally inidicate if we only want valid templates
func (ctx *TemplateBuildContext) GetTemplates(templateTypes []string, valid ...bool) []TemplateIf {
	templates := make([]TemplateIf, 0)
	onlyValid := bool(len(valid) > 0 && valid[0] == true)
	filter := make(map[string]bool)
	for _, tType := range templateTypes {
		filter[tType] = true
	}

	if onlyValid {
		for _, res := range ctx.Results {
			st := res.Template
			if !filter[st.GetType()] {
				continue
			}
			if res.AXErr != nil {
				continue
			}
			templates = append(templates, st)
		}
	} else {
		for _, st := range ctx.Templates {
			if !filter[st.GetType()] {
				continue
			}
			templates = append(templates, st)
		}
	}
	return templates
}

// GetServiceTemplates returns a slice of validated service templates (container, workflow, deployment)
func (ctx *TemplateBuildContext) GetServiceTemplates() []TemplateIf {
	tmpls := make([]TemplateIf, 0)
	for _, tmpl := range ctx.GetTemplates([]string{TemplateTypeContainer, TemplateTypeDeployment, TemplateTypeWorkflow}, true) {
		tmpls = append(tmpls, tmpl)
	}
	return tmpls
}

// GetPolicyTemplates returns a slice of validated policy templates
func (ctx *TemplateBuildContext) GetPolicyTemplates() []*PolicyTemplate {
	tmpls := make([]*PolicyTemplate, 0)
	for _, tmpl := range ctx.GetTemplates([]string{TemplateTypePolicy}, true) {
		tmpls = append(tmpls, tmpl.(*PolicyTemplate))
	}
	return tmpls
}

// GetFixtureTemplates returns a slice of validated fixture templates
func (ctx *TemplateBuildContext) GetFixtureTemplates() []*FixtureTemplate {
	tmpls := make([]*FixtureTemplate, 0)
	for _, tmpl := range ctx.GetTemplates([]string{TemplateTypeFixture}, true) {
		tmpls = append(tmpls, tmpl.(*FixtureTemplate))
	}
	return tmpls
}

// GetProjectTemplates returns a slice of validated project templates
func (ctx *TemplateBuildContext) GetProjectTemplates() []*ProjectTemplate {
	tmpls := make([]*ProjectTemplate, 0)
	for _, tmpl := range ctx.GetTemplates([]string{TemplateTypeProject}, true) {
		tmpls = append(tmpls, tmpl.(*ProjectTemplate))
	}
	return tmpls
}
