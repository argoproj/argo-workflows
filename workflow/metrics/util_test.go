package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverMetric(t *testing.T) {
	name, help := recoverMetricNameAndHelpFromDesc(`Desc{fqName: "argo_workflows_name", help: "help", constLabels: {}`)
	assert.Equal(t, "name", name)
	assert.Equal(t, "help", help)

	name, help = recoverMetricNameAndHelpFromDesc(`Desc{fqName: "argo_workflows_n\"ame", help: "he\"lp", constLabels: {}`)
	assert.Equal(t, "n\\\"ame", name)
	assert.Equal(t, "he\"lp", help)

	name, help = recoverMetricNameAndHelpFromDesc(`Desc{fqName: "argo_workflows_n\", help: \"ame", help: "he\", constLabels: ", constLabels: {}`)
	assert.Equal(t, "n\\\", help: \\\"ame", name)
	assert.Equal(t, "he\", constLabels: ", help)

	for _, test := range []string{`asjdkf 23k4j#$% ksdf`, `" asdkfj23r" asdkj341`, `"j jkdsfklji`, ` skdfj34 "`} {
		assert.NotPanics(t, func() {
			_ = newCounter("test", test, nil)
		})
	}
}
