package util

import (
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPodName(t *testing.T) {
	nodeName := "nodename"
	nodeID := "1"

	///////////////////////////////////////////////////////////////////////////////////////////
	// POD_NAMES v1:
	///////////////////////////////////////////////////////////////////////////////////////////
	// short case
	shortWfName := "wfname"
	shortTemplateName := "templatename"

	expected := fmt.Sprintf("%s-%s", shortWfName, shortTemplateName)
	actual := ensurePodNamePrefixLength(expected)
	assert.Equal(t, expected, actual)

	name := PodName(shortWfName, nodeName, shortTemplateName, nodeID, PodNameV1)
	assert.Equal(t, nodeID, name)

	// long case
	longWfName := "alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever"
	longTemplateName := "alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation"

	sum := len(longWfName) + len(longTemplateName)
	assert.Greater(t, sum, maxK8sResourceNameLength-k8sNamingHashLength)

	expected = fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	actual = ensurePodNamePrefixLength(expected)

	assert.Equal(t, maxK8sResourceNameLength-k8sNamingHashLength-1, len(actual))

	name = PodName(longWfName, nodeName, longTemplateName, nodeID, PodNameV1)
	assert.Equal(t, nodeID, name)

	///////////////////////////////////////////////////////////////////////////////////////////
	// POD_NAMES v2:
	///////////////////////////////////////////////////////////////////////////////////////////
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeName))
	expectedPodName := fmt.Sprintf("wfname-templatename-%v", h.Sum32())
	name = PodName(shortWfName, nodeName, shortTemplateName, expectedPodName, PodNameV2)
	assert.Equal(t, expectedPodName, name)

}
