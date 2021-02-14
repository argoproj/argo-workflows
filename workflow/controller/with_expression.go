package controller

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) expandWithExpression(expression string, localParams common.Parameters) ([]wfv1.Item, error) {
	result, err := common.EvalWithExpression(expression, woc.globalParams.Merge(localParams))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate with parameter expression %q: %w", expression, err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall with parameter result %q: %w", result, err)
	}

	log.WithFields(log.Fields{
		"expression": expression,
		"result":     result,
	}).Debug("expand with param expression result")

	items := make([]wfv1.Item, 0)
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to unmarshall with parameter result into items %q: %w", result, err)
	}

	log.WithFields(log.Fields{
		"expression": expression,
		"result":     result,
		"items":      items,
	}).Debug("expand with param expression items")

	return items, nil
}
