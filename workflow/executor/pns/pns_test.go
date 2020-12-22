package pns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_backoffOver30s(t *testing.T) {
	x := backoffOver30s
	assert.Equal(t, 1*time.Second, x.Step())
	assert.Equal(t, 2*time.Second, x.Step())
	assert.Equal(t, 4*time.Second, x.Step())
	assert.Equal(t, 8*time.Second, x.Step())
	assert.Equal(t, 16*time.Second, x.Step())
	assert.Equal(t, 32*time.Second, x.Step())
	assert.Equal(t, 64*time.Second, x.Step())
}

func TestPNSExecutor_parseContainerIDFromCgroupLine(t *testing.T) {
	testCases := []struct {
		line     string
		expected string
	}{
		{
			line:     "",
			expected: "",
		},
		{
			line:     "5:rdma:/",
			expected: "",
		},
		{
			line:     "8:cpu,cpuacct:/kubepods/besteffort/pod2fad8aad-dcd0-4fef-b45a-151630b9a4b5/b844ef90162af07ad3fb2961ffb2f528f8bf7c9edb8c45465fd8651fab8a78c1",
			expected: "b844ef90162af07ad3fb2961ffb2f528f8bf7c9edb8c45465fd8651fab8a78c1",
		},
		{
			line:     "2:cpu,cpuacct:/system.slice/containerd.service/kubepods-burstable-podf61fae9b_d7a7_11ea_bb0c_12a065621d2b.slice:cri-containerd:b6b894a028b7ec8e0f98c57a5f7b9735ad792276c3ce52bc795fcd367d829de9",
			expected: "b6b894a028b7ec8e0f98c57a5f7b9735ad792276c3ce52bc795fcd367d829de9",
		},
		{
			line:     "8:cpu,cpuacct:/kubepods/besteffort/pod2fad8aad-dcd0-4fef-b45a-151630b9a4b5/crio-7a92a067289f6197148912be1c15f20f0330c7f3c541473d3b9c4043ca137b42.scope",
			expected: "7a92a067289f6197148912be1c15f20f0330c7f3c541473d3b9c4043ca137b42",
		},
		{
			line:     "2:cpuacct,cpu:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod1cd87fe8_8ea0_11ea_8d51_566f300c000a.slice/docker-6b40fc7f75fe3210621a287412ac056e43554b1026a01625b48ba7d136d8a125.scope",
			expected: "6b40fc7f75fe3210621a287412ac056e43554b1026a01625b48ba7d136d8a125",
		},
	}

	for _, testCase := range testCases {
		containerID := parseContainerIDFromCgroupLine(testCase.line)
		assert.Equal(t, testCase.expected, containerID)
	}
}
