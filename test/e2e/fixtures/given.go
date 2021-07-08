package fixtures

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TwinProduction/go-color"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

type Given struct {
	t                 *testing.T
	client            v1alpha1.WorkflowInterface
	wfebClient        v1alpha1.WorkflowEventBindingInterface
	wfTemplateClient  v1alpha1.WorkflowTemplateInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	hydrator          hydrator.Interface
	wf                *wfv1.Workflow
	wfeb              *wfv1.WorkflowEventBinding
	wfTemplates       []*wfv1.WorkflowTemplate
	cwfTemplates      []*wfv1.ClusterWorkflowTemplate
	cronWf            *wfv1.CronWorkflow
	kubeClient        kubernetes.Interface
	bearerToken       string
}

// creates a workflow based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) Workflow(text string) *Given {
	g.t.Helper()
	g.wf = &wfv1.Workflow{}
	g.readResource(text, g.wf)
	if g.wf.Name != "" {
		g.t.Fatalf("workflow %q, but should use generate name", text)
	}
	g.checkImages(g.wf.Spec.Templates)
	return g
}

func (g *Given) readResource(text string, v metav1.Object) {
	g.t.Helper()
	var file string
	if strings.HasPrefix(text, "@") {
		file = strings.TrimPrefix(text, "@")
	} else {
		f, err := ioutil.TempFile("", "argo_e2e")
		if err != nil {
			g.t.Fatal(err)
		}
		_, err = f.Write([]byte(text))
		if err != nil {
			g.t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			g.t.Fatal(err)
		}
		file = f.Name()
	}

	{
		file, err := ioutil.ReadFile(filepath.Clean(file))
		if err != nil {
			g.t.Fatal(err)
		}
		err = yaml.Unmarshal(file, v)
		if err != nil {
			g.t.Fatal(err)
		}
		g.checkLabels(v)
	}
}

func (g *Given) checkImages(templates []wfv1.Template) {
	g.t.Helper()

	// discouraged
	discouraged := func(image string) bool {
		return image == "python:alpine3.6"
	}
	// Using an arbitrary image will result in slow and flakey tests as we can't really predict when they'll be
	// downloaded or evicted. To keep tests fast and reliable you must use allowed images.
	allowed := func(image string) bool {
		return strings.Contains(image, "argoexec:") || image == "argoproj/argosay:v1" || image == "argoproj/argosay:v2" || discouraged(image)
	}
	for _, t := range templates {
		container := t.Container
		if container != nil {
			image := container.Image
			if !allowed(image) {
				g.t.Fatalf("image not allowed in tests: %s", image)
			}
			// (⎈ |docker-desktop:argo)➜  ~ time docker run --rm argoproj/argosay:v2
			// docker run --rm argoproj/argosay˜:v2  0.21s user 0.10s system 16% cpu 1.912 total
			// docker run --rm argoproj/argosay:v1  0.17s user 0.08s system 31% cpu 0.784 total
			if discouraged(image) {
				_, _ = fmt.Println(color.Ize(color.Yellow, "DISCOURAGED IMAGE: "+g.t.Name()+" is using "+image))
			}
		}
	}
}

func (g *Given) checkLabels(m metav1.Object) {
	g.t.Helper()
	if m.GetLabels() == nil {
		m.SetLabels(map[string]string{})
	}
	if m.GetLabels()[Label] == "" {
		m.GetLabels()[Label] = "true"
	}
	if m.GetLabels()[Label] == "" {
		g.t.Fatalf("%s%s does not have %s label", m.GetName(), m.GetGenerateName(), Label)
	}
}

func (g *Given) WorkflowName(name string) *Given {
	g.t.Helper()
	g.wf = &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: name}}
	return g
}

func (g *Given) WorkflowEventBinding(text string) *Given {
	g.t.Helper()
	g.wfeb = &wfv1.WorkflowEventBinding{}
	g.readResource(text, g.wfeb)
	return g
}

func (g *Given) WorkflowTemplate(text string) *Given {
	g.t.Helper()
	wfTemplate := &wfv1.WorkflowTemplate{}
	g.readResource(text, wfTemplate)
	g.checkImages(wfTemplate.Spec.Templates)
	g.wfTemplates = append(g.wfTemplates, wfTemplate)
	return g
}

func (g *Given) CronWorkflow(text string) *Given {
	g.t.Helper()
	g.cronWf = &wfv1.CronWorkflow{}
	g.readResource(text, g.cronWf)
	g.checkImages(g.cronWf.Spec.WorkflowSpec.Templates)
	return g
}

var NoError = func(t *testing.T, output string, err error) {
	t.Helper()
	assert.NoError(t, err, output)
}

var OutputRegexp = func(rx string) func(t *testing.T, output string, err error) {
	return func(t *testing.T, output string, err error) {
		t.Helper()
		if assert.NoError(t, err, output) {
			assert.Regexp(t, rx, output)
		}
	}
}

func (g *Given) Exec(name string, args []string, block func(t *testing.T, output string, err error)) *Given {
	g.t.Helper()
	output, err := Exec(name, args...)
	block(g.t, output, err)
	return g
}

func (g *Given) RunCli(args []string, block func(t *testing.T, output string, err error)) *Given {
	return g.Exec("../../dist/argo", append([]string{"-n", Namespace}, args...), block)
}

func (g *Given) ClusterWorkflowTemplate(text string) *Given {
	g.t.Helper()
	cwfTemplate := &wfv1.ClusterWorkflowTemplate{}
	g.readResource(text, cwfTemplate)
	g.cwfTemplates = append(g.cwfTemplates, cwfTemplate)
	return g
}

func (g *Given) When() *When {
	return &When{
		t:                 g.t,
		wf:                g.wf,
		wfeb:              g.wfeb,
		wfTemplates:       g.wfTemplates,
		cwfTemplates:      g.cwfTemplates,
		cronWf:            g.cronWf,
		client:            g.client,
		wfebClient:        g.wfebClient,
		wfTemplateClient:  g.wfTemplateClient,
		cwfTemplateClient: g.cwfTemplateClient,
		cronClient:        g.cronClient,
		hydrator:          g.hydrator,
		kubeClient:        g.kubeClient,
		bearerToken:       g.bearerToken,
	}
}
