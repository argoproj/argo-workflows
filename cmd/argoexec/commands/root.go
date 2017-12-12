package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/argoproj/argo"
	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	// Global CLI flags
	GlobalArgs globalFlags
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argoexec is the executor sidecar to workflow containers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

type globalFlags struct {
	hostIP             string // --host-ip
	podAnnotationsPath string // --pod-annotations
}

func init() {
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.hostIP, "host-ip", common.EnvVarHostIP, fmt.Sprintf("IP of host. (Default: %s)", common.EnvVarHostIP))
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.podAnnotationsPath, "pod-annotations", common.PodMetadataAnnotationsPath, fmt.Sprintf("Pod annotations fiel from k8s downward API. (Default: %s)", common.PodMetadataAnnotationsPath))
}

func initExecutor() *executor.WorkflowExecutor {
	podAnnotationsPath := common.PodMetadataAnnotationsPath

	// Use the path specified from the flag
	if GlobalArgs.podAnnotationsPath != "" {
		podAnnotationsPath = GlobalArgs.podAnnotationsPath
	}

	var wfTemplate wfv1.Template

	// Read template
	err := getTemplateFromPodAnnotations(podAnnotationsPath, &wfTemplate)
	if err != nil {
		log.Fatalf("Error getting template %v", err)
	}

	// Initialize in-cluster Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		log.Fatalf("Unable to determine pod name from environment variable %s", common.EnvVarPodName)
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		log.Fatalf("Unable to determine pod namespace from environment variable %s", common.EnvVarNamespace)
	}

	// Initialize workflow executor
	wfExecutor := executor.WorkflowExecutor{
		PodName:   podName,
		Template:  wfTemplate,
		ClientSet: clientset,
		Namespace: namespace,
	}
	yamlBytes, _ := yaml.Marshal(&wfExecutor.Template)
	log.Infof("Executor (version: %s) initialized with template:\n%s", argo.FullVersion, string(yamlBytes))
	return &wfExecutor
}

// Open the Kubernetes downward api file and
// read the pod annotation file that contains template and then
// unmarshal the template
func getTemplateFromPodAnnotations(annotationsPath string, template *wfv1.Template) error {
	// Read the annotation file
	file, err := os.Open(annotationsPath)
	if err != nil {
		fmt.Printf("ERROR opening annotation file from %s\n", annotationsPath)
		return errors.InternalWrapError(err)
	}

	defer func() {
		_ = file.Close()
	}()
	reader := bufio.NewReader(file)

	// Prefix of template property in the annotation file
	prefix := fmt.Sprintf("%s=", common.AnnotationKeyTemplate)

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
	return errors.InternalErrorf("No template property found from annotation file: %s", annotationsPath)
}
