package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/util/cmd"
	wfclient "github.com/argoproj/argo/workflow/client"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Global variables
var (
	restConfig       *rest.Config
	clientset        *kubernetes.Clientset
	wfClient         *wfclient.WorkflowClient
	jobStatusIconMap map[string]string
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
)

func initializeSession() {
	jobStatusIconMap = map[string]string{
		wfv1.NodeStatusRunning:   ansiFormat("●", FgCyan),
		wfv1.NodeStatusSucceeded: ansiFormat("✔", FgGreen),
		wfv1.NodeStatusSkipped:   ansiFormat("○", FgWhite),
		wfv1.NodeStatusFailed:    ansiFormat("✖", FgRed),
		wfv1.NodeStatusError:     ansiFormat("⚠", FgRed),
	}
}

func initKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var kubeConfig string
	var err error
	if globalArgs.kubeConfig != "" {
		kubeConfig = globalArgs.kubeConfig
	} else {
		kubeConfig = filepath.Join(cmd.MustHomeDir(), ".kube", "config")
	}
	restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
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

func initWorkflowClient() *wfclient.WorkflowClient {
	if wfClient != nil {
		return wfClient
	}
	initKubeClient()
	var err error
	wfClient, _, err = wfclient.NewClient(restConfig)
	if err != nil {
		log.Fatal(err)
	}
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
