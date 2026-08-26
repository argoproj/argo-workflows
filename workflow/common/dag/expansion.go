package dag

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
)

// ExpandTask expands a single DAG task containing withItems, withParams, withSequence into multiple parallel tasks
// We want to be lazy with expanding. Unfortunately this is not quite possible as the When field might rely on
// expansion to work with the shouldExecute function. To address this we apply a trick, we try to expand, if we fail, we then
// check shouldExecute, if shouldExecute returns false, we continue on as normal else error out
func ExpandTask(ctx context.Context, task wfv1.DAGTask, scope map[string]string, substitutor Substitutor) ([]wfv1.DAGTask, error) {
	var err error
	var items []wfv1.Item
	switch {
	case len(task.WithItems) > 0:
		items = task.WithItems
	case task.WithParam != "":
		resolvedParam, resolveErr := resolveWithParam(task.WithParam, scope, substitutor)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if err = json.Unmarshal([]byte(resolvedParam), &items); err != nil {
			mustExec, mustExecErr := shouldExecute(task.When)
			if mustExecErr != nil || mustExec {
				return nil, errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s: %v", strings.TrimSpace(resolvedParam), err)
			}
		}
	case task.WithSequence != nil:
		seq := task.WithSequence.DeepCopy()
		if substitutor != nil {
			resolveIntOrString := func(val *intstr.IntOrString) (*intstr.IntOrString, error) {
				if val == nil || val.Type == intstr.Int {
					return val, nil
				}
				resolved, subErr := substitutor.Substitute(val.String(), scope)
				if subErr != nil {
					return val, subErr
				}
				return &intstr.IntOrString{Type: intstr.String, StrVal: resolved}, nil
			}
			if seq.Count, err = resolveIntOrString(seq.Count); err != nil {
				return nil, err
			}
			if seq.Start, err = resolveIntOrString(seq.Start); err != nil {
				return nil, err
			}
			if seq.End, err = resolveIntOrString(seq.End); err != nil {
				return nil, err
			}
		}
		items, err = expandSequence(seq)
		if err != nil {
			mustExec, mustExecErr := shouldExecute(task.When)
			if mustExecErr != nil || mustExec {
				return nil, err
			}
		}
	default:
		return []wfv1.DAGTask{task}, nil
	}

	// these fields can be very large (>100m) and marshalling 10k x 100m = 6GB of memory used and
	// very poor performance, so we just nil them out
	task.WithItems = nil
	task.WithParam = ""
	task.WithSequence = nil

	taskBytes, err := json.Marshal(task)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	expandedTasks := make([]wfv1.DAGTask, 0)
	for i, item := range items {
		var newTask wfv1.DAGTask
		newTaskName, err := processItem(ctx, taskBytes, task.Name, i, item, &newTask, scope, substitutor)
		if err != nil {
			return nil, err
		}
		if newTaskName == "" {
			continue
		}
		newTask.Name = newTaskName
		newTask.Template = task.Template
		expandedTasks = append(expandedTasks, newTask)
	}
	return expandedTasks, nil
}

// resolveWithParam resolves template references in a withParam value.
// The substitutor's Substitute method escapes replacement values for safe JSON embedding
// (via strconv.Quote). When withParam is a raw template like "{{steps.X.outputs.result}}",
// direct substitution would produce escaped JSON (e.g., [{\"key\":\"val\"}]).
// To get the raw value, we wrap the withParam in a JSON string context, substitute
// (where escaping is correct), then extract via JSON unmarshal (which reverses the escaping).
// This matches the old resolveDependencyReferences approach that marshalled the entire task
// to JSON before substitution.
func resolveWithParam(withParam string, scope map[string]string, substitutor Substitutor) (string, error) {
	if substitutor == nil {
		return withParam, nil
	}
	// Wrap in a JSON object: {"v":"<withParam>"} so the substitutor's escaping
	// is appropriate for the JSON string context.
	jsonWrapped := `{"v":` + strconv.Quote(withParam) + `}`
	resolved, err := substitutor.Substitute(jsonWrapped, scope)
	if err != nil {
		return "", err
	}
	var wrapper struct {
		V string `json:"v"`
	}
	if err := json.Unmarshal([]byte(resolved), &wrapper); err != nil {
		return "", fmt.Errorf("failed to resolve withParam template %q: %w", withParam, err)
	}
	return wrapper.V, nil
}

func (e *DAGEvaluator) ExpandTask(ctx context.Context, task wfv1.DAGTask, scope map[string]string, substitutor Substitutor) ([]wfv1.DAGTask, error) {
	return ExpandTask(ctx, task, scope, substitutor)
}

// shouldExecute evaluates a when expression that has NOT been through variable substitution.
// Uses argoexpr (not govaluate) because at expansion time, variables haven't been substituted yet.
// Post-substitution evaluation happens in engine.go's evaluateWhenClause using govaluate,
// which handles unquoted string comparisons (e.g., "odd == even").
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	return argoexpr.EvalBool(when, nil)
}

func expandSequence(seq *wfv1.Sequence) ([]wfv1.Item, error) {
	if seq == nil {
		return nil, nil
	}

	var start, end, count int64
	var err error

	if seq.Start != nil {
		if seq.Start.Type == intstr.Int {
			start = int64(seq.Start.IntValue())
		} else {
			start, err = strconv.ParseInt(seq.Start.String(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sequence start: %w", err)
			}
		}
	}

	if seq.Count != nil {
		if seq.Count.Type == intstr.Int {
			count = int64(seq.Count.IntValue())
		} else {
			count, err = strconv.ParseInt(seq.Count.String(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sequence count: %w", err)
			}
		}
	}

	switch {
	case seq.End != nil:
		if seq.End.Type == intstr.Int {
			end = int64(seq.End.IntValue())
		} else {
			end, err = strconv.ParseInt(seq.End.String(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sequence end: %w", err)
			}
		}
	case seq.Count != nil:
		end = start + count - 1
	default:
		return nil, nil
	}

	// Determine step direction: forward (start <= end) or backward (start > end)
	step := int64(1)
	if start > end {
		step = -1
	}

	// When both end and count are specified, count limits the number of items
	numElements := abs64(end-start) + 1
	if seq.Count != nil && count < numElements {
		numElements = count
	}

	format := "%d"
	if seq.Format != "" {
		format = seq.Format
	}

	var items []wfv1.Item
	for i := int64(0); i < numElements; i++ {
		val := start + i*step
		// Always produce JSON string items (matching old ParseItem(`"..."`) behavior).
		strVal := fmt.Sprintf(format, val)
		raw, err := json.Marshal(strVal)
		if err != nil {
			return nil, err
		}
		items = append(items, wfv1.Item{
			Value: raw,
		})
	}

	return items, nil
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func processItem(_ context.Context, taskBytes []byte, taskName string, i int, item wfv1.Item, newTask *wfv1.DAGTask, globalScope map[string]string, substitutor Substitutor) (string, error) {
	var newTaskName string

	err := json.Unmarshal(taskBytes, newTask)
	if err != nil {
		return "", errors.InternalWrapError(err)
	}

	if substitutor != nil {
		substScope := make(map[string]string)
		maps.Copy(substScope, globalScope)
		// Item values are formatted through wfv1.Item exactly as the pre-Engine
		// controller did: normalised JSON for maps and lists (not the raw text
		// the user wrote), so substituted values are stable across whitespace
		// and key order.
		switch item.GetType() {
		case wfv1.String:
			substScope["item"] = item.GetStrVal()
		case wfv1.Map:
			mapVal := item.GetMapVal()
			for k, v := range mapVal {
				substScope["item."+k] = v.String()
			}
			b, err := json.Marshal(mapVal)
			if err != nil {
				return "", errors.InternalWrapError(err)
			}
			substScope["item"] = string(b)
		case wfv1.List:
			b, err := json.Marshal(item.GetListVal())
			if err != nil {
				return "", errors.InternalWrapError(err)
			}
			substScope["item"] = string(b)
		default: // Number, Bool
			substScope["item"] = item.String()
		}
		substScope["index"] = strconv.Itoa(i) // Marshal the new task, substitute, and unmarshal back
		taskJSON, err := json.Marshal(newTask)
		if err != nil {
			return "", errors.InternalWrapError(err)
		}
		substituted, err := substitutor.Substitute(string(taskJSON), substScope)
		if err != nil {
			return "", err
		}
		err = json.Unmarshal([]byte(substituted), newTask)
		if err != nil {
			return "", errors.InternalWrapError(err)
		}
	}

	if newTask.Name != "" && newTask.Name != taskName {
		newTaskName = newTask.Name
	} else {
		// Name text is formatted through wfv1.Item exactly as the pre-Engine
		// controller did, so expanded node names (and hence node IDs) are
		// unchanged: maps as sorted "key:value" pairs, lists as "[a b c]".
		var itemText string
		switch item.GetType() {
		case wfv1.Map:
			mapVal := item.GetMapVal()
			vals := make([]string, 0, len(mapVal))
			for k, v := range mapVal {
				vals = append(vals, fmt.Sprintf("%s:%v", k, v))
			}
			sort.Strings(vals)
			itemText = strings.Join(vals, ",")
		case wfv1.List:
			itemText = fmt.Sprint(item.GetListVal())
		default:
			itemText = item.String()
		}
		if item.Value != nil {
			newTaskName, err = expandedTaskName(taskName, i, itemText)
			if err != nil {
				return "", err
			}
		} else {
			newTaskName = fmt.Sprintf("%s(%d)", taskName, i)
		}
	}

	// The 'when' clause (now substituted with item values) is preserved on the expanded task.
	// Evaluation is deferred to the engine's createDesiredTask, which uses the legacy govaluate
	// evaluator that correctly handles unquoted string comparisons (e.g., "odd == even").

	return newTaskName, nil
}

// expandedTaskName names the index-th expansion of taskName for the given
// item text: "task(index:item)". Parentheses are stripped from the item text
// so the index can always be recovered from the name, which consumers such as
// `argo get` rely on (util.RecoverIndexFromNodeName); that is verified here.
func expandedTaskName(taskName string, index int, itemText string) (string, error) {
	replacer := strings.NewReplacer("(", "", ")", "")
	name := fmt.Sprintf("%s(%d:%s)", taskName, index, replacer.Replace(itemText))
	if got := util.RecoverIndexFromNodeName(name); got != index {
		return "", fmt.Errorf("expanded task name %q does not encode index %d (recovered %d)", name, index, got)
	}
	return name, nil
}
