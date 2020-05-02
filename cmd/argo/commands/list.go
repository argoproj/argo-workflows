package commands

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	argoerrors "github.com/argoproj/pkg/errors"
	argotime "github.com/argoproj/pkg/time"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/printer"
	"github.com/argoproj/argo/workflow/common"
)

type listFlags struct {
	allNamespaces bool     // --all-namespaces
	status        []string // --status
	completed     bool     // --completed
	running       bool     // --running
	prefix        string   // --prefix
	output        string   // --output
	since         string   // --since
	chunkSize     int64    // --chunk-size
	noHeaders     bool     // --no-headers
	continueToken string   // --continue
	limit         int64    // --limit
}

type cursor struct {
	KubeCursor       string `json:"kube_cursor,omitempty"`
	LastWorkflowName string `json:"last_workflow_name,omitempty"`
	Prefix           string `json:"prefix,omitempty"`
	Since            string `json:"since,omitempty"`
}

func NewListCommand() *cobra.Command {
	var (
		listArgs listFlags
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "list workflows",
		Run: func(cmd *cobra.Command, args []string) {
			listWorkflows(&listArgs)
		},
	}
	command.Flags().BoolVar(&listArgs.allNamespaces, "all-namespaces", false, "Show workflows from all namespaces")
	command.Flags().StringVar(&listArgs.prefix, "prefix", "", "Filter workflows by prefix")
	command.Flags().StringSliceVar(&listArgs.status, "status", []string{}, "Filter by status (comma separated)")
	command.Flags().BoolVar(&listArgs.completed, "completed", false, "Show only completed workflows")
	command.Flags().BoolVar(&listArgs.running, "running", false, "Show only running workflows")
	command.Flags().StringVarP(&listArgs.output, "output", "o", "", "Output format. One of: wide|name")
	command.Flags().StringVar(&listArgs.since, "since", "", "Show only workflows newer than a relative duration")
	command.Flags().Int64VarP(&listArgs.chunkSize, "chunk-size", "", 0, "Return large lists in chunks rather than all at once. Pass 0 to disable.")
	command.Flags().BoolVar(&listArgs.noHeaders, "no-headers", false, "Don't print headers (default print headers).")
	command.Flags().StringVar(&listArgs.continueToken, "continue", "", "Return the next batch of workloads starting from this token. Note that the chunk size used to fetch this token must be passed in at the same time.")
	command.Flags().Int64VarP(&listArgs.limit, "limit", "", 500, "Return a list with maximum N workflows. Pass 0 to retrieve the full list.")
	return command
}

func listWorkflows(listArgs *listFlags) {
	kubeCursor, lastWorkflowName, err := getKubeCursor(listArgs)
	if err != nil {
		log.Error(err)
		return
	}
	listOpts := getListOpts(listArgs)

	ctx, apiClient := client.NewAPIClient()
	serviceClient := apiClient.NewWorkflowServiceClient()
	namespace := client.Namespace()
	if listArgs.allNamespaces {
		namespace = ""
	}

	initialFetch := true
	wfName := ""

	// Keep fetching workflows until we've got enough amount
	var workflows wfv1.Workflows
	for initialFetch || kubeCursor != "" {
		listOpts.Continue = kubeCursor
		tmpWfList, err := serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
			Namespace:   namespace,
			ListOptions: &listOpts,
		})
		argoerrors.CheckError(err)

		if initialFetch {
			findTargetWorkflow(tmpWfList, lastWorkflowName)
			initialFetch = false
		}
		filterWorkflow(tmpWfList, listArgs)

		if listArgs.limit != 0 && int64(len(workflows)+len(tmpWfList.Items)) > listArgs.limit {
			if int64(len(workflows)) == listArgs.limit {
				// No need to intake more workflows
				// Will continue from the last workflow to be returned
				wfName = workflows[listArgs.limit-1].Name
			} else {
				wfName = truncateWorkflowList(tmpWfList, &workflows, listArgs)
				workflows = append(workflows, tmpWfList.Items...)
			}
			break
		}

		workflows = append(workflows, tmpWfList.Items...)
		kubeCursor = tmpWfList.ListMeta.Continue
	}

	encodedCursor := ""
	if wfName != "" {
		encodedCursor, err = encodeCursor(kubeCursor, wfName, listArgs)
		if err != nil {
			log.Errorf("Error when preparing the cursor for other workflows: %v", err)
		}
	}

	err = printer.PrintWorkflows(workflows, os.Stdout, printer.PrintOpts{
		NoHeaders: listArgs.noHeaders,
		Namespace: listArgs.allNamespaces,
		Output:    listArgs.output,
	})
	argoerrors.CheckError(err)

	if encodedCursor != "" {
		switch listArgs.output {
		case "", "wide", "name":
			fmt.Printf("There are additional suppressed results, show them by passing in `--continue %s`\n", encodedCursor)
		default:
		}
	}
}

func getListOpts(listArgs *listFlags) metav1.ListOptions {
	listOpts := metav1.ListOptions{
		Limit: listArgs.chunkSize,
	}
	labelSelector := labels.NewSelector()
	if len(listArgs.status) != 0 {
		req, _ := labels.NewRequirement(common.LabelKeyPhase, selection.In, listArgs.status)
		if req != nil {
			labelSelector = labelSelector.Add(*req)
		}
	}
	if listArgs.completed {
		req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"true"})
		labelSelector = labelSelector.Add(*req)
	}
	if listArgs.running {
		req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
		labelSelector = labelSelector.Add(*req)
	}
	listOpts.LabelSelector = labelSelector.String()

	return listOpts
}

func getKubeCursor(listArgs *listFlags) (string, string, error) {
	if listArgs.continueToken != "" {
		jsonString, err := base64.RawURLEncoding.DecodeString(listArgs.continueToken)
		if err != nil {
			return "", "", errors.New("Invalid continue token: malformed value")
		}
		var data cursor
		err = json.Unmarshal([]byte(jsonString), &data)
		if err != nil || data.LastWorkflowName == "" && data.KubeCursor != "" {
			return "", "", errors.New("Invalid continue token: malformed value")
		}
		if data.LastWorkflowName != "" && (data.Prefix != listArgs.prefix || data.Since != listArgs.since) {
			return "", "", errors.New("Invalid continue token: please ensure that you are using the identical values for `prefix` and `since` with which this token was acquired")
		}
		return data.KubeCursor, data.LastWorkflowName, nil
	}
	return "", "", nil
}

func encodeCursor(kubeCursor string, lastWorkflowName string, listArgs *listFlags) (string, error) {
	jsonCursor, err := json.Marshal(cursor{
		KubeCursor:       kubeCursor,
		LastWorkflowName: lastWorkflowName,
		Prefix:           listArgs.prefix,
		Since:            listArgs.since,
	})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(jsonCursor), nil
}

func findTargetWorkflow(wfList *wfv1.WorkflowList, targetWfName string) {
	if targetWfName == "" {
		return
	}
	idx := -1
	for i, wf := range wfList.Items {
		if wf.Name == targetWfName {
			idx = i
			break
		}
	}
	wfList.Items = wfList.Items[idx+1:]
}

func filterWorkflow(wfList *wfv1.WorkflowList, listArgs *listFlags) {
	if listArgs.prefix != "" || listArgs.since != "" {
		var minTime *time.Time
		if listArgs.since != "" {
			t, err := argotime.ParseSince(listArgs.since)
			argoerrors.CheckError(err)
			minTime = t
		}
		tmpWorkflows := make([]wfv1.Workflow, 0)
		for _, wf := range wfList.Items {
			ok := filterByPrefix(&wf, listArgs.prefix) && filterBySince(&wf, minTime)
			if ok {
				tmpWorkflows = append(tmpWorkflows, wf)
			}
		}
		wfList.Items = tmpWorkflows
	}
}

func filterByPrefix(wf *wfv1.Workflow, prefix string) bool {
	return prefix == "" || strings.HasPrefix(wf.ObjectMeta.Name, prefix)
}

func filterBySince(wf *wfv1.Workflow, minTime *time.Time) bool {
	return minTime == nil || wf.Status.FinishedAt.IsZero() || wf.ObjectMeta.CreationTimestamp.After(*minTime)
}

func truncateWorkflowList(wfList *wfv1.WorkflowList, workflows *wfv1.Workflows, listArgs *listFlags) string {
	tail := listArgs.limit - int64(len(*workflows))
	lastWorkflowName := wfList.Items[tail-1].Name
	wfList.Items = wfList.Items[0:tail]
	return lastWorkflowName
}

func countPendingRunningCompleted(wf *wfv1.Workflow) (int, int, int) {
	pending := 0
	running := 0
	completed := 0
	for _, node := range wf.Status.Nodes {
		tmpl := wf.GetTemplateByName(node.TemplateName)
		if tmpl == nil || !tmpl.IsPodType() {
			continue
		}
		if node.Completed() {
			completed++
		} else if node.Phase == wfv1.NodeRunning {
			running++
		} else {
			pending++
		}
	}
	return pending, running, completed
}

// parameterString returns a human readable display string of the parameters, truncating if necessary
func parameterString(params []wfv1.Parameter) string {
	truncateString := func(str string, num int) string {
		bnoden := str
		if len(str) > num {
			if num > 3 {
				num -= 3
			}
			bnoden = str[0:num-15] + "..." + str[len(str)-15:]
		}
		return bnoden
	}

	pStrs := make([]string, 0)
	for _, p := range params {
		if p.Value != nil {
			str := fmt.Sprintf("%s=%s", p.Name, truncateString(*p.Value, 50))
			pStrs = append(pStrs, str)
		}
	}
	return strings.Join(pStrs, ",")
}
