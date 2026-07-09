package controller

import (
	"fmt"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

func (woc *wfOperationCtx) prepareDefaultMetricScope() (map[string]any, map[string]func() float64) {
	localScope := template.ToAnyMap(woc.globalParams())
	localScope[varkeys.MetricDuration.Template()] = "0"
	localScope[varkeys.MetricStatus.Template()] = string(wfv1.NodePending)
	localScope[varkeys.MetricResourcesDurationByName.Concretize(string(v1.ResourceCPU))] = "0"
	localScope[varkeys.MetricResourcesDurationByName.Concretize(string(v1.ResourceMemory))] = "0"

	var realTimeScope = map[string]func() float64{
		varkeys.WorkflowDuration.Template(): woc.workflowDurationSeconds,
	}

	return localScope, realTimeScope
}

func (woc *wfOperationCtx) prepareMetricScope(node *wfv1.NodeStatus) (map[string]any, map[string]func() float64) {
	localScope, realTimeScope := woc.prepareDefaultMetricScope()
	if node.Fulfilled() {
		localScope[varkeys.MetricDuration.Template()] = fmt.Sprintf("%f", node.FinishedAt.Sub(node.StartedAt.Time).Seconds())
		realTimeScope[varkeys.MetricDuration.Template()] = func() float64 {
			return node.FinishedAt.Sub(node.StartedAt.Time).Seconds()
		}
	} else {
		localScope[varkeys.MetricDuration.Template()] = fmt.Sprintf("%f", time.Since(node.StartedAt.Time).Seconds())
		realTimeScope[varkeys.MetricDuration.Template()] = func() float64 {
			return time.Since(node.StartedAt.Time).Seconds()
		}
	}

	if len(node.Children) != 0 {
		localScope[varkeys.Retries.Template()] = strconv.Itoa(len(node.Children) - 1)
	}

	if node.Phase != "" {
		localScope[varkeys.MetricStatus.Template()] = string(node.Phase)
	}

	if node.Inputs != nil {
		for _, param := range node.Inputs.Parameters {
			key := varkeys.InputsParameterByName.Concretize(param.Name)
			if param.Value == nil {
				localScope[key] = ""
			} else {
				localScope[key] = param.Value.String()
			}
		}
	}

	if node.Outputs != nil {
		if node.Outputs.Result != nil {
			localScope[varkeys.MetricOutputsResult.Template()] = *node.Outputs.Result
		}
		if node.Outputs.ExitCode != nil {
			localScope[varkeys.MetricExitCode.Template()] = *node.Outputs.ExitCode
		}
		for _, param := range node.Outputs.Parameters {
			key := varkeys.MetricOutputsParameterByName.Concretize(param.Name)
			if param.Value == nil {
				localScope[key] = ""
			} else {
				localScope[key] = param.Value.String()
			}
		}
	}

	if node.ResourcesDuration != nil {
		for name, duration := range node.ResourcesDuration {
			localScope[varkeys.MetricResourcesDurationByName.Concretize(string(name))] = fmt.Sprint(duration.Duration().Seconds())
		}
	}

	return localScope, realTimeScope
}
