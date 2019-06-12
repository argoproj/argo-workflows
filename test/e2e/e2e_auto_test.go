// +build e2e

package e2e

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type E2ETestProp struct {
	Workflows []E2EWorkflow `yaml:"workflows"`
}
type E2EWorkflow struct {
	Name           string             `yaml:"name"`
	Path           string             `yaml:"path"`
	ExpectedStatus v1alpha1.NodePhase `yaml:"expectedStatus"`
	Timeout        time.Duration      `yaml:"timeout"`
}

func TestE2EWorkflow(t *testing.T) {

	fmt.Println("Starting End-to-End Testing")
	testRunWorkflows(t)

}

func getConf() *E2ETestProp {
	e2eProp := E2ETestProp{}
	yamlFile, err := ioutil.ReadFile("./e2e_test_prop.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println(string(yamlFile))
	err = yaml.Unmarshal(yamlFile, &e2eProp)

	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return &e2eProp
}

func testRunWorkflows(t *testing.T) {
	statusMap := make(map[string]*E2EWorkflow)
	property := getConf()
	commands.NewCommand()

	var workflowPath []string
	for i := range property.Workflows {
		wf := property.Workflows[i]
		statusMap[wf.Name] = &wf
		workflowPath = append(workflowPath, wf.Path)
	}
	log.Printf("Workflow List: %v", workflowPath)

	submittedWfs := commands.SubmitWorkflows(workflowPath, nil, commands.NewCliSubmitOpts("", false, false, false, nil))

	var waitgroup sync.WaitGroup
	for i := range submittedWfs {
		name := submittedWfs[i]
		wfname := name[:strings.LastIndex(name, "-")]
		e2eWf := statusMap[wfname]
		if e2eWf != nil {
			if e2eWf.Timeout == 0 {
				e2eWf.Timeout = 1800
			}
			waitgroup.Add(1)
			go func() {
				defer waitgroup.Done()
				result := getStatus(t, name, e2eWf)
				fmt.Println(name, result, e2eWf.ExpectedStatus)
				assert.True(t, result == e2eWf.ExpectedStatus, "workflow execution is failed.", wfname)
			}()
		}
	}
	waitgroup.Wait()
}

func getStatus(t *testing.T, wfName string, e2eWf *E2EWorkflow) v1alpha1.NodePhase {
	wfClient := commands.InitWorkflowClient()
	//defer CleanUpWorkflow(wfName)
	log.Printf("Start checking status : %s", wfName)
	for start := time.Now(); ; {
		//for{
		wf, _ := wfClient.Get(wfName, v1.GetOptions{})
		result := wf.Status.Phase

		if result == "Succeeded" || result == "Failed" || result == "Error" || result == e2eWf.ExpectedStatus {
			return result
		}

		if time.Since(start) > e2eWf.Timeout*time.Second {
			log.Printf("Workflow execution timed out. %s ", wfName)
			assert.True(t, false)
			return ""
		}
		time.Sleep(1 * time.Minute)
		//log.Printf("%s is still in  %s", wfName, result)
	}
}

func CleanUpWorkflow(wfName string) {
	//log.Println("Cleaning up workflows")
	wfClient := commands.InitWorkflowClient()

	err := wfClient.Delete(wfName, nil)
	if err != nil {
		log.Println(err)
	}

	//cmd1 := exec.Command("kubectl", "delete", "wf", "--all")
	//var stderr bytes.Buffer
	//cmd1.Stderr = &stderr
	//var out1 bytes.Buffer
	//cmd1.Stdout = &out1
	//err := cmd1.Run()

}
