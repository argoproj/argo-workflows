package fixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TwiN/go-color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	wftsClient        v1alpha1.WorkflowTaskSetInterface
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
	restConfig        *rest.Config
}

// creates a workflow based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) Workflow(text string) *Given {
	g.t.Helper()
	g.wf = &wfv1.Workflow{}
	g.readResource(text, g.wf)
	g.checkImages(g.wf)
	return g
}

func (g *Given) readResource(text string, v metav1.Object) {
	g.t.Helper()
	var file string
	if strings.HasPrefix(text, "@") {
		file = strings.TrimPrefix(text, "@")
	} else {
		f, err := os.CreateTemp("", "argo_e2e")
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
		file, err := os.ReadFile(filepath.Clean(file))
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

func (g *Given) checkImages(wf interface{}) {
	g.t.Helper()
	var defaultImage string
	var templates []wfv1.Template
	switch baseTemplate := wf.(type) {
	case *wfv1.Workflow:
		if baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.TemplateDefaults.Container.Image
			templates = baseTemplate.Spec.Templates
		}
	case *wfv1.WorkflowTemplate:
		if baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.TemplateDefaults.Container.Image
			templates = baseTemplate.Spec.Templates
		}
	case *wfv1.CronWorkflow:
		if baseTemplate.Spec.WorkflowSpec.TemplateDefaults != nil && baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container != nil && baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container.Image
			templates = baseTemplate.Spec.WorkflowSpec.Templates
		}
	default:
		g.t.Fatalf("Unsupported checkImage workflow type: %s", wf)
	}

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
			var image string
			if container.Image != "" {
				image = container.Image
			} else {
				image = defaultImage
			}
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
	g.checkImages(wfTemplate)
	g.wfTemplates = append(g.wfTemplates, wfTemplate)
	return g
}

func (g *Given) CronWorkflow(text string) *Given {
	g.t.Helper()
	g.cronWf = &wfv1.CronWorkflow{}
	g.readResource(text, g.cronWf)
	g.checkImages(g.cronWf)
	return g
}

var NoError = func(t *testing.T, output string, err error) {
	t.Helper()
	require.NoError(t, err, output)
}

var OutputRegexp = func(rx string) func(t *testing.T, output string, err error) {
	return func(t *testing.T, output string, err error) {
		t.Helper()
		require.NoError(t, err, output)
		assert.Regexp(t, rx, output)
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
		wftsClient:        g.wftsClient,
		cwfTemplateClient: g.cwfTemplateClient,
		cronClient:        g.cronClient,
		hydrator:          g.hydrator,
		kubeClient:        g.kubeClient,
		bearerToken:       g.bearerToken,
		restConfig:        g.restConfig,
	}
}
