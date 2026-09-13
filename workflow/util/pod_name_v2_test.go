package util

import (
	"fmt"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPodNameV2(t *testing.T) {
	nodeName := "nodename"
	// a node's ID is <workflow name>-<fnv32a(node name)>; the pod name takes
	// the hash from there, so the expected values below must match rehashing
	// nodeName as the pre-existing implementation did.
	nodeID := func(wfName, nodeName string) string {
		h := fnv.New32a()
		_, _ = h.Write([]byte(nodeName))
		return fmt.Sprintf("%s-%v", wfName, h.Sum32())
	}

	///////////////////////////////////////////////////////////////////////////////////////////
	// short case
	///////////////////////////////////////////////////////////////////////////////////////////
	shortWfName := "wfname"
	shortTemplateName := "templatename"

	expected := fmt.Sprintf("%s-%s", shortWfName, shortTemplateName)
	actual := ensurePodNamePrefixLength(expected)
	assert.Equal(t, expected, actual)

	// derive the expected pod name
	h := fnv.New32a()
	_, _ = h.Write([]byte(nodeName))
	expectedPodName := fmt.Sprintf("wfname-templatename-%v", h.Sum32())

	name := GeneratePodName(shortWfName, nodeName, shortTemplateName, nodeID(shortWfName, nodeName), PodNameV2)
	assert.Equal(t, expectedPodName, name)

	///////////////////////////////////////////////////////////////////////////////////////////
	// long case
	///////////////////////////////////////////////////////////////////////////////////////////
	longWfName := "alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever"
	longTemplateName := "alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation"

	sum := len(longWfName) + len(longTemplateName)
	assert.Greater(t, sum, maxK8sResourceNameLength-k8sNamingHashLength)

	expected = fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	actual = ensurePodNamePrefixLength(expected)

	assert.Len(t, actual, maxK8sResourceNameLength-k8sNamingHashLength-1)

	longPrefix := fmt.Sprintf("%s-%s", longWfName, longTemplateName)
	expectedPodName = fmt.Sprintf("%s-%v", longPrefix[0:maxK8sResourceNameLength-k8sNamingHashLength-1], h.Sum32())

	name = GeneratePodName(longWfName, nodeName, longTemplateName, nodeID(longWfName, nodeName), PodNameV2)
	assert.Equal(t, expectedPodName, name)

	h = fnv.New32a()
	_, _ = h.Write([]byte("stp.inline"))
	name = GeneratePodName(shortWfName, "stp.inline", "", nodeID(shortWfName, "stp.inline"), PodNameV2)
	assert.Equal(t, fmt.Sprintf("wfname-%v", h.Sum32()), name)

	// an ID that is not <workflow name>-<hash> falls back to hashing the node
	// name, keeping the pod name within the k8s length limit
	name = GeneratePodName(longWfName, nodeName, longTemplateName, "unrelated", PodNameV2)
	assert.Equal(t, expectedPodName, name)
	assert.LessOrEqual(t, len(name), maxK8sResourceNameLength)

	// a node that lost an ID collision carries the suffix in its ID, and so in
	// its pod name, while its name is unchanged
	name = GeneratePodName(shortWfName, nodeName, shortTemplateName, nodeID(shortWfName, nodeName+"~1"), PodNameV2)
	h = fnv.New32a()
	_, _ = h.Write([]byte(nodeName + "~1"))
	assert.Equal(t, fmt.Sprintf("wfname-templatename-%v", h.Sum32()), name)
}
