package fixtures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/config"
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
	config            *config.Config
}

// creates a workflow based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) Workflow(text string) *Given {
	g.t.Helper()
	g.wf = &wfv1.Workflow{}
	g.readResource(text, g.wf)
	g.checkImages(g.wf, false)
	return g
}

// Load parsed Workflow that's assumed to be from the "examples/" directory
func (g *Given) ExampleWorkflow(wf *wfv1.Workflow) *Given {
	g.t.Helper()
	g.wf = wf
	g.checkLabels(wf)
	g.checkImages(g.wf, true)
	return g
}

// Load created workflow
func (g *Given) WorkflowWorkflow(wf *wfv1.Workflow) *Given {
	g.t.Helper()
	g.wf = wf
	g.checkImages(g.wf, false)
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

// Check if given Workflow, WorkflowTemplate, or CronWorkflow uses forbidden images.
// Using an arbitrary image will result in slow and flakey tests as we can't really predict when they'll be
// downloaded or evicted. To keep tests fast and reliable you must use allowed images.
// Workflows from the examples/ folder are given special treatment and allowed to use a wider range of images.
func (g *Given) checkImages(wf interface{}, isExample bool) {
	g.t.Helper()
	var defaultImage string
	var templates []wfv1.Template
	switch baseTemplate := wf.(type) {
	case *wfv1.Workflow:
		templates = baseTemplate.Spec.Templates
		if baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.TemplateDefaults.Container.Image
		}
	case *wfv1.WorkflowTemplate:
		templates = baseTemplate.Spec.Templates
		if baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.TemplateDefaults.Container.Image
		}
	case *wfv1.CronWorkflow:
		templates = baseTemplate.Spec.WorkflowSpec.Templates
		if baseTemplate.Spec.WorkflowSpec.TemplateDefaults != nil && baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container != nil && baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container.Image != "" {
			defaultImage = baseTemplate.Spec.WorkflowSpec.TemplateDefaults.Container.Image
		}
	default:
		g.t.Fatalf("Unsupported checkImage workflow type: %s", wf)
	}

	allowed := func(image string) bool {
		return strings.Contains(image, "argoexec:") ||
			image == "argoproj/argosay:v1" ||
			image == "argoproj/argosay:v2" ||
			image == "quay.io/argoproj/argocli:latest" ||
			(isExample && (image == "busybox" || image == "python:alpine3.6"))
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
	g.checkImages(wfTemplate, false)
	g.wfTemplates = append(g.wfTemplates, wfTemplate)
	return g
}

func (g *Given) CronWorkflow(text string) *Given {
	g.t.Helper()
	g.cronWf = &wfv1.CronWorkflow{}
	g.readResource(text, g.cronWf)
	g.checkImages(g.cronWf, false)
	return g
}

var NoError = func(t *testing.T, output string, err error) {
	t.Helper()
	require.NoError(t, err, output)
}

var ErrorOutput = func(contains string) func(t *testing.T, output string, err error) {
	return func(t *testing.T, output string, err error) {
		t.Helper()
		require.Error(t, err)
		assert.Contains(t, output, contains)
	}
}

var OutputRegexp = func(rx string) func(t *testing.T, output string, err error) {
	return func(t *testing.T, output string, err error) {
		t.Helper()
		require.NoError(t, err, output)
		assert.Regexp(t, rx, output)
	}
}

func (g *Given) Exec(name string, args []string, stdin string, block func(t *testing.T, output string, err error)) *Given {
	g.t.Helper()
	output, err := Exec(name, stdin, args...)
	block(g.t, output, err)
	return g
}

// Use Kubectl to server-side apply the given file
func (g *Given) KubectlApply(file string, block func(t *testing.T, output string, err error)) *Given {
	g.t.Helper()
	return g.Exec("kubectl", append([]string{"-n", Namespace, "apply", "--server-side", "-f"}, file), "", block)
}

func (g *Given) RunCli(args []string, block func(t *testing.T, output string, err error)) *Given {
	return g.RunCliStdin(args, "", block)
}

func (g *Given) RunCliStdin(args []string, stdin string, block func(t *testing.T, output string, err error)) *Given {
	return g.Exec("../../dist/argo", append([]string{"-n", Namespace}, args...), stdin, block)
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
		config:            g.config,
	}
}
