package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
)

// Global variables
var (
	// DEPRECATED
	restConfig *rest.Config
	// DEPRECATED
	clientset *kubernetes.Clientset
	// DEPRECATED
	wfClientset *versioned.Clientset
	// DEPRECATED
	wfClient v1alpha1.WorkflowInterface
	// DEPRECATED
	wftmplClient     v1alpha1.WorkflowTemplateInterface
	jobStatusIconMap map[wfv1.NodePhase]string
	noColor          bool
	// DEPRECATED
	namespace string
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

// Default status for printWorkflow
const DefaultStatus = ""

func initializeSession() {
	jobStatusIconMap = map[wfv1.NodePhase]string{
		wfv1.NodePending:   ansiFormat("◷", FgYellow),
		wfv1.NodeRunning:   ansiFormat("●", FgCyan),
		wfv1.NodeSucceeded: ansiFormat("✔", FgGreen),
		wfv1.NodeSkipped:   ansiFormat("○", FgDefault),
		wfv1.NodeFailed:    ansiFormat("✖", FgRed),
		wfv1.NodeError:     ansiFormat("⚠", FgRed),
	}
}

// DEPRECATED
func InitKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var err error
	restConfig, err = client.Config.ClientConfig()
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
// DEPRECATED
func InitWorkflowClient(ns ...string) v1alpha1.WorkflowInterface {
	if wfClient != nil && (len(ns) == 0 || ns[0] == namespace) {
		return wfClient
	}
	InitKubeClient()
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = client.Config.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfClientset = versioned.NewForConfigOrDie(restConfig)
	wfClient = wfClientset.ArgoprojV1alpha1().Workflows(namespace)
	wftmplClient = wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace)
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
	if noColor || os.Getenv("TERM") == "dumb" || len(codes) == 0 {
		return s
	}
	codeStrs := make([]string, len(codes))
	for i, code := range codes {
		codeStrs[i] = strconv.Itoa(code)
	}
	sequence := strings.Join(codeStrs, ";")
	return fmt.Sprintf("%s[%sm%s%s[%dm", escape, sequence, s, escape, noFormat)
}

// LazyWorkflowTemplateGetter is a wrapper of v1alpha1.WorkflowTemplateInterface which
// supports lazy initialization.
// DEPRECATED
type LazyWorkflowTemplateGetter struct{}

// Get initializes it just before it's actually used and returns a retrieved workflow template.
// DEPRECATED
func (c LazyWorkflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if wftmplClient == nil {
		_ = InitWorkflowClient()
	}
	return templateresolution.WrapWorkflowTemplateInterface(wftmplClient).Get(name)
}

// DEPRECATED
var _ templateresolution.WorkflowTemplateNamespacedGetter = &LazyWorkflowTemplateGetter{}

// DEPRECATED
func GetWFApiServerGRPCClient(conn *grpc.ClientConn) (workflow.WorkflowServiceClient, context.Context) {
	return workflow.NewWorkflowServiceClient(conn), client.GetContext()
}
