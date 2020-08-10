package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newFailureText(t *testing.T) {
	x := newFailureText("github.com/argoproj/argo/controller", `time="2020-08-10T11:04:52-07:00" level=info msg="Processing workflow" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:52-07:00" level=info msg="Updated phase  -> Running" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:52-07:00" level=info msg="Created PDB resource for workflow." namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:52-07:00" level=info msg="Pod node artifact-repo-config-ref initialized Pending" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:52-07:00" level=info msg="Created pod: artifact-repo-config-ref (artifact-repo-config-ref)" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:52-07:00" level=info msg="Workflow update successful" namespace= phase=Running resourceVersion= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Updated phase Running -> Succeeded" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Marking workflow completed" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Deleted PDB resource for workflow." namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Processing workflow" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=error msg="Unexpected pod phase for artifact-repo-config-ref: "
time="2020-08-10T11:04:53-07:00" level=info msg="Updating node artifact-repo-config-ref status Pending -> Error"
time="2020-08-10T11:04:53-07:00" level=info msg="Updating node artifact-repo-config-ref message: Unexpected pod phase for artifact-repo-config-ref: "
time="2020-08-10T11:04:53-07:00" level=info msg="Updated phase Succeeded -> Error" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Updated message  -> Unexpected pod phase for artifact-repo-config-ref: " namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Marking workflow completed" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=warning msg="Failed to delete PDB." err="poddisruptionbudgets.policy \"artifact-repo-config-ref\" not found" namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Deleted PDB resource for workflow." namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Checking daemoned children of " namespace= workflow=artifact-repo-config-ref
time="2020-08-10T11:04:53-07:00" level=info msg="Workflow update successful" namespace= phase=Error resourceVersion= workflow=artifact-repo-config-ref
operator_test.go:2912: 
Error Trace:	operator_test.go:2912
Error:      	Received unexpected error:
            	poddisruptionbudgets.policy "artifact-repo-config-ref" not found
Test:       	TestPDBCreation`)
	assert.Equal(t, failureText{
		file:    "controller/operator_test.go",
		line:    2912,
		message: `Error Trace:	operator_test.go:2912%0AError:      	Received unexpected error:%0A            	poddisruptionbudgets.policy "artifact-repo-config-ref" not found%0ATest:       	TestPDBCreation`,
	}, x)
}
