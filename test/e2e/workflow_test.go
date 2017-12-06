package e2e

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/argoproj/argo/cmd/argo/commands"
)

func TestRunWorkflowBasic(t *testing.T) {
	workflowName := "my-test"
	workflowYaml := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    name: ` + workflowName + `
spec:
    entrypoint: run-workflow
    templates:
    - name: run-workflow
      container:
        image: alpine:3.6
        command: [date]
`

	content := []byte(workflowYaml)
	tmpfile, err := ioutil.TempFile("", "argo_test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	commands.SubmitWorkflows(nil, []string{tmpfile.Name()})

	wfClient := commands.InitWorkflowClient()

	for {
		wf, err := wfClient.GetWorkflow(workflowName)
		if err != nil {
			log.Fatal(err)
		}
		var allCompleted bool
		allCompleted = true
		for k, v := range wf.Status.Nodes {
			if !v.Completed() {
				fmt.Printf("Status of %s: %v\n", k, v.Phase)
				allCompleted = false
			}
		}

		if allCompleted {
			fmt.Printf("Workflow %s completed successfully", workflowName)
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}
