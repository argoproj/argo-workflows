package archive

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/packer"
)

type cliSubmitOpts struct {
	priority *int32 // --priority
	output   string // --output
	wait     bool   // --wait
	watch    bool   // --watch
	log      bool   // --log
}

func waitWatchOrLog(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, cliSubmitOpts cliSubmitOpts) {
	if cliSubmitOpts.log {
		for _, workflow := range workflowNames {
			logWorkflow(ctx, serviceClient, namespace, workflow, "", "", "", &corev1.PodLogOptions{
				Container: common.MainContainerName,
				Follow:    true,
				Previous:  false,
			})
		}
	}
	if cliSubmitOpts.wait {
		waitWorkflows(ctx, serviceClient, namespace, workflowNames, false, !(cliSubmitOpts.output == "" || cliSubmitOpts.output == "wide"))
	} else if cliSubmitOpts.watch {
		for _, workflow := range workflowNames {
			watchWorkflow(ctx, serviceClient, namespace, workflow, cliSubmitOpts.output)
		}
	}
}

func logWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace, workflow, podName, grep, selector string, logOptions *corev1.PodLogOptions) {
	// logs
	stream, err := serviceClient.WorkflowLogs(ctx, &workflowpkg.WorkflowLogRequest{
		Name:       workflow,
		Namespace:  namespace,
		PodName:    podName,
		LogOptions: logOptions,
		Selector:   selector,
		Grep:       grep,
	})
	errors.CheckError(err)

	// loop on log lines
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return
		}
		errors.CheckError(err)
		fmt.Println(ansiFormat(fmt.Sprintf("%s: %s", event.PodName, event.Content), ansiColorCode(event.PodName)))
	}
}

var noColor bool

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

func ansiColorCode(s string) int {
	i := 0
	for _, c := range s {
		i += int(c)
	}
	colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite}
	return colors[i%len(colors)]
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

// waitWorkflows waits for the given workflowNames.
func waitWorkflows(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflowNames []string, ignoreNotFound, quiet bool) {
	var wg sync.WaitGroup
	wfSuccessStatus := true

	for _, name := range workflowNames {
		wg.Add(1)
		go func(name string) {
			if !waitOnOne(serviceClient, ctx, name, namespace, ignoreNotFound, quiet) {
				wfSuccessStatus = false
			}
			wg.Done()
		}(name)

	}
	wg.Wait()

	if !wfSuccessStatus {
		os.Exit(1)
	}
}

func waitOnOne(serviceClient workflowpkg.WorkflowServiceClient, ctx context.Context, wfName, namespace string, ignoreNotFound, quiet bool) bool {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(wfName),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound && ignoreNotFound {
			return true
		}
		errors.CheckError(err)
		return false
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Debug("Re-establishing workflow watch")
			stream, err = serviceClient.WatchWorkflows(ctx, req)
			errors.CheckError(err)
			continue
		}
		errors.CheckError(err)
		if event == nil {
			continue
		}
		wf := event.Object
		if !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				fmt.Printf("%s %s at %v\n", wfName, wf.Status.Phase, wf.Status.FinishedAt)
			}
			if wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError {
				return false
			}
			return true
		}
	}
}

func watchWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace string, workflow string, output string) {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(workflow),
			ResourceVersion: "0",
		},
	}
	stream, err := serviceClient.WatchWorkflows(ctx, req)
	errors.CheckError(err)

	wfChan := make(chan *wfv1.Workflow)
	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Debug("Re-establishing workflow watch")
				stream, err = serviceClient.WatchWorkflows(ctx, req)
				errors.CheckError(err)
				continue
			}
			errors.CheckError(err)
			if event == nil {
				continue
			}
			wfChan <- event.Object
		}
	}()

	var wf *wfv1.Workflow
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case newWf := <-wfChan:
			// If we get a new event, update our workflow
			if newWf == nil {
				return
			}
			wf = newWf
		case <-ticker.C:
			// If we don't, refresh the workflow screen every second
		case <-ctx.Done():
			// When the context gets canceled
			return
		}

		printWorkflowStatus(wf, output)
		if wf != nil && !wf.Status.FinishedAt.IsZero() {
			return
		}
	}
}

func printWorkflowStatus(wf *wfv1.Workflow, output string) {
	if wf == nil {
		return
	}
	err := packer.DecompressWorkflow(wf)
	errors.CheckError(err)
	print("\033[H\033[2J")
	print("\033[0;0H")
	printWorkflow(wf, output)
}
