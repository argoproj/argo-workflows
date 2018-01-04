package commands

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Global variables
var (
	restConfig       *rest.Config
	clientConfig     clientcmd.ClientConfig
	clientset        *kubernetes.Clientset
	wfClient         v1alpha1.WorkflowInterface
	jobStatusIconMap map[wfv1.NodePhase]string
)

func init() {
	cobra.OnInitialize(initializeSession)
}

// ANSI escape codes
const (
	escape    = "\x1b"
	noFormat  = 0
	Bold      = 1
	FgBlack   = 30
	FgRed     = 31
	FgGreen   = 32
	FgYellow  = 33
	FgBlue    = 34
	FgMagenta = 35
	FgCyan    = 36
	FgWhite   = 37
	FgDefault = 39
)

func initializeSession() {
	jobStatusIconMap = map[wfv1.NodePhase]string{
		//Pending:   ansiFormat("◷", FgDefault),
		wfv1.NodeRunning:   ansiFormat("●", FgCyan),
		wfv1.NodeSucceeded: ansiFormat("✔", FgGreen),
		wfv1.NodeSkipped:   ansiFormat("○", FgDefault),
		wfv1.NodeFailed:    ansiFormat("✖", FgRed),
		wfv1.NodeError:     ansiFormat("⚠", FgRed),
	}
}

func initKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var err error
	restConfig, err = clientConfig.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	return clientset
}

// InitWorkflowClient creates a new client for the Kubernetes Workflow CRD.
func InitWorkflowClient(ns ...string) v1alpha1.WorkflowInterface {
	if wfClient != nil {
		return wfClient
	}
	initKubeClient()
	var namespace string
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfcs := wfclientset.NewForConfigOrDie(restConfig)
	wfClient = wfcs.ArgoprojV1alpha1().Workflows(namespace)
	return wfClient
}

// ansiFormat wraps ANSI escape codes to a string to format the string to a desired color.
// NOTE: we still apply formatting even if there is no color formatting desired.
// The purpose of doing this is because when we apply ANSI color escape sequences to our
// output, this confuses the tabwriter library which miscalculates widths of columns and
// misaligns columns. By always applying a ANSI escape sequence (even when we don't want
// color, it provides more consistent string lengths so that tabwriter can calculate
// widths correctly.
func ansiFormat(s string, codes ...int) string {
	if globalArgs.noColor || os.Getenv("TERM") == "dumb" || len(codes) == 0 {
		return s
	}
	codeStrs := make([]string, len(codes))
	for i, code := range codes {
		codeStrs[i] = strconv.Itoa(code)
	}
	sequence := strings.Join(codeStrs, ";")
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, sequence, s, escape, noFormat)
}

var yamlSeparator = regexp.MustCompile("\\n---")

// splitYAMLFile is a helper to split a body into multiple workflow objects
func splitYAMLFile(body []byte) ([]wfv1.Workflow, error) {
	manifestsStrings := yamlSeparator.Split(string(body), -1)
	manifests := make([]wfv1.Workflow, 0)
	for _, manifestStr := range manifestsStrings {
		if strings.TrimSpace(manifestStr) == "" {
			continue
		}
		var wf wfv1.Workflow
		err := yaml.Unmarshal([]byte(manifestStr), &wf)
		if wf.Kind != "" && wf.Kind != wfv1.CRDKind {
			// If we get here, it was a k8s manifest which was not of type 'Workflow'
			// We ignore these since we only care about validating Workflow manifests.
			continue
		}
		if err != nil {
			return nil, errors.New(errors.CodeBadRequest, err.Error())
		}
		manifests = append(manifests, wf)
	}
	return manifests, nil
}
