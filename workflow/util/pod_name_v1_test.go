package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPodNameV1(t *testing.T) {
	nodeName := "nodename"
	nodeID := "1"

	// short case
	shortWfName := "wfname"
	shortTemplateName := "templatename"

	expected := fmt.Sprintf("%s-%s", shortWfName, shortTemplateName)
	actual := ensurePodNamePrefixLength(expected)
	assert.Equal(t, expected, actual)

	name := GeneratePodName(shortWfName, nodeName, shortTemplateName, nodeID, PodNameV1)
	assert.Equal(t, nodeID, name)

	// long case
	longWfName := "alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever"
	longTemplateName := "alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation"

	sum := len(longWfName) + len(longTemplateName)
	assert.Greater(t, sum, maxK8sResourceNameLength-k8sNamingHashLength)

	expected = fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	actual = ensurePodNamePrefixLength(expected)

	assert.Equal(t, maxK8sResourceNameLength-k8sNamingHashLength-1, len(actual))

	name = GeneratePodName(longWfName, nodeName, longTemplateName, nodeID, PodNameV1)
	assert.Equal(t, nodeID, name)

}
