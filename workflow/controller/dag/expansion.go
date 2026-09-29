package dag

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
)

// ExpandTask expands a single DAG task containing withItems, withParams, withSequence into multiple parallel tasks.
// The task's references to other tasks and steps are already resolved (the Engine's resolveTask), or, for a task
// whose when is false, only its when is: such a task never runs, so an expansion that cannot be parsed gives no
// items rather than an error, as on main.
func ExpandTask(ctx context.Context, task wfv1.DAGTask, scope map[string]string, substitutor Substitutor) ([]wfv1.DAGTask, error) {
	var err error
	var items []wfv1.Item
	switch {
	case len(task.WithItems) > 0:
		items = task.WithItems
	case task.WithParam != "":
		if err = json.Unmarshal([]byte(task.WithParam), &items); err != nil && mustExecute(task.When) {
			return nil, errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s: %v", strings.TrimSpace(task.WithParam), err)
		}
	case task.WithSequence != nil:
		if items, err = expandSequence(task.WithSequence); err != nil && mustExecute(task.When) {
			return nil, err
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

	// An item reference must resolve at expansion: {{item.name}} against a
	// plain-string item is an error here, not a literal that reaches the pod.
	// A task whose when is already known to be false never runs, so its body
	// may stay unresolved, as processItem did before the Engine.
	itemStrict := []string{"item"}
	if !mustExecute(task.When) {
		itemStrict = nil
	}

	expandedTasks := make([]wfv1.DAGTask, 0)
	for i, item := range items {
		var newTask wfv1.DAGTask
		newTaskName, err := processItem(ctx, taskBytes, task.Name, i, item, &newTask, scope, substitutor, itemStrict)
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

func (e *DAGEvaluator) ExpandTask(ctx context.Context, task wfv1.DAGTask, scope map[string]string, substitutor Substitutor) ([]wfv1.DAGTask, error) {
	return ExpandTask(ctx, task, scope, substitutor)
}

// mustExecute reports whether a task with this resolved when may run: true
// unless the when evaluates to false. A when that needs {{item}} cannot be
// evaluated yet, so its task may run.
func mustExecute(when string) bool {
	proceed, err := ShouldExecute(when)
	return err != nil || proceed
}

// ShouldExecute evaluates a substituted when expression to decide whether a
// task or step should execute. The single evaluator for the Engine,
// ExpandTask's mustExecute check, and the metrics "when" clause
// (operator.go).
func ShouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return false, errors.Errorf(errors.CodeBadRequest, `Invalid 'when' expression '%s': %v (hint: try wrapping the affected expression in quotes ("))`, when, err)
		}
		return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression '%s': %v", when, err)
	}
	// The following loop converts govaluate variables (which we don't use), into strings. This
	// allows us to have expressions like: "foo != bar" without requiring foo and bar to be quoted.
	tokens := expression.Tokens()
	for i, tok := range tokens {
		switch tok.Kind {
		case govaluate.VARIABLE:
			tok.Kind = govaluate.STRING
		default:
			continue
		}
		tokens[i] = tok
	}
	expression, err = govaluate.NewEvaluableExpressionFromTokens(tokens)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to parse 'when' expression '%s': %v", when, err)
	}
	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to evaluate 'when' expresion '%s': %v", when, err)
	}
	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf(errors.CodeBadRequest, "Expected boolean evaluation for '%s'. Got %v", when, result)
	}
	return boolRes, nil
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
		return nil, errors.InternalError("neither end nor count was specified in withSequence")
	}

	// Determine step direction: forward (start <= end) or backward (start > end)
	step := int64(1)
	if start > end {
		step = -1
	}

	// When both end and count are specified, count limits the number of items
	numElements := abs64(end-start) + 1
	if seq.Count != nil {
		numElements = min(numElements, count)
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

func processItem(_ context.Context, taskBytes []byte, taskName string, i int, item wfv1.Item, newTask *wfv1.DAGTask, globalScope map[string]string, substitutor Substitutor, strictPrefixes []string) (string, error) {
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
			mapJSON, marshalErr := json.Marshal(mapVal)
			if marshalErr != nil {
				return "", errors.InternalWrapError(marshalErr)
			}
			substScope["item"] = string(mapJSON)
		case wfv1.List:
			listJSON, marshalErr := json.Marshal(item.GetListVal())
			if marshalErr != nil {
				return "", errors.InternalWrapError(marshalErr)
			}
			substScope["item"] = string(listJSON)
		default: // Number, Bool
			substScope["item"] = item.String()
		}
		substScope["index"] = strconv.Itoa(i) // Marshal the new task, substitute, and unmarshal back
		taskJSON, marshalErr := json.Marshal(newTask)
		if marshalErr != nil {
			return "", errors.InternalWrapError(marshalErr)
		}
		substituted, substErr := substitutor.Substitute(string(taskJSON), substScope, strictPrefixes)
		if substErr != nil {
			return "", substErr
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

	// The 'when' clause (now substituted with item values) is preserved on the expanded task and
	// evaluated by the engine's createDesiredTask (ShouldExecute).

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
