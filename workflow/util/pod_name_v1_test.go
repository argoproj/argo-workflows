package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPodNameV1(t *testing.T) {
	nodeName := "nodename"
	nodeID := "1"

	// short case
	shortWfName := "wfname"
	shortTemplateName := "templatename"

	expected := fmt.Sprintf("%s-%s", shortWfName, shortTemplateName)
	actual := ensurePodNamePrefixLength(expected)
	require.Equal(t, expected, actual)

	name := GeneratePodName(shortWfName, nodeName, shortTemplateName, nodeID, PodNameV1)
	require.Equal(t, nodeID, name)

	// long case
	longWfName := "alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever"
	longTemplateName := "alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation"

	sum := len(longWfName) + len(longTemplateName)
	require.Greater(t, sum, maxK8sResourceNameLength-k8sNamingHashLength)

	expected = fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	actual = ensurePodNamePrefixLength(expected)

	require.Len(t, actual, maxK8sResourceNameLength-k8sNamingHashLength-1)

	name = GeneratePodName(longWfName, nodeName, longTemplateName, nodeID, PodNameV1)
	require.Equal(t, nodeID, name)

}
