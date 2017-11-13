package commands

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get WORKFLOW",
	Short: "get commands",
	Run:   getWorkflow,
}

func getWorkflow(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	wfClient := initWorkflowClient()
	wf, err := wfClient.GetWorkflow(args[0])
	if err != nil {
		log.Fatal(err)
	}
	printWorkflow(wf)
}

func printWorkflow(wf *wfv1.Workflow) {
	const fmtStr = "%-17s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Status:", worklowStatus(wf))
	fmt.Printf(fmtStr, "Created:", humanizeTimestamp(wf.ObjectMeta.CreationTimestamp.Unix()))

	//var duration time.Duration
	// if svc.EndTime > 0 {
	// 	fmt.Printf(svcFmtStr, "Completed:", humanizeTimestamp(svc.EndTime))
	// 	duration = time.Second * time.Duration(svc.EndTime-svc.LaunchTime)
	// } else {
	// 	duration = 0
	// }
	//fmt.Printf(fmtStr, "Duration:", humanizeDuration(duration))
	node, ok := wf.Status.Nodes[wf.ObjectMeta.Name]
	if !ok {
		// can get here if we just created the workflow
		return
	}
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "STEP\tPODNAME\n")
	printNodeTree(w, wf, node, 0, " ", " ")
	w.Flush()
}

func printNodeTree(w *tabwriter.Writer, wf *wfv1.Workflow, node wfv1.NodeStatus, depth int, nodePrefix string, childPrefix string) {
	nodeName := fmt.Sprintf("%s %s", jobStatusIconMap[node.Status], node.Name)
	if len(node.Children) == 0 && node.Status != wfv1.NodeStatusSkipped {
		fmt.Fprintf(w, "%s%s\t%s\n", nodePrefix, nodeName, node.ID)
	} else {
		fmt.Fprintf(w, "%s%s\t\n", nodePrefix, nodeName)
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
			j = j + 1
		}
	}
}

func humanizeTimestamp(epoch int64) string {
	ts := time.Unix(epoch, 0)
	return fmt.Sprintf("%s (%s)", ts.Format("Mon Jan 02 15:04:05 -0700"), humanize.Time(ts))
}
