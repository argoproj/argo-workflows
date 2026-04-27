package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

// TestSetGlobalParameters_ScopeMirrorsGlobalParams verifies that after
// setGlobalParameters the keys migrated to scope writes contain the same
// values as the legacy globalParams map. This is the safety net for the
// gradual migration from globalParams to scope: if any dual-write drifts,
// this fails.
func TestSetGlobalParameters_ScopeMirrorsGlobalParams(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: mirror-test
  namespace: argo
  uid: xyz-789
spec:
  entrypoint: main
  serviceAccountName: runner
  arguments:
    parameters:
      - name: greeting
        value: hello
      - name: target
        value: world
  templates:
    - name: main
      container:
        image: alpine
`)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	require.NoError(t, woc.setGlobalParameters(wf.Spec.Arguments))

	// Keys migrated so far: identity + parameters.<name>.
	identity := map[string]func() string{
		"workflow.name":                func() string { return woc.wf.Name },
		"workflow.namespace":           func() string { return woc.wf.Namespace },
		"workflow.uid":                 func() string { return string(woc.wf.UID) },
		"workflow.mainEntrypoint":      func() string { return woc.execWf.Spec.Entrypoint },
		"workflow.serviceAccountName":  func() string { return woc.execWf.Spec.ServiceAccountName },
	}
	for key, wantFn := range identity {
		want := wantFn()
		assert.Equal(t, want, woc.globalParams[key], "globalParams[%s]", key)
		got, ok := woc.scope.AsAnyMap()[key].(string)
		assert.True(t, ok, "scope[%s] missing or wrong type", key)
		assert.Equal(t, want, got, "scope[%s]", key)
	}

	// workflow.parameters.<name> — parameterised key.
	for _, p := range wf.Spec.Arguments.Parameters {
		concrete := fmt.Sprintf("workflow.parameters.%s", p.Name)
		want := p.Value.String()
		assert.Equal(t, want, woc.globalParams[concrete])
		got, _ := varkeys.WorkflowParametersByName.Get(woc.scope, p.Name)
		assert.Equal(t, want, got, "scope workflow.parameters.%s", p.Name)
	}
}

