package parameterization

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// retryInvocation identifies a declared call, not a scheduling node. Native
// Argo still computes the retry reset; we reject resets outside this analysis.
type retryInvocation struct {
	task        wfv1.DAGTask
	template    wfv1.Template
	affected    bool
	replaceLoop bool
	children    map[string]*retryInvocation
	stepGroups  map[string]int
}
type retryGraph struct {
	root      *retryInvocation
	leaves    map[string]*retryInvocation
	templates map[string]wfv1.Template
	changed   []string
	calls     int
	selected  map[string]bool
	seen      map[string]bool
	required  map[*retryInvocation]bool
}

func newRetryGraph(spec *wfv1.WorkflowSpec, changed []string, callPaths ...string) (*retryGraph, error) {
	g := &retryGraph{templates: map[string]wfv1.Template{}, leaves: map[string]*retryInvocation{}, changed: changed, selected: map[string]bool{}, seen: map[string]bool{}, required: map[*retryInvocation]bool{}}
	for _, path := range callPaths {
		if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") || g.selected[path] {
			return nil, fmt.Errorf("retry call paths must be unique declared invocation paths")
		}
		g.selected[path] = true
	}
	for _, t := range spec.Templates {
		g.templates[t.Name] = t
	}
	entry, ok := g.templates[spec.Entrypoint]
	if !ok || (entry.DAG == nil && len(entry.Steps) == 0) {
		return nil, fmt.Errorf("partial retry requires a frozen DAG or Steps entrypoint")
	}
	// A changing global parameter may only enter through call arguments. Local
	// input dependencies are checked in each individual invocation below.
	structural := spec.DeepCopy()
	for i := range structural.Templates {
		clearRetryArguments(&structural.Templates[i])
		// A resource leaf may consume workflow parameters directly. Its exact
		// references are attributed to each invocation below, not to all calls.
		structural.Templates[i].Resource = nil
	}
	refs, err := p.References(structural)
	if err != nil {
		return nil, err
	}
	if refs.Uses("workflow", changed) {
		return nil, fmt.Errorf("changed parameters affect a condition, loop identity or implementation outside task arguments")
	}
	rootInputs := []string{}
	for _, input := range entry.Inputs.Parameters {
		for _, name := range changed {
			if name == input.Name {
				rootInputs = append(rootInputs, name)
			}
		}
	}
	g.root, err = g.analyze(entry, wfv1.DAGTask{}, "", rootInputs, false, map[string]bool{})
	if err == nil {
		for path := range g.selected {
			if !g.seen[path] {
				return nil, fmt.Errorf("unknown retry call path %s", path)
			}
		}
		g.markRequired(g.root)
		for path, call := range g.leaves {
			if call.affected && call.template.Suspend == nil && call.template.Metadata.Annotations[p.Group+"/retry-safe"] != "idempotent" {
				return nil, fmt.Errorf("task %s must explicitly declare retry-safe=idempotent", path)
			}
		}
	}
	return g, err
}

func (g *retryGraph) analyze(t wfv1.Template, call wfv1.DAGTask, path string, inputs []string, force bool, ancestors map[string]bool) (*retryInvocation, error) {
	// Native input defaults are evaluated for each invocation. A reference in
	// one default must not invalidate every use of the template, especially
	// calls which supply their own argument. Attribute it to this call before
	// checking whether those inputs affect children or structural fields.
	inputs = slices.Clone(inputs)
	for pass := 0; pass < len(t.Inputs.Parameters); pass++ {
		before := len(inputs)
		for _, input := range t.Inputs.Parameters {
			argument := call.Arguments.GetParameterByName(input.Name)
			if argument != nil && argument.Value != nil {
				continue
			}
			refs, err := p.References(struct {
				Default *wfv1.AnyString
				Value   *wfv1.AnyString
			}{input.Default, input.Value})
			if err != nil {
				return nil, err
			}
			if (refs.Uses("workflow", g.changed) || refs.Uses("inputs", inputs)) && !slices.Contains(inputs, input.Name) {
				inputs = append(inputs, input.Name)
			}
		}
		if len(inputs) == before {
			break
		}
	}
	g.calls++
	if g.calls > 1000 || len(ancestors) >= 64 || ancestors[t.Name] {
		return nil, fmt.Errorf("partial retry requires bounded nonrecursive calls")
	}
	if (t.RetryStrategy != nil && t.Container == nil && t.Script == nil && t.Resource == nil) || t.ContainerSet != nil || t.Memoize != nil || (t.Daemon != nil && *t.Daemon) {
		return nil, fmt.Errorf("call %s needs artifact/retry-group/daemon reuse evidence not supported yet", path)
	}
	if err := validateRetryArtifactContract(t, call); err != nil {
		return nil, err
	}
	if g.selected[path] {
		force = true
		g.seen[path] = true
	}
	n := &retryInvocation{task: call, template: t, affected: force || len(inputs) > 0}
	if t.DAG == nil && len(t.Steps) == 0 {
		refs, err := p.References(t.Resource)
		if err != nil {
			return nil, err
		}
		n.affected = n.affected || refs.Uses("workflow", g.changed)
		g.leaves[path] = n
		return n, nil
	}
	structural := t.DeepCopy()
	clearRetryArguments(structural)
	refs, err := p.References(structural)
	if err != nil {
		return nil, err
	}
	if refs.Uses("inputs", inputs) {
		return nil, fmt.Errorf("changed inputs affect DAG structure or outputs outside child call arguments at %s", path)
	}
	n.children = map[string]*retryInvocation{}
	declared, groups, err := retryCalls(t)
	if err != nil {
		return nil, err
	}
	n.stepGroups = groups
	n.affected = false
	tasks := map[string]wfv1.DAGTask{}
	for _, task := range declared {
		if _, ok := tasks[task.Name]; ok {
			return nil, fmt.Errorf("duplicate task identity")
		}
		tasks[task.Name] = task
	}
	ancestors[t.Name] = true
	defer delete(ancestors, t.Name)
	visiting := map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		if _, ok := n.children[name]; ok {
			return nil
		}
		task, ok := tasks[name]
		if !ok || visiting[name] {
			return fmt.Errorf("invalid explicit dependency at %s", path)
		}
		visiting[name] = true
		changedDependencies := []string{}
		for _, dep := range task.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
			if n.children[dep].affected {
				changedDependencies = append(changedDependencies, dep)
			}
		}
		child, ok := g.templates[task.Template]
		// Legacy onExit must remain a retry boundary until hook replay is supported.
		if !ok || task.TemplateRef != nil || task.Inline != nil || len(task.Hooks) > 0 || task.OnExit != "" || task.Depends != "" {
			return fmt.Errorf("partial retry requires resolved calls and explicit dependencies")
		}
		dependencyAffected := len(changedDependencies) > 0
		if child.Metadata.Annotations[p.Group+"/retry-dependencies"] == "inputs" {
			// Explicit ordering-only dependencies do not invalidate immutable
			// results. Output references still carry data dependencies, and a
			// directly selected parent always forces its full invocation.
			refs, err := p.References(task)
			if err != nil {
				return err
			}
			dependencyAffected = refs.Uses("tasks", changedDependencies) || refs.Uses("steps", changedDependencies)
		}
		forced := force || dependencyAffected
		loopRefs, loopErr := p.References(struct {
			WithParam    string
			WithItems    []wfv1.Item
			WithSequence *wfv1.Sequence
		}{task.WithParam, task.WithItems, task.WithSequence})
		if loopErr != nil {
			return loopErr
		}
		replaceLoop := hasRetryLoop(task) && (loopRefs.Uses("workflow", g.changed) || loopRefs.Uses("inputs", inputs) || (forced && (loopRefs.Uses("tasks", task.Dependencies) || loopRefs.Uses("steps", task.Dependencies))))
		if replaceLoop && !allowsLoopReplacement(t, task.Name) {
			return fmt.Errorf("loop topology change at %s/%s requires author-approved retry-loop-calls", path, task.Name)
		}
		forced = forced || replaceLoop
		changedInputs := []string{}
		for _, arg := range task.Arguments.Parameters {
			refs, err := p.References(arg)
			if err != nil {
				return err
			}
			if refs.Uses("workflow", g.changed) || refs.Uses("inputs", inputs) {
				changedInputs = append(changedInputs, arg.Name)
			}
		}
		artifactRefs, artifactErr := p.References(task.Arguments.Artifacts)
		if artifactErr != nil {
			return artifactErr
		}
		for producer := range artifactRefs["tasks"] {
			if !slices.Contains(task.Dependencies, producer) {
				return fmt.Errorf("artifact producer must be an explicit dependency")
			}
		}
		forced = forced || artifactRefs.Uses("workflow", g.changed) || artifactRefs.Uses("inputs", inputs)
		childPath := task.Name
		if path != "" {
			childPath = path + "/" + task.Name
		}
		resolved, err := g.analyze(child, task, childPath, changedInputs, forced, ancestors)
		if err != nil {
			return err
		}
		resolved.replaceLoop = replaceLoop
		n.children[name] = resolved
		n.affected = n.affected || resolved.affected
		delete(visiting, name)
		return nil
	}
	for _, task := range declared {
		if err := visit(task.Name); err != nil {
			return nil, err
		}
	}
	changedTasks := []string{}
	for name, c := range n.children {
		if c.affected {
			changedTasks = append(changedTasks, name)
		}
	}
	// Changing upstream output-driven loop identities/conditions cannot be
	// proven reusable by an unchanged WorkflowSpec alone.
	for _, task := range declared {
		check := task.DeepCopy()
		check.Arguments = wfv1.Arguments{}
		if n.children[task.Name].replaceLoop {
			check.WithParam = ""
			check.WithItems = nil
			check.WithSequence = nil
		}
		refs, err := p.References(check)
		if err != nil {
			return nil, err
		}
		if refs.Uses("tasks", changedTasks) || refs.Uses("steps", changedTasks) {
			return nil, fmt.Errorf("changed task output affects a condition or loop identity at %s/%s", path, task.Name)
		}
	}
	return n, nil
}

func hasRetryLoop(t wfv1.DAGTask) bool {
	return t.WithParam != "" || len(t.WithItems) > 0 || t.WithSequence != nil
}
func (g *retryGraph) affectedLeaves() []string {
	out := []string{}
	for path, n := range g.leaves {
		if n.affected {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func (g *retryGraph) matchNodes(wf *wfv1.Workflow) (map[string]bool, map[string]string, error) {
	root, ok := wf.Status.Nodes[wf.Name]
	if !ok || (root.Type != wfv1.NodeTypeDAG && root.Type != wfv1.NodeTypeSteps) || root.BoundaryID != "" {
		return nil, nil, fmt.Errorf("missing native DAG boundary")
	}
	boundaries := map[string]*retryInvocation{wf.Name: g.root}
	paths := map[string]string{wf.Name: ""}
	selected := map[string]bool{}
	retained := map[string]string{}
	visited := map[string]bool{wf.Name: true}
	attemptParents, err := retryAttemptParents(wf.Status.Nodes)
	if err != nil {
		return nil, nil, err
	}
	retryCalls := map[string]*retryInvocation{}
	for len(visited) < len(wf.Status.Nodes) {
		progress := false
		for id, node := range wf.Status.Nodes {
			if visited[id] {
				continue
			}
			if retryID, attempt := attemptParents[id]; attempt {
				call, known := retryCalls[retryID]
				if !known {
					continue
				}
				if node.Phase != wfv1.NodeSucceeded && node.Phase != wfv1.NodeFailed && node.Phase != wfv1.NodeError {
					return nil, nil, fmt.Errorf("retry attempt is not terminal")
				}
				if call.affected {
					selected[id] = true
				} else {
					// A completed retry group includes its earlier failed attempts.
					// Preserve that history, not just the final successful Pod.
					retained[id] = p.ResultFingerprint(node)
				}
				visited[id], progress = true, true
				continue
			}
			parent, ok := boundaries[node.BoundaryID]
			if !ok {
				continue
			}
			boundary := wf.Status.Nodes[node.BoundaryID]
			if node.Type == wfv1.NodeTypeStepGroup && len(parent.template.Steps) > 0 {
				index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(node.Name, boundary.Name+"["), "]"))
				if err != nil || index < 0 || index >= len(parent.template.Steps) || node.Name != fmt.Sprintf("%s[%d]", boundary.Name, index) {
					return nil, nil, fmt.Errorf("invalid native step group identity")
				}
				affected := false
				for name, group := range parent.stepGroups {
					if group == index && parent.children[name].affected {
						affected = true
					}
				}
				if !affected {
					if node.Phase != wfv1.NodeSucceeded {
						return nil, nil, fmt.Errorf("unrelated step group is not complete")
					}
					retained[id] = p.ResultFingerprint(node)
				}
				visited[id] = true
				progress = true
				continue
			}
			var call *retryInvocation
			path := ""
			for name, c := range parent.children {
				expected := boundary.Name + "." + name
				if group, ok := parent.stepGroups[name]; ok {
					expected = fmt.Sprintf("%s[%d].%s", boundary.Name, group, name)
				}
				match := node.Name == expected || (hasRetryLoop(c.task) && strings.HasPrefix(node.Name, expected+"(") && strings.HasSuffix(node.Name, ")"))
				if match && node.TemplateName == c.task.Template {
					if call != nil {
						return nil, nil, fmt.Errorf("ambiguous native invocation identity")
					}
					call = c
					path = name
					if paths[node.BoundaryID] != "" {
						path = paths[node.BoundaryID] + "/" + name
					}
				}
			}
			if call == nil {
				return nil, nil, fmt.Errorf("node %s has unsupported invocation identity", id)
			}
			if node.Type == wfv1.NodeTypeSkipped && node.Phase == wfv1.NodeOmitted {
				// An omitted invocation never ran. It is not a reusable result or
				// an approved decision, even with an ordering-only dependency.
				// Let native Argo evaluate it again after the selected retry.
				var mark func(*retryInvocation)
				mark = func(n *retryInvocation) {
					n.affected = true
					for _, child := range n.children {
						mark(child)
					}
				}
				mark(call)
				selected[id], visited[id], progress = true, true, true
				continue
			}
			if call.replaceLoop {
				selected[id] = true
			}
			switch node.Type {
			case wfv1.NodeTypeRetry:
				if call.template.RetryStrategy == nil || len(node.Children) == 0 {
					return nil, nil, fmt.Errorf("retry group has no declared strategy or observed attempts")
				}
				if call.affected {
					selected[id] = true
				} else if call.template.Metadata.Annotations[p.Group+"/result-reuse"] != "same-inputs" {
					return nil, nil, fmt.Errorf("retry group %s has no explicit same-inputs result reuse contract", path)
				}
				retryCalls[id] = call
			case wfv1.NodeTypeDAG, wfv1.NodeTypeSteps:
				if hasRetryLoop(call.task) && !strings.HasSuffix(node.Name, ")") {
					return nil, nil, fmt.Errorf("loop boundary lacks iteration identity")
				}
				if (node.Type == wfv1.NodeTypeDAG && call.template.DAG == nil) || (node.Type == wfv1.NodeTypeSteps && len(call.template.Steps) == 0) {
					return nil, nil, fmt.Errorf("unexpected DAG boundary")
				}
				boundaries[id] = call
				paths[id] = path
			case wfv1.NodeTypeTaskGroup:
				if !hasRetryLoop(call.task) || strings.HasSuffix(node.Name, ")") {
					return nil, nil, fmt.Errorf("unexpected task group")
				}
			default:
				if ((call.template.DAG != nil || len(call.template.Steps) > 0) && (!call.replaceLoop || node.Type != wfv1.NodeTypeSkipped)) || (node.Type != wfv1.NodeTypePod && node.Type != wfv1.NodeTypeSuspend && node.Type != wfv1.NodeTypeHTTP && node.Type != wfv1.NodeTypePlugin && node.Type != wfv1.NodeTypeSkipped) {
					return nil, nil, fmt.Errorf("unsupported retry node type at %s", path)
				}
				if call.affected {
					if call.template.Suspend == nil && call.template.Metadata.Annotations[p.Group+"/retry-safe"] != "idempotent" {
						return nil, nil, fmt.Errorf("task %s must explicitly declare retry-safe=idempotent", path)
					}
					selected[id] = true
				} else if call.template.Suspend == nil && call.template.Metadata.Annotations[p.Group+"/result-reuse"] != "same-inputs" {
					return nil, nil, fmt.Errorf("task %s has no explicit same-inputs result reuse contract", path)
				}
			}
			if !call.affected {
				if node.Phase != wfv1.NodeSucceeded && node.Phase != wfv1.NodeSkipped &&
					(len(g.selected) == 0 || g.required[call] || (node.Phase != wfv1.NodeFailed && node.Phase != wfv1.NodeError)) {
					return nil, nil, fmt.Errorf("unrelated task %s is not successfully completed", path)
				}
				// Group outputs and completion evidence also belong to the preserved result.
				retained[id] = p.ResultFingerprint(node)
			}
			visited[id] = true
			progress = true
		}
		if !progress {
			return nil, nil, fmt.Errorf("unresolved native invocation boundaries")
		}
	}
	return selected, retained, nil
}

// Steps order remains in the native spec. Only output consumers contribute to
// the parameter impact analysis; sequence ordering alone does not invalidate
// a successful independent call or approval.
func retryCalls(t wfv1.Template) ([]wfv1.DAGTask, map[string]int, error) {
	if t.DAG != nil {
		return t.DAG.Tasks, nil, nil
	}
	var tasks []wfv1.DAGTask
	groups := map[string]int{}
	for i, group := range t.Steps {
		for _, step := range group.Steps {
			raw, err := json.Marshal(step)
			if err != nil {
				return nil, nil, err
			}
			var call wfv1.DAGTask
			if err = json.Unmarshal(raw, &call); err != nil {
				return nil, nil, err
			}
			refs, err := p.References(step.Arguments)
			if err != nil {
				return nil, nil, err
			}
			for name, index := range groups {
				if index < i && refs.Uses("steps", []string{name}) {
					call.Dependencies = append(call.Dependencies, name)
				}
			}
			sort.Strings(call.Dependencies)
			if _, exists := groups[step.Name]; exists {
				return nil, nil, fmt.Errorf("duplicate step identity")
			}
			groups[step.Name] = i
			tasks = append(tasks, call)
		}
	}
	return tasks, groups, nil
}
func clearRetryArguments(t *wfv1.Template) {
	// Defaults/values are input sources, just like explicit call arguments.
	// ValueFrom selectors and all structural consumers remain checked.
	for i := range t.Inputs.Parameters {
		t.Inputs.Parameters[i].Default = nil
		t.Inputs.Parameters[i].Value = nil
	}
	if t.DAG != nil {
		for i := range t.DAG.Tasks {
			t.DAG.Tasks[i].Arguments = wfv1.Arguments{}
			if allowsLoopReplacement(*t, t.DAG.Tasks[i].Name) {
				t.DAG.Tasks[i].WithParam = ""
				t.DAG.Tasks[i].WithItems = nil
				t.DAG.Tasks[i].WithSequence = nil
			}
		}
	}
	for i := range t.Steps {
		for j := range t.Steps[i].Steps {
			t.Steps[i].Steps[j].Arguments = wfv1.Arguments{}
			if allowsLoopReplacement(*t, t.Steps[i].Steps[j].Name) {
				t.Steps[i].Steps[j].WithParam = ""
				t.Steps[i].Steps[j].WithItems = nil
				t.Steps[i].Steps[j].WithSequence = nil
			}
		}
	}
}

// Authors approve replacing the entire expansion, never matching changed items
// against old positional results. Ordinary loop retries remain conservative.
func allowsLoopReplacement(t wfv1.Template, name string) bool {
	var calls []string
	if json.Unmarshal([]byte(t.Metadata.Annotations[p.Group+"/retry-loop-calls"]), &calls) != nil {
		return false
	}
	return slices.Contains(calls, name)
}

// Failed dependencies cannot be retained as if they were successful. Only a
// terminal branch independent of the selected downstream closure may stay failed.
func (g *retryGraph) markRequired(parent *retryInvocation) {
	if parent == nil {
		return
	}
	var markTree func(*retryInvocation)
	markTree = func(n *retryInvocation) {
		if !n.affected {
			g.required[n] = true
		}
		for _, c := range n.children {
			markTree(c)
		}
	}
	var markAncestors func(string, map[string]bool)
	markAncestors = func(name string, visited map[string]bool) {
		if visited[name] {
			return
		}
		visited[name] = true
		n := parent.children[name]
		if n == nil {
			return
		}
		for _, dep := range n.task.Dependencies {
			markTree(parent.children[dep])
			markAncestors(dep, visited)
		}
		if group, ok := parent.stepGroups[name]; ok {
			for prior, index := range parent.stepGroups {
				if index < group {
					markTree(parent.children[prior])
				}
			}
		}
	}
	for name, n := range parent.children {
		if n.affected {
			markAncestors(name, map[string]bool{})
		}
		g.markRequired(n)
	}
}
