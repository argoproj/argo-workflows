package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	humanize "github.com/dustin/go-humanize"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&getArgs.output, "output", "o", "", "Output format. One of: json|yaml|wide")
	getCmd.Flags().BoolVar(&globalArgs.noColor, "no-color", false, "Disable colorized output")
}

type getFlags struct {
	output string // --output
}

var getArgs getFlags

var getCmd = &cobra.Command{
	Use:   "get WORKFLOW",
	Short: "display details about a workflow",
	Run:   GetWorkflow,
}

// GetWorkflow gets the workflow passed in as args
func GetWorkflow(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	wfClient := InitWorkflowClient()
	wf, err := wfClient.GetWorkflow(args[0])
	if err != nil {
		log.Fatal(err)
	}
	printWorkflow(getArgs.output, wf)
}

func printWorkflow(outFmt string, wf *wfv1.Workflow) {
	switch outFmt {
	case "name":
		fmt.Println(wf.ObjectMeta.Name)
	case "json":
		outBytes, _ := json.MarshalIndent(wf, "", "    ")
		fmt.Println(string(outBytes))
	case "yaml":
		outBytes, _ := yaml.Marshal(wf)
		fmt.Print(string(outBytes))
	case "wide", "":
		printWorkflowHelper(wf)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printWorkflowHelper(wf *wfv1.Workflow) {
	const fmtStr = "%-17s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
	fmt.Printf(fmtStr, "Status:", worklowStatus(wf))
	if wf.Status.Message != "" {
		fmt.Printf(fmtStr, "Message:", wf.Status.Message)
	}
	fmt.Printf(fmtStr, "Created:", humanizeTimestamp(wf.ObjectMeta.CreationTimestamp.Unix()))
	if !wf.Status.StartedAt.IsZero() {
		fmt.Printf(fmtStr, "Started:", humanizeTimestamp(wf.Status.StartedAt.Unix()))
	}
	if !wf.Status.FinishedAt.IsZero() {
		fmt.Printf(fmtStr, "Finished:", humanizeTimestamp(wf.Status.FinishedAt.Unix()))
	}
	if !wf.Status.StartedAt.IsZero() {
		var duration time.Duration
		if !wf.Status.FinishedAt.IsZero() {
			duration = time.Second * time.Duration(wf.Status.FinishedAt.Unix()-wf.Status.StartedAt.Unix())
		} else {
			duration = time.Second * time.Duration(time.Now().UTC().Unix()-wf.Status.StartedAt.Unix())
		}
		fmt.Printf(fmtStr, "Duration:", humanizeDuration(duration))
	}

	if len(wf.Spec.Arguments.Parameters) > 0 {
		fmt.Printf(fmtStr, "Parameters:", "")
		for _, param := range wf.Spec.Arguments.Parameters {
			if param.Value == nil {
				continue
			}
			fmt.Printf(fmtStr, "  "+param.Name+":", *param.Value)
		}
	}

	if wf.Status.Nodes != nil {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Println()
		// apply a dummy FgDefault format to align tabwriter with the rest of the columns
		if getArgs.output == "wide" {
			fmt.Fprintf(w, "%s\tPODNAME\tDURATION\tARTIFACTS\tMESSAGE\n", ansiFormat("STEP", FgDefault))
		} else {
			fmt.Fprintf(w, "%s\tPODNAME\tMESSAGE\n", ansiFormat("STEP", FgDefault))
		}
		node, ok := wf.Status.Nodes[wf.ObjectMeta.Name]
		if ok {
			printNodeTree(w, wf, node, 0, " ", " ")
		}
		onExitNode, ok := wf.Status.Nodes[wf.NodeID(wf.ObjectMeta.Name+".onExit")]
		if ok {
			fmt.Fprintf(w, "\t\t\t\t\n")
			onExitNode.Name = "onExit"
			printNodeTree(w, wf, onExitNode, 0, " ", " ")
		}
		_ = w.Flush()
	}
}

func printNodeTree(w *tabwriter.Writer, wf *wfv1.Workflow, node wfv1.NodeStatus, depth int, nodePrefix string, childPrefix string) {
	nodeName := fmt.Sprintf("%s %s", jobStatusIconMap[node.Phase], node.Name)
	var args []interface{}
	if len(node.Children) == 0 && node.Phase != wfv1.NodeSkipped {
		args = []interface{}{nodePrefix, nodeName, node.ID, node.Message}
	} else {
		args = []interface{}{nodePrefix, nodeName, "", ""}
	}
	if getArgs.output == "wide" {
		msg := args[len(args)-1]
		args[len(args)-1] = humanizeDurationShort(node.StartedAt, node.FinishedAt)
		args = append(args, getArtifactsString(node))
		args = append(args, msg)
		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n", args...)
	} else {
		fmt.Fprintf(w, "%s%s\t%s\t%s\n", args...)
	}

	// If the node has children, the node is a workflow template and
	// node.Children prepresent a list of parallel steps. We skip
	// a generation when recursing since the children nodes of workflow
	// templates represent a virtual step group, which are not worh printing.
	for i, stepGroupNodeID := range node.Children {
		lastStepGroup := bool(i == len(node.Children)-1)
		var part1, subp1 string
		if lastStepGroup {
			part1 = "└-"
			subp1 = "  "
		} else {
			part1 = "├-"
			subp1 = "| "
		}
		stepGroupNode := wf.Status.Nodes[stepGroupNodeID]
		for j, childNodeID := range stepGroupNode.Children {
			childNode := wf.Status.Nodes[childNodeID]
			if j > 0 {
				if lastStepGroup {
					part1 = "  "
				} else {
					part1 = "| "
				}
			}
			firstParallel := bool(j == 0)
			lastParallel := bool(j == len(stepGroupNode.Children)-1)
			var part2, subp2 string
			if firstParallel {
				if len(stepGroupNode.Children) == 1 {
					part2 = "--"
				} else {
					part2 = "·-"
				}
				if !lastParallel {
					subp2 = "| "
				} else {
					subp2 = "  "
				}

			} else if lastParallel {
				part2 = "└-"
				subp2 = "  "
			} else {
				part2 = "├-"
				subp2 = "| "
			}
			childNodePrefix := childPrefix + part1 + part2
			childChldPrefix := childPrefix + subp1 + subp2
			// Remove stepgroup name from being displayed
			childNode.Name = strings.TrimPrefix(childNode.Name, stepGroupNode.Name+".")
			printNodeTree(w, wf, childNode, depth+1, childNodePrefix, childChldPrefix)
		}
	}
}

func getArtifactsString(node wfv1.NodeStatus) string {
	if node.Outputs == nil {
		return ""
	}
	artNames := []string{}
	for _, art := range node.Outputs.Artifacts {
		artNames = append(artNames, art.Name)
	}
	return strings.Join(artNames, ",")
}

func humanizeTimestamp(epoch int64) string {
	ts := time.Unix(epoch, 0)
	return fmt.Sprintf("%s (%s)", ts.Format("Mon Jan 02 15:04:05 -0700"), humanize.Time(ts))
}

func humanizeDurationShort(start, finish metav1.Time) string {
	if finish.IsZero() {
		finish = metav1.Time{Time: time.Now().UTC()}
	}
	return humanize.CustomRelTime(start.Time, finish.Time, "", "", timeMagnitudes)
}

// humanizeDuration humanizes time.Duration output to a meaningful value,
func humanizeDuration(duration time.Duration) string {
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d minutes %d seconds", int64(duration.Minutes()), int64(remainingSeconds))
	}
	if duration.Hours() < 24.0 {
		remainingMinutes := math.Mod(duration.Minutes(), 60)
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		return fmt.Sprintf("%d hours %d minutes %d seconds",
			int64(duration.Hours()), int64(remainingMinutes), int64(remainingSeconds))
	}
	remainingHours := math.Mod(duration.Hours(), 24)
	remainingMinutes := math.Mod(duration.Minutes(), 60)
	remainingSeconds := math.Mod(duration.Seconds(), 60)
	return fmt.Sprintf("%d days %d hours %d minutes %d seconds",
		int64(duration.Hours()/24), int64(remainingHours),
		int64(remainingMinutes), int64(remainingSeconds))
}
