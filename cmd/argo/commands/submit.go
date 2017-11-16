package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

var (
	yamlSeparator = regexp.MustCompile("\\n---")
)

func init() {
	RootCmd.AddCommand(submitCmd)
}

var submitCmd = &cobra.Command{
	Use:   "submit FILE1 FILE2...",
	Short: "submit a workflow",
	Run:   submitWorkflows,
}

func submitWorkflows(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	initWorkflowClient()
	for _, filePath := range args {
		var body []byte
		var err error
		if cmdutil.IsURL(filePath) {
			response, err := http.Get(filePath)
			if err != nil {
				log.Fatal(err)
			}
			body, err = ioutil.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			body, err = ioutil.ReadFile(filePath)
			if err != nil {
				log.Fatal(err)
			}
		}
		manifests := yamlSeparator.Split(string(body), -1)
		for _, manifestStr := range manifests {
			if strings.TrimSpace(manifestStr) == "" {
				continue
			}
			var wf wfv1.Workflow
			err := yaml.Unmarshal([]byte(manifestStr), &wf)
			if err != nil {
				log.Fatalf("Workflow manifest %s failed to parse: %v\n%s", filePath, err, manifestStr)
			}
			err = common.ValidateWorkflow(&wf)
			if err != nil {
				log.Fatalf("Workflow manifest %s failed validation: %v", filePath, err)
			}
			created, err := wfClient.CreateWorkflow(&wf)
			if err != nil {
				log.Fatal(err)
			}
			printWorkflow(created)
		}
	}
}
