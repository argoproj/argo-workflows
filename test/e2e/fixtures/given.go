package fixtures

import (
	"io/ioutil"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Given struct {
	t                     *testing.T
	diagnostics           *Diagnostics
	client                v1alpha1.WorkflowInterface
	wfTemplateClient      v1alpha1.WorkflowTemplateInterface
	cronClient            v1alpha1.CronWorkflowInterface
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	wf                    *wfv1.Workflow
	wfTemplates           []*wfv1.WorkflowTemplate
	cronWf                *wfv1.CronWorkflow
	workflowName          string
}

// creates a workflow based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) Workflow(text string) *Given {
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
	// read the file in
	{
		file, err := ioutil.ReadFile(file)
		if err != nil {
			g.t.Fatal(err)
		}
		g.wf = &wfv1.Workflow{}
		err = yaml.Unmarshal(file, g.wf)
		if err != nil {
			g.t.Fatal(err)
		}
	}
	return g
}

func (g *Given) WorkflowName(name string) *Given {
	g.workflowName = name
	return g
}

func (g *Given) WorkflowTemplate(text string) *Given {
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
	// read the file in
	{
		file, err := ioutil.ReadFile(file)
		if err != nil {
			g.t.Fatal(err)
		}
		wfTemplate := &wfv1.WorkflowTemplate{}
		err = yaml.Unmarshal(file, wfTemplate)
		if err != nil {
			g.t.Fatal(err)
		}
		if wfTemplate.GetLabels() == nil {
			wfTemplate.SetLabels(map[string]string{})
		}
		wfTemplate.GetLabels()[label] = "true"
		g.wfTemplates = append(g.wfTemplates, wfTemplate)
	}
	return g
}

func (g *Given) CronWorkflow(text string) *Given {
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
	// read the file in
	{
		file, err := ioutil.ReadFile(file)
		if err != nil {
			g.t.Fatal(err)
		}
		g.cronWf = &wfv1.CronWorkflow{}
		err = yaml.Unmarshal(file, g.cronWf)
		if err != nil {
			g.t.Fatal(err)
		}
		if g.cronWf.GetLabels() == nil {
			g.cronWf.SetLabels(map[string]string{})
		}
		g.cronWf.GetLabels()[label] = "true"
	}
	return g
}

func (g *Given) RunCli(args []string, block func(*testing.T, string, error)) *Given {
	output, err := runCli(g.diagnostics, args)
	block(g.t, output, err)
	return g
}

func (g *Given) When() *When {
	return &When{
		t:                     g.t,
		diagnostics:           g.diagnostics,
		wf:                    g.wf,
		wfTemplates:           g.wfTemplates,
		cronWf:                g.cronWf,
		client:                g.client,
		wfTemplateClient:      g.wfTemplateClient,
		cronClient:            g.cronClient,
		offloadNodeStatusRepo: g.offloadNodeStatusRepo,
		workflowName:          g.workflowName,
	}
}
