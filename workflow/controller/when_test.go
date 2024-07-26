package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldExecute(t *testing.T) {
	trueExpressions := []string{
		"foo == foo",
		"'ref/branch/master' == 'ref/branch/master'",
		"foo != bar",
		"1 == 1",
		"1 != 2",
		"1 < 2",
		"1 <= 1",
		"1/2 == 0.5",
		"a < b",
		"(foo == bar) || (foo == foo)",
		"(1 > 0) && (1 < 2)",
		"Error in (Failed, Error)",
		"!(Succeeded in (Failed, Error))",
		"true == true",
	}
	for _, trueExp := range trueExpressions {
		res, err := shouldExecute(trueExp)
		require.NoError(t, err)
		assert.True(t, res)
	}

	falseExpressions := []string{
		"foo != foo",
		"'ref/branch/master' != 'ref/branch/master'",
		"foo == bar",
		"1 != 1",
		"1 == 2",
		"1 > 2",
		"1 <= 0",
		"1/2 != 0.5",
		"a > b",
		"(foo == bar) || (bar == foo)",
		"(1 > 0) && (11 < 2)",
		"Succeeded in (Failed, Error)",
		"!(Error in (Failed, Error))",
		"false == true",
	}
	for _, falseExp := range falseExpressions {
		res, err := shouldExecute(falseExp)
		require.NoError(t, err)
		assert.False(t, res)
	}
}
