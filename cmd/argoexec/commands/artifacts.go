package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	//b64 "encoding/base64"
)

func init() {
	RootCmd.AddCommand(artifactsCmd)
	artifactsCmd.AddCommand(artifactsLoadCmd)
}

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Artifacts commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var artifactsLoadCmd = &cobra.Command{
	Use:   "load ARTIFACTS_JSON",
	Short: "Load artifacts according to a json specification",
	Run:   loadArtifacts,
}

// Open the Kubernetes downward api file and
// read the pod annotation file that contains template and then
// unmarshal the template
func GetTemplateFromPodAnnotations(annotationsPath string, template *wfv1.Template) error {
	// Read the annotation file
	file, err := os.Open(annotationsPath)
	if err != nil {
		fmt.Printf("ERROR opening annotation file from %s\n", annotationsPath)
		return errors.InternalWrapError(err)
	}

	defer file.Close()
	reader := bufio.NewReader(file)

	// Prefix of template property in the annotation file
	prefix := fmt.Sprintf("%s=", common.PodAnnotationsTemplatePropertyName)

	for {
		// Read line-by-line
		var buffer bytes.Buffer

		var l []byte
		var isPrefix bool
		for {
			l, isPrefix, err = reader.ReadLine()
			buffer.Write(l)

			// If we've reached the end of the line, stop reading.
			if !isPrefix {
				break
			}

			// If we're just at the EOF, break
			if err != nil {
				break
			}
		}

		// The end of the annotation file
		if err == io.EOF {
			break
		}

		line := buffer.String()

		// Read template property
		if strings.HasPrefix(line, prefix) {
			// Trim the prefix
			templateContent := strings.TrimPrefix(line, prefix)

			// This part is a bit tricky in terms of unmarshalling
			// The content in the file will be something like,
			// `"{\"type\":\"container\",\"inputs\":{},\"outputs\":{}}"`
			// which is required to unmarshal twice

			// First unmarshal to a string without escaping characters
			var templateString string
			err = json.Unmarshal([]byte(templateContent), &templateString)
			if err != nil {
				fmt.Printf("Error unmarshalling annotation into template string, %s, %v\n", templateContent, err)
				return errors.InternalWrapError(err)
			}

			// Second unmarshal to a template
			err = json.Unmarshal([]byte(templateString), template)
			if err != nil {
				fmt.Printf("Error unmarshalling annotation into template, %s, %v\n", templateString, err)
				return errors.InternalWrapError(err)
			}
			return nil
		}
	}

	if err != io.EOF {
		return errors.InternalWrapError(err)
	}

	// If reaching here, then no template prefix in the file
	return errors.Errorf(errors.CodeInternal, "No template property found from annotation file: %s", annotationsPath)
}

func LoadInputArtifacts(template *wfv1.Template) error {
	return nil
}

func loadArtifacts(cmd *cobra.Command, args []string) {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	var wfTemplate wfv1.Template

	// Read template
	err := GetTemplateFromPodAnnotations(podAnnotationsPath, &wfTemplate)
	if err != nil {
		fmt.Printf("Error getting template %v\n", err)
		os.Exit(1)
	}

	// Initialize in-cluster Kubernetes client
	config, err := rest.InClusterConfig()
	//config, err := clientcmd.BuildConfigFromFlags("", "/Users/Tianhe/.kube/cluster_minikube.conf")
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Initialize workflow executor
	wfExecutor := executor.WorkflowExecutor{
		Template:  wfTemplate,
		ClientSet: clientset,
	}

	// Download input artifacts
	err = wfExecutor.LoadArtifacts()
	if err != nil {
		fmt.Printf("Error downloading input artifacts, %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
