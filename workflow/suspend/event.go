package suspend

import (
	"strconv"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func IncrementWaitCount(wf *wfv1.Workflow) {
	count, _ := strconv.Atoi(wf.GetLabels()[common.LabelKeyEventWait])
	wf.GetLabels()[common.LabelKeyEventWait] = strconv.Itoa(count + 1)
}

func DecrementEventWait(wf *wfv1.Workflow) {
	count, _ := strconv.Atoi(wf.GetLabels()[common.LabelKeyEventWait])
	if count > 1 {
		wf.GetLabels()[common.LabelKeyEventWait] = strconv.Itoa(count - 1)
	} else {
		delete(wf.GetLabels(), common.LabelKeyEventWait)
	}
}
