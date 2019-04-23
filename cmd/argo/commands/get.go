package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/pkg/humanize"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const onExitSuffix = "onExit"

func NewGetCommand() *cobra.Command {
	var (
		output string
	)

	var command = &cobra.Command{
		Use:   "get WORKFLOW",
		Short: "display details about a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			wfClient := InitWorkflowClient()
			wf, err := wfClient.Get(args[0], metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}
			err = util.DecompressWorkflow(wf)
			if err != nil {
				log.Fatal(err)
			}
			printWorkflow(wf, output)
		},
	}

	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: json|yaml|wide")
	command.Flags().BoolVar(&noColor, "no-color", false, "Disable colorized output")
	return command
}

func printWorkflow(wf *wfv1.Workflow, outFmt string) {
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
		printWorkflowHelper(wf, outFmt)
	default:
		log.Fatalf("Unknown output format: %s", outFmt)
	}
}

func printWorkflowHelper(wf *wfv1.Workflow, outFmt string) {
	const fmtStr = "%-20s %v\n"
	fmt.Printf(fmtStr, "Name:", wf.ObjectMeta.Name)
	fmt.Printf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
	serviceAccount := wf.Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	fmt.Printf(fmtStr, "ServiceAccount:", serviceAccount)
	fmt.Printf(fmtStr, "Status:", workflowStatus(wf))
	if wf.Status.Message != "" {
		fmt.Printf(fmtStr, "Message:", wf.Status.Message)
	}
	fmt.Printf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
	if !wf.Status.StartedAt.IsZero() {
		fmt.Printf(fmtStr, "Started:", humanize.Timestamp(wf.Status.StartedAt.Time))
	}
	if !wf.Status.FinishedAt.IsZero() {
		fmt.Printf(fmtStr, "Finished:", humanize.Timestamp(wf.Status.FinishedAt.Time))
	}
	if !wf.Status.StartedAt.IsZero() {
		fmt.Printf(fmtStr, "Duration:", humanize.RelativeDuration(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time))
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
	if wf.Status.Outputs != nil {
		//fmt.Printf(fmtStr, "Outputs:", "")
		if len(wf.Status.Outputs.Parameters) > 0 {
			fmt.Printf(fmtStr, "Output Parameters:", "")
			for _, param := range wf.Status.Outputs.Parameters {
				fmt.Printf(fmtStr, "  "+param.Name+":", *param.Value)
			}
		}
		if len(wf.Status.Outputs.Artifacts) > 0 {
			fmt.Printf(fmtStr, "Output Artifacts:", "")
			for _, art := range wf.Status.Outputs.Artifacts {
				if art.S3 != nil {
					fmt.Printf(fmtStr, "  "+art.Name+":", art.S3.String())
				} else if art.Artifactory != nil {
					fmt.Printf(fmtStr, "  "+art.Name+":", art.Artifactory.String())
				}
			}
		}
	}
	printTree := true
	if wf.Status.Nodes == nil {
		printTree = false
	} else if _, ok := wf.Status.Nodes[wf.ObjectMeta.Name]; !ok {
		printTree = false
	}
	if printTree {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Println()
		// apply a dummy FgDefault format to align tabwriter with the rest of the columns
		if outFmt == "wide" {
			fmt.Fprintf(w, "%s\tPODNAME\tDURATION\tARTIFACTS\tMESSAGE\n", ansiFormat("STEP", FgDefault))
		} else {
			fmt.Fprintf(w, "%s\tPODNAME\tDURATION\tMESSAGE\n", ansiFormat("STEP", FgDefault))
		}

		// Convert Nodes to Render Trees
		roots := convertToRenderTrees(wf)

		// Print main and onExit Trees
		mainRoot := roots[wf.ObjectMeta.Name]
		mainRoot.renderNodes(w, wf, 0, " ", " ", outFmt)

		onExitID := wf.NodeID(wf.ObjectMeta.Name + "." + onExitSuffix)
		if onExitRoot, ok := roots[onExitID]; ok {
			fmt.Fprintf(w, "\t\t\t\t\t\n")
			onExitRoot.renderNodes(w, wf, 0, " ", " ", outFmt)
		}
		_ = w.Flush()
	}
}

type nodeInfoInterface interface {
	getID() string
	getNodeStatus(wf *wfv1.Workflow) wfv1.NodeStatus
	getStartTime(wf *wfv1.Workflow) metav1.Time
}

type nodeInfo struct {
	id string
}

func (n *nodeInfo) getID() string {
	return n.id
}

func (n *nodeInfo) getNodeStatus(wf *wfv1.Workflow) wfv1.NodeStatus {
	return wf.Status.Nodes[n.id]
}

func (n *nodeInfo) getStartTime(wf *wfv1.Workflow) metav1.Time {
	return wf.Status.Nodes[n.id].StartedAt
}

// Interface to represent Nodes in render form types
type renderNode interface {
	// Render this renderNode and its children
	renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, outFmt string)
	nodeInfoInterface
}

// Currently this is Pod or Resource Nodes
type executionNode struct {
	nodeInfo
}

// Currently this is the step groups or retry nodes
type nonBoundaryParentNode struct {
	nodeInfo
	children []renderNode // Can be boundaryNode or executionNode
}

// Currently this is the virtual Template node
type boundaryNode struct {
	nodeInfo
	boundaryContained []renderNode // Can be nonBoundaryParent or executionNode or boundaryNode
}

func isBoundaryNode(node wfv1.NodeType) bool {
	return (node == wfv1.NodeTypeDAG) || (node == wfv1.NodeTypeSteps)
}

func isNonBoundaryParentNode(node wfv1.NodeType) bool {
	return (node == wfv1.NodeTypeStepGroup) || (node == wfv1.NodeTypeRetry)
}

func isExecutionNode(node wfv1.NodeType) bool {
	return (node == wfv1.NodeTypePod) || (node == wfv1.NodeTypeSkipped) || (node == wfv1.NodeTypeSuspend)
}

func insertSorted(wf *wfv1.Workflow, sortedArray []renderNode, item renderNode) []renderNode {
	insertTime := item.getStartTime(wf)
	var index int
	for index = 0; index < len(sortedArray); index++ {
		existingItem := sortedArray[index]
		t := existingItem.getStartTime(wf)
		if insertTime.Before(&t) {
			break
		} else if insertTime.Equal(&t) {
			// If they are equal apply alphabetical order so we
			// get some consistent printing
			insertName := item.getNodeStatus(wf).DisplayName
			equalName := existingItem.getNodeStatus(wf).DisplayName
			if insertName < equalName {
				break
			}
		}
	}
	sortedArray = append(sortedArray, nil)
	copy(sortedArray[index+1:], sortedArray[index:])
	sortedArray[index] = item
	return sortedArray
}

// Attach render node n to its parent based on what has been parsed previously
// In some cases add it to list of things that still needs to be attached to parent
// Return if I am a possible root
func attachToParent(wf *wfv1.Workflow, n renderNode,
	nonBoundaryParentChildrenMap map[string]*nonBoundaryParentNode, boundaryID string,
	boundaryNodeMap map[string]*boundaryNode, parentBoundaryMap map[string][]renderNode) bool {

	// Check first if I am a child of a nonBoundaryParent
	// that implies I attach to that instead of my boundary. This was already
	// figured out in Pass 1
	if nonBoundaryParent, ok := nonBoundaryParentChildrenMap[n.getID()]; ok {
		nonBoundaryParent.children = insertSorted(wf, nonBoundaryParent.children, n)
		return false
	}

	// If I am not attached to a nonBoundaryParent and I have no Boundary ID then
	// I am a possible root
	if boundaryID == "" {
		return true
	}
	if parentBoundary, ok := boundaryNodeMap[boundaryID]; ok {
		parentBoundary.boundaryContained = insertSorted(wf, parentBoundary.boundaryContained, n)
	} else {
		// put ourselves to be added by the parent when we get to it later
		if _, ok := parentBoundaryMap[boundaryID]; !ok {
			parentBoundaryMap[boundaryID] = make([]renderNode, 0)
		}
		parentBoundaryMap[boundaryID] = append(parentBoundaryMap[boundaryID], n)
	}
	return false
}

// This takes the map of NodeStatus and converts them into a forrest
// of trees of renderNodes and returns the set of roots for each tree
func convertToRenderTrees(wf *wfv1.Workflow) map[string]renderNode {

	renderTreeRoots := make(map[string]renderNode)

	// Used to store all boundary nodes so future render children can attach
	// Maps node Name -> *boundaryNode
	boundaryNodeMap := make(map[string]*boundaryNode)
	// Used to store children of a boundary node that has not been parsed yet
	// Maps boundary Node name -> array of render Children
	parentBoundaryMap := make(map[string][]renderNode)

	// Used to store Non Boundary Parent nodes so render children can attach
	// Maps non Boundary Parent Node name -> *nonBoundaryParentNode
	nonBoundaryParentMap := make(map[string]*nonBoundaryParentNode)
	// Used to store children which have a Non Boundary Parent from rendering perspective
	// Maps non Boundary render Children name -> *nonBoundaryParentNode
	nonBoundaryParentChildrenMap := make(map[string]*nonBoundaryParentNode)

	// We have to do a 2 pass approach because anything that is a child
	// of a nonBoundaryParent and also has a boundaryID we may not know which
	// parent to attach to if we didn't see the nonBoundaryParent earlier
	// in a 1 pass strategy

	// 1st Pass Process enough of nonBoundaryParent nodes to know all their children
	for id, status := range wf.Status.Nodes {
		if status.Type == "" {
			log.Fatal("Missing node type in status node. Cannot get workflows created with Argo <= 2.0 using the default or wide output option.")
			return nil
		}
		if isNonBoundaryParentNode(status.Type) {
			n := nonBoundaryParentNode{nodeInfo: nodeInfo{id: id}}
			nonBoundaryParentMap[id] = &n

			for _, child := range status.Children {
				nonBoundaryParentChildrenMap[child] = &n
			}
		}
	}

	// 2nd Pass process everything
	for id, status := range wf.Status.Nodes {
		switch {
		case isBoundaryNode(status.Type):
			n := boundaryNode{nodeInfo: nodeInfo{id: id}}
			boundaryNodeMap[id] = &n
			// Attach to my parent if needed
			if attachToParent(wf, &n, nonBoundaryParentChildrenMap,
				status.BoundaryID, boundaryNodeMap, parentBoundaryMap) {
				renderTreeRoots[n.getID()] = &n
			}
			// Attach nodes who are in my boundary already seen before me to me
			for _, val := range parentBoundaryMap[id] {
				n.boundaryContained = insertSorted(wf, n.boundaryContained, val)
			}
		case isNonBoundaryParentNode(status.Type):
			nPtr, ok := nonBoundaryParentMap[id]
			if !ok {
				log.Fatal("Unable to lookup node " + id)
				return nil
			}
			// Attach to my parent if needed
			if attachToParent(wf, nPtr, nonBoundaryParentChildrenMap,
				status.BoundaryID, boundaryNodeMap, parentBoundaryMap) {
				renderTreeRoots[nPtr.getID()] = nPtr
			}
			// All children attach directly to the nonBoundaryParents since they are already created
			// in pass 1 so no need to do that here
		case isExecutionNode(status.Type):
			n := executionNode{nodeInfo: nodeInfo{id: id}}
			// Attach to my parent if needed
			if attachToParent(wf, &n, nonBoundaryParentChildrenMap,
				status.BoundaryID, boundaryNodeMap, parentBoundaryMap) {
				renderTreeRoots[n.getID()] = &n
			}
			// Execution nodes don't have other render nodes as children
		}
	}

	return renderTreeRoots
}

// This function decides if a Node will be filtered from rendering and returns
// two things. First argument tells if the node is filtered and second argument
// tells whether the children need special indentation due to filtering
// Return Values: (is node filtered, do children need special indent)
func filterNode(node wfv1.NodeStatus) (bool, bool) {
	if node.Type == wfv1.NodeTypeRetry && len(node.Children) == 1 {
		return true, false
	} else if node.Type == wfv1.NodeTypeStepGroup {
		return true, true
	}
	return false, false
}

// Render the child of a given node based on information about the parent such as:
// whether it was filtered and does this child need special indent
func renderChild(w *tabwriter.Writer, wf *wfv1.Workflow, nInfo renderNode, depth int,
	nodePrefix string, childPrefix string, parentFiltered bool,
	childIndex int, maxIndex int, childIndent bool, outFmt string) {
	var part, subp string
	if parentFiltered && childIndent {
		if maxIndex == 0 {
			part = "--"
			subp = "  "
		} else if childIndex == 0 {
			part = "·-"
			subp = "| "
		} else if childIndex == maxIndex {
			part = "└-"
			subp = "  "
		} else {
			part = "├-"
			subp = "| "
		}
	} else if !parentFiltered {
		if childIndex == maxIndex {
			part = "└-"
			subp = "  "
		} else {
			part = "├-"
			subp = "| "
		}
	}
	var childNodePrefix, childChldPrefix string
	if !parentFiltered {
		depth = depth + 1
		childNodePrefix = childPrefix + part
		childChldPrefix = childPrefix + subp
	} else {
		if childIndex == 0 {
			childNodePrefix = nodePrefix + part
		} else {
			childNodePrefix = childPrefix + part
		}
		childChldPrefix = childPrefix + subp
	}
	nInfo.renderNodes(w, wf, depth, childNodePrefix, childChldPrefix, outFmt)
}

// Main method to print information of node in get
func printNode(w *tabwriter.Writer, wf *wfv1.Workflow, node wfv1.NodeStatus, depth int, nodePrefix string, childPrefix string, outFmt string) {
	nodeName := fmt.Sprintf("%s %s", jobStatusIconMap[node.Phase], node.DisplayName)
	var args []interface{}
	duration := humanize.RelativeDurationShort(node.StartedAt.Time, node.FinishedAt.Time)
	if node.Type == wfv1.NodeTypePod {
		args = []interface{}{nodePrefix, nodeName, node.ID, duration, node.Message}
	} else {
		args = []interface{}{nodePrefix, nodeName, "", "", node.Message}
	}
	if outFmt == "wide" {
		msg := args[len(args)-1]
		args[len(args)-1] = getArtifactsString(node)
		args = append(args, msg)
		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n", args...)
	} else {
		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", args...)
	}
}

// renderNodes for each renderNode Type
// boundaryNode
func (nodeInfo *boundaryNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, outFmt string) {
	filtered, childIndent := filterNode(nodeInfo.getNodeStatus(wf))
	if !filtered {
		printNode(w, wf, nodeInfo.getNodeStatus(wf), depth, nodePrefix, childPrefix, outFmt)
	}

	for i, nInfo := range nodeInfo.boundaryContained {
		renderChild(w, wf, nInfo, depth, nodePrefix, childPrefix, filtered, i,
			len(nodeInfo.boundaryContained)-1, childIndent, outFmt)
	}
}

// nonBoundaryParentNode
func (nodeInfo *nonBoundaryParentNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, outFmt string) {
	filtered, childIndent := filterNode(nodeInfo.getNodeStatus(wf))
	if !filtered {
		printNode(w, wf, nodeInfo.getNodeStatus(wf), depth, nodePrefix, childPrefix, outFmt)
	}

	for i, nInfo := range nodeInfo.children {
		renderChild(w, wf, nInfo, depth, nodePrefix, childPrefix, filtered, i,
			len(nodeInfo.children)-1, childIndent, outFmt)
	}
}

// executionNode
func (nodeInfo *executionNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, outFmt string) {
	filtered, _ := filterNode(nodeInfo.getNodeStatus(wf))
	if !filtered {
		printNode(w, wf, nodeInfo.getNodeStatus(wf), depth, nodePrefix, childPrefix, outFmt)
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
