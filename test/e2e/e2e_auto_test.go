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

//var kubeConfig = flag.String("kubeconfig", "", "Path to Kubernetes config file")
// TestSuspendResume tests the suspend and resume feature

func TestRunWorkflowAuto(t *testing.T) {

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
		//submittedWf := commands.SubmitWorkflows([]string{wf.Path}, nil, commands.NewCliSubmitOpts("",true,false,false,nil) )
		//log.Printf("%s submitted successfully", submittedWf)
		//submittedWfs = append(submittedWfs, submittedWf...)

	}
	log.Printf("Workflow List: %v", workflowPath)

	submittedWfs := commands.SubmitWorkflows(workflowPath, nil, commands.NewCliSubmitOpts("", true, false, false, nil))

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
				assert.True(t, result == e2eWf.ExpectedStatus)
			}()
		}
	}
	waitgroup.Wait()
	//cmd := exec.Command ("kubectl", "get", "wf")
	//var out bytes.Buffer
	//cmd.Stdout = &out
	//
	//err := cmd.Run()
	//if err != nil {
	//	log.Println(err)
	//}
	//var waitgroup sync.WaitGroup
	//
	//sc := bufio.NewScanner(strings.NewReader(out.String()))
	//for sc.Scan() {
	//	line := strings.Split(sc.Text()," ")
	//	if line[0] == "NAME" {
	//		continue
	//	}
	//
	//	name := line[0][:strings.LastIndex(line[0], "-")]
	//	fmt.Println(name)
	//
	//	e2eWf := statusMap[name]
	//
	//	if e2eWf != nil {
	//		waitgroup.Add(1)
	//		go func() {
	//			status := getStatus(line[0], t, e2eWf.ExpectedStatus, e2eWf.Timeout)
	//			assert.True(t, status == e2eWf.ExpectedStatus)
	//			fmt.Printf("%s workflow completed. status=%s", line[0], status)
	//			waitgroup.Done()
	//		}()
	//	}
	//}
	//waitgroup.Wait()
}

func getStatus(t *testing.T, wfName string, e2eWf *E2EWorkflow) v1alpha1.NodePhase {
	wfClient := commands.InitWorkflowClient()
	defer CleanUpWorkflow(wfName)
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
		log.Printf("%s is still in  %s", wfName, result)
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
