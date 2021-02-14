package common

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/antonmedv/expr"
	"github.com/doublerebel/bellows"
	log "github.com/sirupsen/logrus"
)

func EvalWithExpression(expression string, params Parameters) (interface{}, error) {
	replaceMap := make(map[string]interface{})
	for k, v := range params {
		replaceMap[k] = v
	}

	log.WithFields(log.Fields{
		"expression": expression,
		"params":     params,
	}).Debug("Evaluating parameter expression")

	var castErr error // stash any casting error here
	replaceMap["int"] = func(s string) int {
		v, err := strconv.Atoi(s)
		if err != nil {
			castErr = fmt.Errorf("failed to cast %q to int: %w", s, err)
		}
		return v
	}
	replaceMap["array"] = func(s string) []interface{} {
		array := make([]interface{}, 0)
		if err := json.Unmarshal([]byte(s), &array); err != nil {
			castErr = fmt.Errorf("failed to cast %q to array: %w", s, err)
		}
		return array
	}

	result, err := expr.Eval(expression, bellows.Expand(replaceMap))
	if err == nil {
		err = castErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression %q: %w", expression, err)
	}
	return result, nil
}
