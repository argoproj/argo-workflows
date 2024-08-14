package util

import (
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPodNameV2(t *testing.T) {
	nodeName := "nodename"
	nodeID := "1"

	///////////////////////////////////////////////////////////////////////////////////////////
	// short case
	///////////////////////////////////////////////////////////////////////////////////////////
	shortWfName := "wfname"
	shortTemplateName := "templatename"

	expected := fmt.Sprintf("%s-%s", shortWfName, shortTemplateName)
	actual := ensurePodNamePrefixLength(expected)
	require.Equal(t, expected, actual)

	// derive the expected pod name
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeName))
	expectedPodName := fmt.Sprintf("wfname-templatename-%v", h.Sum32())

	name := GeneratePodName(shortWfName, nodeName, shortTemplateName, nodeID, PodNameV2)
	require.Equal(t, expectedPodName, name)

	///////////////////////////////////////////////////////////////////////////////////////////
	// long case
	///////////////////////////////////////////////////////////////////////////////////////////
	longWfName := "alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever"
	longTemplateName := "alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation"

	sum := len(longWfName) + len(longTemplateName)
	require.Greater(t, sum, maxK8sResourceNameLength-k8sNamingHashLength)

	expected = fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	actual = ensurePodNamePrefixLength(expected)

	require.Len(t, actual, maxK8sResourceNameLength-k8sNamingHashLength-1)

	longPrefix := fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	expectedPodName = fmt.Sprintf("%s-%v", longPrefix[0:maxK8sResourceNameLength-k8sNamingHashLength-1], h.Sum32())

	name = GeneratePodName(longWfName, nodeName, longTemplateName, nodeID, PodNameV2)
	require.Equal(t, expectedPodName, name)

	h = fnv.New32a()
	_, _ = h.Write([]byte("stp.inline"))
	name = GeneratePodName(shortWfName, "stp.inline", "", nodeID, PodNameV2)
	require.Equal(t, fmt.Sprintf("wfname-%v", h.Sum32()), name)
}
