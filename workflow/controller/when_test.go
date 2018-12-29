package controller

import (
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestShouldExecute(t *testing.T) {
	//res, err := shouldExecute("JSONPath(\"{\"success\": \"true\"}\", \".success\") == true")
	res, err := shouldExecute(`JSONPath("{\"success\": true}", ".success") == true`)
	println(err)
	println(res)

	trueExpressions := []string{
		"foo == foo",
		"foo != bar",
		"1 == 1",
		"1 != 2",
		"1 < 2",
		"1 <= 1",
		"a < b",
		"(foo == bar) || (foo == foo)",
		"(1 > 0) && (1 < 2)",
		"Error in (Failed, Error)",
		"!(Succeeded in (Failed, Error))",
		"true == true",
	}
	for _, trueExp := range trueExpressions {
		res, err := shouldExecute(trueExp)
		assert.Nil(t, err)
		assert.True(t, res)
	}

	falseExpressions := []string{
		"foo != foo",
		"foo == bar",
		"1 != 1",
		"1 == 2",
		"1 > 2",
		"1 <= 0",
		"a > b",
		"(foo == bar) || (bar == foo)",
		"(1 > 0) && (11 < 2)",
		"Succeeded in (Failed, Error)",
		"!(Error in (Failed, Error))",
		"false == true",
	}
	for _, falseExp := range falseExpressions {
		res, err := shouldExecute(falseExp)
		assert.Nil(t, err)
		assert.False(t, res)
	}
}

type whenExp struct {
	when     string
	bindings map[string]string
}

func TestShouldExecuteWithBindingsAndFunctions(t *testing.T) {
	trueExpressions := []whenExp{
		{when: "JSONPath(param, '.foo') == bar", bindings: map[string]string{"param": `{"foo": "bar"}`}},
		{when: "bar in JSONPath(param, '.foo')", bindings: map[string]string{"param": `{"foo": ["bar"]}`}},
	}
	for _, trueExp := range trueExpressions {
		bindings := make([]v1alpha1.WhenBinding, 0)
		for p, v := range trueExp.bindings {
			bindings = append(bindings, v1alpha1.WhenBinding{Name: p, Value: v})
		}
		res, err := shouldExecute(trueExp.when, bindings...)
		assert.Nil(t, err)
		assert.True(t, res)
	}
}
