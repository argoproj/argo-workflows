package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(submitCmd)
	submitCmd.Flags().StringVar(&submitArgs.entrypoint, "entrypoint", "", "override entrypoint")
	submitCmd.Flags().StringSliceVarP(&submitArgs.parameters, "parameter", "p", []string{}, "pass an input parameter")
}

type submitFlags struct {
	entrypoint string   // --entrypoint
	parameters []string // --parameter
}

var submitArgs submitFlags

var submitCmd = &cobra.Command{
	Use:   "submit FILE1 FILE2...",
	Short: "submit a workflow",
	Run:   SubmitWorkflows,
}

var yamlSeparator = regexp.MustCompile("\\n---")

// SubmitWorkflows runs the given workflow
func SubmitWorkflows(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	InitWorkflowClient()
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
			if submitArgs.entrypoint != "" {
				wf.Spec.Entrypoint = submitArgs.entrypoint
			}
			if len(submitArgs.parameters) > 0 {
				newParams := make([]wfv1.Parameter, 0)
				passedParams := make(map[string]bool)
				for _, paramStr := range submitArgs.parameters {
					parts := strings.SplitN(paramStr, "=", 2)
					if len(parts) == 1 {
						log.Fatalf("Expected parameter of the form: NAME=VALUE. Recieved: %s", paramStr)
					}
					param := wfv1.Parameter{
						Name:  parts[0],
						Value: &parts[1],
					}
					newParams = append(newParams, param)
					passedParams[param.Name] = true
				}
				for _, param := range wf.Spec.Arguments.Parameters {
					if _, ok := passedParams[param.Name]; ok {
						// this parameter was overridden via command line
						continue
					}
					newParams = append(newParams, param)
				}
				wf.Spec.Arguments.Parameters = newParams
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
