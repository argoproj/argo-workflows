package fixtures

import (
	"io/ioutil"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Given struct {
	t          *testing.T
	client     v1alpha1.WorkflowInterface
	cronClient v1alpha1.CronWorkflowInterface
	wf         *wfv1.Workflow
	cronWf     *wfv1.CronWorkflow
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
		if g.wf.GetLabels() == nil {
			g.wf.SetLabels(map[string]string{})
		}
		g.wf.GetLabels()[label] = "true"
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

func (g *Given) When() *When {
	return &When{
		t:          g.t,
		wf:         g.wf,
		cronWf:     g.cronWf,
		client:     g.client,
		cronClient: g.cronClient,
	}
}
