package suspend

import (
	"strconv"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func IncrementEventWaitCount(wf *wfv1.Workflow) {
	count, _ := strconv.Atoi(wf.GetLabels()[common.LabelKeyEventWaitCount])
	wf.GetLabels()[common.LabelKeyEventWaitCount] = strconv.Itoa(count + 1)
}

func DecrementEventWaitCount(wf *wfv1.Workflow) {
	count, _ := strconv.Atoi(wf.GetLabels()[common.LabelKeyEventWaitCount])
	if count > 1 {
		wf.GetLabels()[common.LabelKeyEventWaitCount] = strconv.Itoa(count - 1)
	} else {
		delete(wf.GetLabels(), common.LabelKeyEventWaitCount)
	}
}
