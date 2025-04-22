package common

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/tabwriter"

	"github.com/argoproj/pkg/humanize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoutil "github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/printer"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

const onExitSuffix = "onExit"

type GetFlags struct {
	Output                  EnumFlagValue
	NodeFieldSelectorString string

	// Only used for backwards compatibility
	Status string
}

func statusToNodeFieldSelector(status string) string {
	return fmt.Sprintf("phase=%s", status)
}

func (g GetFlags) shouldPrint(node wfv1.NodeStatus) bool {
	if g.Status != "" {
		// Adapt --status to a node field selector for compatibility
		if g.NodeFieldSelectorString != "" {
			log.Fatalf("cannot use both --status and --node-field-selector")
		}
		g.NodeFieldSelectorString = statusToNodeFieldSelector(g.Status)
	}
	if g.NodeFieldSelectorString != "" {
		selector, err := fields.ParseSelector(g.NodeFieldSelectorString)
		if err != nil {
			log.Fatalf("selector is invalid: %s", err)
		}
		return util.SelectorMatchesNode(selector, node)
	}
	return true
}

func PrintWorkflowHelper(wf *wfv1.Workflow, getArgs GetFlags) string {
	const fmtStr = "%-20s %v\n"
	out := ""
	out += fmt.Sprintf(fmtStr, "Name:", wf.ObjectMeta.Name)
	out += fmt.Sprintf(fmtStr, "Namespace:", wf.ObjectMeta.Namespace)
	serviceAccount := wf.GetExecSpec().ServiceAccountName
	if serviceAccount == "" {
		// if serviceAccountName was not specified in a submitted Workflow, we will
		// use the serviceAccountName provided in Workflow Defaults (if any). If that
		// also isn't set, we will use the 'default' ServiceAccount in the namespace
		// the workflow will run in.
		if wf.Spec.WorkflowTemplateRef != nil {
			serviceAccount = "unset"
		} else {
			serviceAccount = "unset (will run with the default ServiceAccount)"
		}
	}
	out += fmt.Sprintf(fmtStr, "ServiceAccount:", serviceAccount)
	out += fmt.Sprintf(fmtStr, "Status:", printer.WorkflowStatus(wf))
	if wf.Status.Message != "" {
		out += fmt.Sprintf(fmtStr, "Message:", wf.Status.Message)
	}
	if len(wf.Status.Conditions) > 0 {
		out += wf.Status.Conditions.DisplayString(fmtStr, WorkflowConditionIconMap)
	}
	out += fmt.Sprintf(fmtStr, "Created:", humanize.Timestamp(wf.ObjectMeta.CreationTimestamp.Time))
	if !wf.Status.StartedAt.IsZero() {
		out += fmt.Sprintf(fmtStr, "Started:", humanize.Timestamp(wf.Status.StartedAt.Time))
	}
	if !wf.Status.FinishedAt.IsZero() {
		out += fmt.Sprintf(fmtStr, "Finished:", humanize.Timestamp(wf.Status.FinishedAt.Time))
	}
	if !wf.Status.StartedAt.IsZero() {
		out += fmt.Sprintf(fmtStr, "Duration:", humanize.RelativeDuration(wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time))
	}
	if wf.Status.Phase == wfv1.WorkflowRunning {
		if wf.Status.EstimatedDuration > 0 {
			out += fmt.Sprintf(fmtStr, "EstimatedDuration:", humanize.Duration(wf.Status.EstimatedDuration.ToDuration()))
		}
	}
	out += fmt.Sprintf(fmtStr, "Progress:", wf.Status.Progress)
	if !wf.Status.ResourcesDuration.IsZero() {
		out += fmt.Sprintf(fmtStr, "ResourcesDuration:", wf.Status.ResourcesDuration)
	}
	if len(wf.GetExecSpec().Arguments.Parameters) > 0 {
		out += fmt.Sprintf(fmtStr, "Parameters:", "")
		for _, param := range wf.GetExecSpec().Arguments.Parameters {
			if param.Value == nil {
				continue
			}
			out += fmt.Sprintf(fmtStr, "  "+param.Name+":", *param.Value)
		}
	}
	if wf.Status.Outputs != nil {
		if len(wf.Status.Outputs.Parameters) > 0 {
			out += fmt.Sprintf(fmtStr, "Output Parameters:", "")
			for _, param := range wf.Status.Outputs.Parameters {
				if param.HasValue() {
					out += fmt.Sprintf(fmtStr, "  "+param.Name+":", param.GetValue())
				}
			}
		}
		if len(wf.Status.Outputs.Artifacts) > 0 {
			out += fmt.Sprintf(fmtStr, "Output Artifacts:", "")
			for _, art := range wf.Status.Outputs.Artifacts {
				if art.S3 != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.S3.String())
				} else if art.Git != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.Git.String())
				} else if art.HTTP != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.HTTP.String())
				} else if art.Artifactory != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.Artifactory.String())
				} else if art.HDFS != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.HDFS.String())
				} else if art.Raw != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.Raw.String())
				} else if art.OSS != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.OSS.String())
				} else if art.GCS != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.GCS.String())
				} else if art.Azure != nil {
					out += fmt.Sprintf(fmtStr, "  "+art.Name+":", art.Azure.String())
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
		writerBuffer := new(bytes.Buffer)
		w := tabwriter.NewWriter(writerBuffer, 0, 0, 2, ' ', 0)
		out += "\n"
		// apply a dummy FgDefault format to align tab writer with the rest of the columns
		if getArgs.Output.String() == "wide" {
			_, _ = fmt.Fprintf(w, "%s\tTEMPLATE\tPODNAME\tDURATION\tARTIFACTS\tMESSAGE\tRESOURCESDURATION\tNODENAME\n", ansiFormat("STEP", FgDefault))
		} else if getArgs.Output.String() == "short" {
			_, _ = fmt.Fprintf(w, "%s\tTEMPLATE\tPODNAME\tDURATION\tMESSAGE\tNODENAME\n", ansiFormat("STEP", FgDefault))
		} else {
			_, _ = fmt.Fprintf(w, "%s\tTEMPLATE\tPODNAME\tDURATION\tMESSAGE\n", ansiFormat("STEP", FgDefault))
		}

		// Convert Nodes to Render Trees
		roots := convertToRenderTrees(wf)

		// Print main and onExit Trees
		mainRoot := roots[wf.ObjectMeta.Name]
		if mainRoot == nil {
			panic("failed to get the entrypoint node")
		}
		mainRoot.renderNodes(w, wf, 0, " ", " ", getArgs)

		onExitID := wf.NodeID(wf.ObjectMeta.Name + "." + onExitSuffix)
		if onExitRoot, ok := roots[onExitID]; ok {
			_, _ = fmt.Fprintf(w, "\t\t\t\t\t\n")
			onExitRoot.renderNodes(w, wf, 0, " ", " ", getArgs)
		}
		_ = w.Flush()
		if getArgs.Output.String() == "short" {
			out = writerBuffer.String()
		} else {
			out += writerBuffer.String()
		}
	}
	writerBuffer := new(bytes.Buffer)
	out += writerBuffer.String()
	return out
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
	renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, getArgs GetFlags)
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
	return (node == wfv1.NodeTypePod) || (node == wfv1.NodeTypeSkipped) || (node == wfv1.NodeTypeSuspend) || (node == wfv1.NodeTypeHTTP) || (node == wfv1.NodeTypePlugin)
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

			// If they are both elements of a list (e.g. withParams, withSequence, etc.) order by index number instead of
			// alphabetical order
			insertIndex := argoutil.RecoverIndexFromNodeName(insertName)
			equalIndex := argoutil.RecoverIndexFromNodeName(equalName)
			if insertIndex >= 0 && equalIndex >= 0 {
				if insertIndex < equalIndex {
					break
				}
			} else {
				if insertName < equalName {
					break
				}
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
func filterNode(node wfv1.NodeStatus, getArgs GetFlags) (bool, bool) {
	if node.Type == wfv1.NodeTypeRetry && len(node.Children) == 1 {
		return true, false
	} else if node.Type == wfv1.NodeTypeStepGroup {
		return true, true
	} else if node.Type == wfv1.NodeTypeSkipped && node.Phase == wfv1.NodeOmitted {
		return true, false
	} else if !getArgs.shouldPrint(node) {
		return true, false
	}
	return false, false
}

// Render the child of a given node based on information about the parent such as:
// whether it was filtered and does this child need special indent
func renderChild(w *tabwriter.Writer, wf *wfv1.Workflow, nInfo renderNode, depth int,
	nodePrefix string, childPrefix string, parentFiltered bool,
	childIndex int, maxIndex int, childIndent bool, getArgs GetFlags) {
	var part, subp string
	if NoUtf8 {
		if parentFiltered && childIndent {
			if maxIndex == 0 {
				part = "--"
				subp = "  "
			} else if childIndex == 0 {
				part = "+-"
				subp = "| "
			} else if childIndex == maxIndex {
				part = "`-"
				subp = "  "
			} else {
				part = "|-"
				subp = "| "
			}
		} else if !parentFiltered {
			if childIndex == maxIndex {
				part = "`-"
				subp = "  "
			} else {
				part = "|-"
				subp = "| "
			}
		}
	} else {
		if parentFiltered && childIndent {
			if maxIndex == 0 {
				part = "──"
				subp = "  "
			} else if childIndex == 0 {
				part = "┬─"
				subp = "│ "
			} else if childIndex == maxIndex {
				part = "└─"
				subp = "  "
			} else {
				part = "├─"
				subp = "│ "
			}
		} else if !parentFiltered {
			if childIndex == maxIndex {
				part = "└─"
				subp = "  "
			} else {
				part = "├─"
				subp = "│ "
			}
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
	nInfo.renderNodes(w, wf, depth, childNodePrefix, childChldPrefix, getArgs)
}

// Main method to print information of node in get
func printNode(w *tabwriter.Writer, node wfv1.NodeStatus, wfName, nodePrefix string, getArgs GetFlags, podNameVersion util.PodNameVersion) {
	nodeName := node.Name
	fmtNodeName := fmt.Sprintf("%s %s", JobStatusIconMap[node.Phase], node.DisplayName)
	if node.IsActiveSuspendNode() {
		fmtNodeName = fmt.Sprintf("%s %s", NodeTypeIconMap[node.Type], node.DisplayName)
	}
	templateName := util.GetTemplateFromNode(node)
	fmtTemplateName := ""
	if node.TemplateRef != nil {
		fmtTemplateName = fmt.Sprintf("%s/%s", node.TemplateRef.Name, node.TemplateRef.Template)
	} else if node.TemplateName != "" {
		fmtTemplateName = node.TemplateName
	}
	var args []interface{}
	duration := humanize.RelativeDurationShort(node.StartedAt.Time, node.FinishedAt.Time)
	if node.Type == wfv1.NodeTypePod {
		podName := util.GeneratePodName(wfName, nodeName, templateName, node.ID, podNameVersion)
		args = []interface{}{nodePrefix, fmtNodeName, fmtTemplateName, podName, duration, node.Message, ""}
	} else {
		args = []interface{}{nodePrefix, fmtNodeName, fmtTemplateName, "", "", node.Message, ""}
	}
	if getArgs.Output.String() == "wide" {
		msg := args[len(args)-2]
		args[len(args)-2] = getArtifactsString(node)
		args[len(args)-1] = msg
		args = append(args, node.ResourcesDuration, "")
		if node.Type == wfv1.NodeTypePod {
			args[len(args)-1] = node.HostNodeName
		}
		_, _ = fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", args...)
	} else if getArgs.Output.String() == "short" {
		if node.Type == wfv1.NodeTypePod {
			args[len(args)-1] = node.HostNodeName
		}
		_, _ = fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n", args...)
	} else {
		_, _ = fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n", args...)
	}
}

// renderNodes for each renderNode Type
// boundaryNode
func (nodeInfo *boundaryNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, getArgs GetFlags) {
	filtered, childIndent := filterNode(nodeInfo.getNodeStatus(wf), getArgs)
	if !filtered {
		version := util.GetWorkflowPodNameVersion(wf)
		printNode(w, nodeInfo.getNodeStatus(wf), wf.ObjectMeta.Name, nodePrefix, getArgs, version)
	}

	for i, nInfo := range nodeInfo.boundaryContained {
		renderChild(w, wf, nInfo, depth, nodePrefix, childPrefix, filtered, i,
			len(nodeInfo.boundaryContained)-1, childIndent, getArgs)
	}
}

// nonBoundaryParentNode
func (nodeInfo *nonBoundaryParentNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, depth int, nodePrefix string, childPrefix string, getArgs GetFlags) {
	filtered, childIndent := filterNode(nodeInfo.getNodeStatus(wf), getArgs)
	if !filtered {
		version := util.GetWorkflowPodNameVersion(wf)
		printNode(w, nodeInfo.getNodeStatus(wf), wf.ObjectMeta.Name, nodePrefix, getArgs, version)
	}

	for i, nInfo := range nodeInfo.children {
		renderChild(w, wf, nInfo, depth, nodePrefix, childPrefix, filtered, i,
			len(nodeInfo.children)-1, childIndent, getArgs)
	}
}

// executionNode
func (nodeInfo *executionNode) renderNodes(w *tabwriter.Writer, wf *wfv1.Workflow, _ int, nodePrefix string, _ string, getArgs GetFlags) {
	filtered, _ := filterNode(nodeInfo.getNodeStatus(wf), getArgs)
	if !filtered {
		version := util.GetWorkflowPodNameVersion(wf)
		printNode(w, nodeInfo.getNodeStatus(wf), wf.ObjectMeta.Name, nodePrefix, getArgs, version)
	}
}

func getArtifactsString(node wfv1.NodeStatus) string {
	if node.Outputs == nil {
		return ""
	}
	var artNames []string
	for _, art := range node.Outputs.Artifacts {
		artNames = append(artNames, art.Name)
	}
	return strings.Join(artNames, ",")
}
