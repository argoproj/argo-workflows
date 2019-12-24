package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/plugin/pkg/client/auth/exec"
	"k8s.io/client-go/transport"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
	apiServer "github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow"
)

// Global variables
var (
	restConfig       *rest.Config
	clientConfig     clientcmd.ClientConfig
	clientset        *kubernetes.Clientset
	wfClientset      *versioned.Clientset
	wfClient         v1alpha1.WorkflowInterface
	wftmplClient     v1alpha1.WorkflowTemplateInterface
	jobStatusIconMap map[wfv1.NodePhase]string
	noColor          bool
	namespace        string
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

func InitKubeClient() *kubernetes.Clientset {
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
	if wfClient != nil && (len(ns) == 0 || ns[0] == namespace) {
		return wfClient
	}
	InitKubeClient()
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = clientConfig.Namespace()
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
type LazyWorkflowTemplateGetter struct{}

// Get initializes it just before it's actually used and returns a retrieved workflow template.
func (c LazyWorkflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if wftmplClient == nil {
		_ = InitWorkflowClient()
	}
	return templateresolution.WrapWorkflowTemplateInterface(wftmplClient).Get(name)
}

var _ templateresolution.WorkflowTemplateNamespacedGetter = &LazyWorkflowTemplateGetter{}


func GetKubeConfigWithExecProviderToken() *workflow.ClientConfig{
	var err error
	restConfig, err = clientConfig.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	if restConfig.ExecProvider != nil {
		tc,_ := restConfig.TransportConfig()
		auth, _:= exec.GetAuthenticator(restConfig.ExecProvider)
		auth.UpdateTransportConfig(tc)
		rt,_ := transport.New(tc)
		req:=http.Request{Header: map[string][]string{}}
		rt.RoundTrip(&req)
		token := req.Header.Get("Authorization")
		restConfig.BearerToken = strings.TrimPrefix(token, "Bearer ")
	}
	var clientConfig workflow.ClientConfig
	copier.Copy(&clientConfig, restConfig)

	return &clientConfig
}

func GetApiServerGRPCClient(conn *grpc.ClientConn) (apiServer.WorkflowServiceClient, context.Context ){
	localConfig := GetKubeConfigWithExecProviderToken()
	configByte, err := json.Marshal(localConfig)
	if err != nil {
		log.Fatal(err)
	}
	configEncoded := base64.StdEncoding.EncodeToString(configByte)
	client := apiServer.NewWorkflowServiceClient(conn)
	md := metadata.Pairs("grpcgateway-authorization", configEncoded, "token", localConfig.BearerToken)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	return client, ctx
}
