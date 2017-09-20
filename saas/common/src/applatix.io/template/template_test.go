package template_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"applatix.io/axamm/utils"
	"applatix.io/axops/fixture"
	"applatix.io/axops/service"
	"applatix.io/template"
	"github.com/stretchr/testify/assert"
)

var (
	sourceRoot  string
	badYAMLdir  string
	goodYAMLdir string
	yaml20dir   string
)

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic(path)
}

func FindSourceRoot() string {
	_, filename, _, _ := runtime.Caller(1)
	dirName := filepath.Dir(filename)
	for {
		gitDir := filepath.Join(dirName, ".git")
		if exists(gitDir) {
			return dirName
		}
		if dirName == "/" {
			log.Println("Failed to determine source root dir")
			os.Exit(1)
		}
		dirName = filepath.Clean(filepath.Join(dirName, ".."))
	}
}

func init() {
	sourceRoot = FindSourceRoot()
	badYAMLdir = path.Join(sourceRoot, "saas/common/src/applatix.io/template/bad")
	goodYAMLdir = path.Join(sourceRoot, "saas/common/src/applatix.io/template/examples")
	yaml20dir = path.Join(sourceRoot, ".argo")
	utils.InitLoggers("")
}
func TestEmbeddedTemplate(t *testing.T) {
	matches, err := filepath.Glob(goodYAMLdir + "/*.yaml")
	assert.Nil(t, err)
	if len(matches) == 0 {
		t.Fatalf("No YAMLs found")
	}
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = false
	for _, match := range matches {
		body, err := ioutil.ReadFile(match)
		assert.Nil(t, err)
		axErr := ctx.ParseFile(body, match)
		if axErr != nil {
			t.Fatal(axErr)
		}
	}
	if len(ctx.Results) != 0 {
		t.Fatalf("Expected no parse errors: %s", ctx.Results)
	}
	ctx.Validate()

	results := ctx.Results
	//results := []template.ValidationResult{ctx.Results["wokflow-passing-fixtures-volumes"]}
	//results := []template.ValidationResult{ctx.Results["inlined-container-and-fixture-name-collision"]}
	//results := []template.ValidationResult{ctx.Results["export_artifact"]}
	//results := []template.ValidationResult{ctx.Results["policy-container-with-default-secret"]}
	//results := []template.ValidationResult{ctx.Results["workflow-with-fixtures"]}
	//results := []template.ValidationResult{ctx.Results["example-workflow-inline"]}
	//results := []template.ValidationResult{ctx.Results["claudia-userdb"]}
	//results := []template.ValidationResult{ctx.Results["example-checkout"]}

	for _, result := range results {
		switch result.Template.GetType() {
		case template.TemplateTypeContainer, template.TemplateTypeDeployment, template.TemplateTypeWorkflow:
			// Attempt to generate the embedded template
			eTmpl, axErr := service.EmbedServiceTemplate(result.Template, ctx)
			assert.Nil(t, axErr)
			logTemplate(t, eTmpl)
			assert.Equal(t, result.Template.GetName(), eTmpl.GetName())
			assert.Equal(t, result.Template.GetType(), eTmpl.GetType())
			if result.Template.GetOutputs() != nil && result.Template.GetOutputs().Artifacts != nil {
				assert.NotNil(t, eTmpl.GetOutputs())
				assert.NotNil(t, eTmpl.GetOutputs().Artifacts)
				assert.Equal(t, len(result.Template.GetOutputs().Artifacts), len(eTmpl.GetOutputs().Artifacts))
			}

			// Make sure we can json marshall it
			data, err := json.Marshal(eTmpl)
			assert.Nil(t, err)

			// Make sure we can unmarshal it back
			eTmpl2, axErr := service.UnmarshalEmbeddedTemplate(data)
			if axErr != nil {
				t.Errorf("Failed to unmarshal %s: %s", result.Template.GetName(), err)
				log.Println(string(data))
				continue
			}
			assert.Equal(t, result.Template.GetName(), eTmpl2.GetName())
			assert.Equal(t, result.Template.GetType(), eTmpl2.GetType())
			if result.Template.GetOutputs() != nil && result.Template.GetOutputs().Artifacts != nil {
				assert.NotNil(t, eTmpl2.GetOutputs())
				assert.NotNil(t, eTmpl2.GetOutputs().Artifacts)
				assert.Equal(t, len(result.Template.GetOutputs().Artifacts), len(eTmpl2.GetOutputs().Artifacts))
			}

			logTemplate(t, eTmpl2)
			switch result.Template.GetType() {
			case template.TemplateTypeDeployment:
				dt := eTmpl.(*service.EmbeddedDeploymentTemplate)
				dt2 := eTmpl2.(*service.EmbeddedDeploymentTemplate)
				for cName, svc := range dt2.Containers {
					assert.NotNil(t, svc.Template)
					before := dt.Containers[cName].Template.(*service.EmbeddedContainerTemplate)
					after := dt2.Containers[cName].Template.(*service.EmbeddedContainerTemplate)
					assert.Equal(t, before.Image, after.Image)
					assert.Equal(t, before.Name, after.Name)
					assert.Equal(t, before.ID, after.ID)
					assert.Equal(t, before.ImagePullPolicy, after.ImagePullPolicy)
				}
			}

			// Finally attempt to marshall the unmarshaled one again
			_, err = json.Marshal(eTmpl2)
			assert.Nil(t, err)

		}
	}
}

func TestServiceMarshaling(t *testing.T) {
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = false
	ctx.ParseDirectory(yaml20dir)
	axErr := ctx.Validate()
	assert.Nil(t, axErr)

	results := ctx.Results
	//results := []template.ValidationResult{ctx.Results["wokflow-passing-fixtures-volumes"]}
	//results := []template.ValidationResult{ctx.Results["inlined-container-and-fixture-name-collision"]}
	//results := []template.ValidationResult{ctx.Results["export_artifact"]}
	//results := []template.ValidationResult{ctx.Results["policy-container-with-default-secret"]}
	//results := []template.ValidationResult{ctx.Results["workflow-with-fixtures"]}
	//results := []template.ValidationResult{ctx.Results["example-workflow-inline"]}
	//results := []template.ValidationResult{ctx.Results["test-workflow-fixture-request"]}

	for _, result := range results {
		switch result.Template.GetType() {
		case template.TemplateTypeContainer, template.TemplateTypeDeployment, template.TemplateTypeWorkflow:
			// Attempt to generate the embedded template
			eTmpl, axErr := service.EmbedServiceTemplate(result.Template, ctx)
			assert.Nil(t, axErr)
			//t.Log(json.MarshalIndent(eTmpl, " ", "  "))

			svc := service.Service{
				Template: eTmpl,
			}

			// Make sure we can json marshall it
			data, err := json.MarshalIndent(svc, " ", "    ")
			assert.Nil(t, err)

			// Make sure we can unmarshal it back
			var svc2 service.Service
			err = json.Unmarshal(data, &svc2)
			if err != nil {
				t.Errorf("Failed to unmarshal service %s: %s", result.Template.GetName(), err)
				log.Println(string(data))
			}
			// Checks if fixtures section is still valid
			if result.Template.GetType() == template.TemplateTypeWorkflow {
				wf := result.Template.(*template.WorkflowTemplate)
				if len(wf.Fixtures) > 0 {
					ewf := svc.Template.(*service.EmbeddedWorkflowTemplate)
					assert.True(t, len(wf.Fixtures) == len(ewf.Fixtures))
					for _, parallelFixtures := range ewf.Fixtures {
						for _, val := range parallelFixtures {
							assert.NotNil(t, val)
						}
					}
				}
			}

			// Print the service
			data, _ = json.MarshalIndent(svc2, " ", "    ")
			log.Println(string(data))
		}
	}
}

func TestValidYAMLs(t *testing.T) {
	matches, err := filepath.Glob(goodYAMLdir + "/*.yaml")
	assert.Nil(t, err)
	if len(matches) == 0 {
		t.Fatalf("No YAMLs found")
	}
	ctx := template.NewTemplateBuildContext()
	for _, match := range matches {
		body, err := ioutil.ReadFile(match)
		assert.Nil(t, err)
		ctx.ParseFile(body, match)
	}
	if len(ctx.Results) != 0 {
		t.Fatal("Expected no parse errors")
	}
	ctx.Validate()
	for tmplName, result := range ctx.Results {
		assert.Nil(t, result.AXErr, "%s unexpectedly failed", tmplName)
	}
}

func TestDetectYAMLErrors(t *testing.T) {
	matches, err := filepath.Glob(badYAMLdir + "/*.yaml")
	assert.Nil(t, err)
	if len(matches) == 0 {
		t.Fatalf("No YAMLs found")
	}
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = true
	for _, match := range matches {
		body, err := ioutil.ReadFile(match)
		assert.Nil(t, err)
		ctx.ParseFile(body, match)
	}
	if len(ctx.Results) == 0 {
		// There are a few bad examples that are caught during parsing. Make sure we find them
		t.Fatalf("Expected 1+ parse errors")
	}
	for tmplName, result := range ctx.Results {
		if result.AXErr == nil {
			t.Errorf("%s unexpectedly passed", tmplName)
		}
	}
	ctx.Validate()
	for tmplName, result := range ctx.Results {
		if result.AXErr == nil && !strings.HasSuffix(result.Template.GetName(), "-ok") {
			t.Errorf("%s unexpectedly passed", tmplName)
		}
	}
}

func TestArgumentSubstitution(t *testing.T) {
	matches, err := filepath.Glob(yaml20dir + "/*.yaml")
	assert.Nil(t, err)
	if len(matches) == 0 {
		t.Fatalf("No YAMLs found")
	}
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = false
	axErr := ctx.ParseDirectory(yaml20dir)
	assert.Nil(t, axErr)
	axErr = ctx.ParseDirectory(goodYAMLdir)
	assert.Nil(t, axErr)
	if len(ctx.Results) != 0 {
		t.Fatalf("Expected no parse errors: %s", ctx.Results)
	}
	axErr = ctx.Validate()
	assert.Nil(t, axErr)
	var tmpl template.TemplateIf
	var subTmpl, eTmpl service.EmbeddedTemplateIf

	tmpl = ctx.Results["test-container-with-input-parameter"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	var sixty = "60"
	subTmpl, axErr = eTmpl.SubstituteArguments(template.Arguments{"parameters.SLEEP": &sixty})
	assert.Nil(t, axErr)
	data, _ := json.MarshalIndent(subTmpl, " ", "    ")
	log.Println(string(data))

	tmpl = ctx.Results["test-container-with-default-input-parameter"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	subTmpl, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)
	data, _ = json.MarshalIndent(subTmpl, " ", "    ")
	log.Println(string(data))

	tmpl = ctx.Results["test-workflow-with-child-argument"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	subTmpl, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)
	data, _ = json.MarshalIndent(subTmpl, " ", "    ")
	log.Println(string(data))

	// Test we return error when input was not satisfied
	tmpl = ctx.Results["test-container-with-input-parameter"].Template
	inputs := tmpl.GetInputs()
	log.Println(inputs)
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	inputs = eTmpl.GetInputs()
	log.Println(inputs)
	subTmpl, axErr = eTmpl.SubstituteArguments(nil)
	if axErr == nil {
		t.Errorf("Expected error")
		data, _ = json.MarshalIndent(subTmpl, " ", "    ")
		log.Println(string(data))
	}

	// Test argument substitution with artifacts
	tmpl = ctx.Results["test-workflow-passing-artifacts"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	subTmpl, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)
	logTemplate(t, subTmpl)
	wfTmpl := subTmpl.(*service.EmbeddedWorkflowTemplate)
	child := wfTmpl.Steps[1]["STEP2"].Template
	childArtifacts := child.GetInputs().Artifacts
	assert.True(t, strings.HasPrefix(childArtifacts["BIN-INPUT"].From, "%%service."))

	// Test argument substitution with nested artifacts
	tmpl = ctx.Results["test-workflow-nested-artifacts"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	subTmpl, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)
	logTemplate(t, subTmpl)
	wfTmpl = subTmpl.(*service.EmbeddedWorkflowTemplate)
	child = wfTmpl.Steps[1]["STEP2"].Template
	childArtifacts = child.GetInputs().Artifacts
	assert.True(t, strings.HasPrefix(childArtifacts["BIN-INPUT"].From, "%%service."))

	// test inlined containers
	tmpl = ctx.Results["test-workflow-passing-artifacts-inlined"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	logTemplate(t, eTmpl)
	assert.Nil(t, axErr)
	_, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)

	// test with presence of yaml multiline scalar
	tmpl = ctx.Results["workflow-yaml-multiline"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	logTemplate(t, eTmpl)
	assert.Nil(t, axErr)
	_, axErr = eTmpl.SubstituteArguments(nil)
	assert.Nil(t, axErr)

	// test substitution of various deployment fields
	tmpl = ctx.Results["paramaterize-deployment-fields"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	eTmpl, axErr = eTmpl.SubstituteArguments(nil)
	logTemplate(t, eTmpl)
	assert.Nil(t, axErr)
	dTmpl := eTmpl.(*service.EmbeddedDeploymentTemplate)
	assert.Equal(t, dTmpl.ApplicationName, "hello-world")
	assert.Equal(t, dTmpl.DeploymentName, "hello-world")
	assert.Equal(t, dTmpl.ExternalRoutes[0].TargetPort, "1234")
	assert.Equal(t, dTmpl.InternalRoutes[0].Ports[0].Port, "1234")
	assert.Equal(t, dTmpl.InternalRoutes[0].Ports[0].TargetPort, "1234")
	assert.Equal(t, dTmpl.InternalRoutes[0].Ports[0].TargetPort, "1234")
	axErr = dTmpl.Validate(true)
	assert.Nil(t, axErr)

	// test validate method still catches errors after invalid substitution by user
	tmpl = ctx.Results["paramaterize-deployment-fields"].Template
	log.Println(tmpl)
	eTmpl, axErr = service.EmbedServiceTemplate(tmpl, ctx)
	assert.Nil(t, axErr)
	abc := "abc"
	eTmpl, axErr = eTmpl.SubstituteArguments(template.Arguments{"parameters.PORT": &abc})
	assert.Nil(t, axErr)
	dTmpl = eTmpl.(*service.EmbeddedDeploymentTemplate)
	axErr = dTmpl.Validate(true)
	assert.NotNil(t, axErr)
}

func logTemplate(t *testing.T, tmpl service.EmbeddedTemplateIf) {
	tmplStr, _ := json.MarshalIndent(tmpl, " ", "  ")
	t.Log(string(tmplStr))
}

func newString(s interface{}) string {
	ns := fmt.Sprintf("%s", s)
	return ns
}

func TestTemplateSerialization(t *testing.T) {
	matches, err := filepath.Glob(yaml20dir + "/*.yaml")
	assert.Nil(t, err)
	if len(matches) == 0 {
		t.Fatalf("No YAMLs found")
	}
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = false
	ctx.Repo = "http://github.com/Applatix/argos.git"
	ctx.Branch = "master"
	ctx.Revision = "1234567890abcdef1234567890abcdef12345678"
	ctx.ParseDirectory(yaml20dir)
	axErr := ctx.Validate()
	assert.Nil(t, axErr)

	results := ctx.Results
	//results := []template.ValidationResult{ctx.Results["wokflow-passing-fixtures-volumes"]}
	//results := []template.ValidationResult{ctx.Results["inlined-container-and-fixture-name-collision"]}
	//results := []template.ValidationResult{ctx.Results["export_artifact"]}
	//results := []template.ValidationResult{ctx.Results["policy-container-with-default-secret"]}
	//results := []template.ValidationResult{ctx.Results["workflow-with-fixtures"]}
	//results := []template.ValidationResult{ctx.Results["example-workflow-inline"]}

	for _, result := range results {
		switch result.Template.GetType() {
		case template.TemplateTypeContainer, template.TemplateTypeDeployment, template.TemplateTypeWorkflow:
			eTmpl, axErr := service.EmbedServiceTemplate(result.Template, ctx)
			assert.Nil(t, axErr)
			assert.NotNil(t, eTmpl)
			tmplMap := service.TemplateToMap(eTmpl)
			deserialized, axErr := service.MapToTemplate(tmplMap)
			assert.Nil(t, axErr)
			assert.Equal(t, deserialized.GetName(), result.Template.GetName())
			assert.Equal(t, deserialized.GetID(), template.GenerateTemplateUUID(ctx.Repo, ctx.Branch, eTmpl.GetName()))
			assert.Equal(t, deserialized.GetRepo(), ctx.Repo)
			assert.Equal(t, deserialized.GetBranch(), ctx.Branch)
			_, err := json.Marshal(deserialized)
			assert.Nil(t, err)

		case template.TemplateTypeFixture:
			fTmpl := result.Template.(*template.FixtureTemplate)
			// fTmpl.ID = template.GenerateTemplateUUID(ctx.Repo, ctx.Branch, f.GetName())
			// fTmpl.Repo = ctx.Repo
			// fTmpl.Branch = ctx.Branch
			tmplDB, axErr := fixture.ToTemplateDB(fTmpl)
			assert.Nil(t, axErr)
			deserialized, axErr := tmplDB.Template()
			assert.Nil(t, axErr)
			assert.Equal(t, deserialized.GetName(), result.Template.GetName())
			assert.Equal(t, deserialized.GetID(), template.GenerateTemplateUUID(ctx.Repo, ctx.Branch, fTmpl.GetName()))
			assert.Equal(t, deserialized.GetRepo(), ctx.Repo)
			assert.Equal(t, deserialized.GetBranch(), ctx.Branch)
			_, err := json.Marshal(deserialized)
			assert.Nil(t, err)
			//log.Println(string(tmplBytes))
		}
	}
}

func TestTemplateDefaultParamValue(t *testing.T) {
	ctx := template.NewTemplateBuildContext()
	ctx.IgnoreErrors = false

	// Verify that if input parameter's default field is null, it is omitted
	axErr := ctx.ParseFile([]byte(`
type: container
version: 1
name: null-default-val
image: alpine:latest
inputs:
  parameters:
    MYPARAM:
      default:
`), "")
	assert.Nil(t, axErr)
	axErr = ctx.Validate()
	assert.Nil(t, axErr)
	eTmpl, axErr := service.EmbedServiceTemplate(ctx.Templates["null-default-val"], ctx)
	assert.Nil(t, axErr)
	jsonBytes, err := json.Marshal(eTmpl)
	assert.Nil(t, err)
	log.Println(string(jsonBytes))
	assert.False(t, strings.Contains(string(jsonBytes), "\"default\""))

	// Verify that if input parameter's default field is "", it is not omitted
	axErr = ctx.ParseFile([]byte(`
type: container
version: 1
name: empty-str-default-val
image: alpine:latest
inputs:
  parameters:
    MYPARAM:
      default: ""
`), "")
	assert.Nil(t, axErr)
	eTmpl, axErr = service.EmbedServiceTemplate(ctx.Templates["empty-str-default-val"], ctx)
	assert.Nil(t, axErr)
	jsonBytes, err = json.Marshal(eTmpl)
	assert.Nil(t, err)
	log.Println(string(jsonBytes))
	assert.True(t, strings.Contains(string(jsonBytes), "\"default\":\"\""))

}
