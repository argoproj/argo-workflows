package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Regression tests from #16223 (drop values when skipped arguments are substituted),
// restored verbatim from the pre-Engine controller.

var dagSkippedOutputDefault = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-skipped-output-default-
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
      - name: consumer
        template: consume
        depends: "producer.Succeeded || producer.Skipped"
        arguments:
          parameters:
          - name: in
            value: "{{tasks.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
          default: "default-from-producer"
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
  - name: consume
    inputs:
      parameters:
      - name: in
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.in}}"]
`

// TestDAGSkippedOutputDefault verifies that when a DAG task is skipped and its template declares
// an output parameter with a valueFrom.default, a downstream task referencing that output in its
// INPUT receives the producer's declared default instead of an empty string.
func TestDAGSkippedOutputDefault(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagSkippedOutputDefault)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "default-from-producer", in.Value.String())
}

var dagSkippedOutputDefaultAggregate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-skipped-output-default-aggregate
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
    outputs:
      parameters:
      - name: result
        valueFrom:
          parameter: "{{tasks.producer.outputs.parameters.msg}}"
          default: "default-from-aggregator"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
          default: "default-from-producer"
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
`

// TestDAGSkippedOutputDefaultAggregate verifies the precedence decision: when a skipped producer
// declares an output valueFrom.default AND the aggregating template's output parameter declares its
// own valueFrom.default, the producer's default wins (it populates scope as a real value, so the
// aggregator's skipped-fallback never fires).
func TestDAGSkippedOutputDefaultAggregate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagSkippedOutputDefaultAggregate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("dag-skipped-output-default-aggregate")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 1)
	assert.Equal(t, "default-from-producer", dagNode.Outputs.Parameters[0].Value.String())
}

var dagSkippedOutputExprDefaultAggregate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-skipped-output-expr-default-aggregate
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
    outputs:
      parameters:
      - name: result
        valueFrom:
          expression: "tasks.producer.outputs.parameters.msg"
          default: "default-from-aggregator"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
`

// TestDAGSkippedOutputExprDefaultAggregate verifies that a ValueFrom.Expression referencing a skipped
// defaultless output WITHOUT handling the absent (nil) optional mirrors the inline {{= ...}} semantics:
// the expression fails to resolve, and the output parameter's own valueFrom.default applies via the
// error fallback (instead of silently emitting "").
func TestDAGSkippedOutputExprDefaultAggregate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagSkippedOutputExprDefaultAggregate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("dag-skipped-output-expr-default-aggregate")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 1)
	assert.Equal(t, "default-from-aggregator", dagNode.Outputs.Parameters[0].Value.String())
}

var dagSkippedRefDynamicTemplateName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-skipped-ref-dynamic-template
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
      - name: consumer
        templateRef:
          name: "{{item.wftmpl}}"
          template: "{{item.tmpl}}"
        withItems:
        - { wftmpl: "skipped-ref-consume", tmpl: "consume" }
        depends: "producer.Succeeded || producer.Skipped"
        arguments:
          parameters:
          - name: in
            value: "{{tasks.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
`

// TestDAGSkippedRefDynamicTemplateName verifies that a task whose templateRef is itself templated
// ("{{item.*}}", resolved only at expansion) is still rescued by the consumed template's input
// default when an argument references a skipped defaultless output: the argument is marked with the
// absent-optional sentinel before substitution and ProcessArgs interprets it at consumption time,
// when the dynamic templateRef has been resolved.
func TestDAGSkippedRefDynamicTemplateName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow(dagSkippedRefDynamicTemplateName), wfv1.MustUnmarshalWorkflowTemplate(skippedRefConsumeWorkflowTemplate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow(dagSkippedRefDynamicTemplateName), controller)
	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	var consumer *wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeError, node.Phase, "node %q should not error: %s", node.DisplayName, node.Message)
		if strings.HasPrefix(node.DisplayName, "consumer(") {
			n := node
			consumer = &n
		}
	}
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "FALLBACK", in.Value.String())
}

var dagSkippedInputDefaultSuppressed = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-skipped-input-default-suppressed
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
      - name: consumer
        template: consume
        depends: "producer.Succeeded || producer.Skipped"
        arguments:
          parameters:
          - name: in
            value: "{{tasks.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt   # NOTE: no valueFrom.default here
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
  - name: consume
    inputs:
      parameters:
      - name: in
        default: "FALLBACK-FROM-INPUT"
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.in}}"]
`

// TestDAGSkippedInputDefaultUsed verifies that when a producer is skipped and its output declares NO
// valueFrom.default, a consumer referencing that output in its input falls back to the consumer's OWN
// input default rather than receiving the empty skipped-marker.
func TestDAGSkippedInputDefaultUsed(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagSkippedInputDefaultSuppressed)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	// The skipped reference must NOT clobber the consumer's own input default.
	assert.Equal(t, "FALLBACK-FROM-INPUT", in.Value.String(),
		"a skipped output reference should fall back to the consumer's input default")
}

var dagSkippedInlineExpressionFallback = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-skipped-inline-expr-fallback-
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: producer
        template: produce
        when: "false"
      - name: consumer
        template: consume
        depends: "producer.Succeeded || producer.Skipped"
        arguments:
          parameters:
          - name: in
            value: "{{= tasks.producer.outputs.parameters.msg ?? 'inline-fallback'}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
  - name: consume
    inputs:
      parameters:
      - name: in
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.in}}"]
`

// TestDAGSkippedInlineExpressionFallback verifies that an inline {{= ... ?? ...}} expression in a
// task argument sees a skipped/omitted dependency's defaultless output as nil (absent), so the ??
// fallback applies, instead of the empty-string flattening that previously made ?? a no-op.
func TestDAGSkippedInlineExpressionFallback(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagSkippedInlineExpressionFallback)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "inline-fallback", in.Value.String())
}

var stepsSkippedOutputDefault = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-skipped-output-default-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: producer
        template: produce
        when: "false"
    - - name: consumer
        template: consume
        arguments:
          parameters:
          - name: in
            value: "{{steps.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
          default: "default-from-producer"
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
  - name: consume
    inputs:
      parameters:
      - name: in
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.in}}"]
`

// TestStepsSkippedOutputDefault verifies that when a step is skipped and its template declares an
// output parameter with a valueFrom.default, a downstream step referencing that output in its INPUT
// receives the producer's declared default instead of an empty string.
func TestStepsSkippedOutputDefault(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsSkippedOutputDefault)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	consumer := woc.wf.Status.Nodes.FindByDisplayName("consumer")
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "default-from-producer", in.Value.String())
}

var stepsSkippedRefDynamicTemplateName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-skipped-ref-dynamic-template
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: producer
        template: produce
        when: "false"
    - - name: consumer
        templateRef:
          name: "{{item.wftmpl}}"
          template: "{{item.tmpl}}"
        withItems:
        - { wftmpl: "skipped-ref-consume", tmpl: "consume" }
        arguments:
          parameters:
          - name: in
            value: "{{steps.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
`

// TestStepsSkippedRefDynamicTemplateName verifies that a step whose templateRef is itself templated
// ("{{item.*}}", resolved only at expansion) is still rescued by the consumed template's input
// default when an argument references a skipped defaultless output: the argument is marked with the
// absent-optional sentinel before substitution and ProcessArgs interprets it at consumption time,
// when the dynamic templateRef has been resolved.
func TestStepsSkippedRefDynamicTemplateName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow(stepsSkippedRefDynamicTemplateName), wfv1.MustUnmarshalWorkflowTemplate(skippedRefConsumeWorkflowTemplate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow(stepsSkippedRefDynamicTemplateName), controller)
	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	var consumer *wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeError, node.Phase, "node %q should not error: %s", node.DisplayName, node.Message)
		if strings.HasPrefix(node.DisplayName, "consumer(") {
			n := node
			consumer = &n
		}
	}
	require.NotNil(t, consumer, "consumer should be scheduled even though producer was skipped")
	require.NotNil(t, consumer.Inputs)
	in := consumer.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "FALLBACK", in.Value.String())
}

var stepsWhenFalseSkippedRefDrop = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-when-false-skipped-ref-drop
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: producer
        template: produce
        when: "false"
    - - name: consumer
        templateRef:
          name: "{{item.wftmpl}}"
          template: "{{item.tmpl}}"
        withItems:
        - { wftmpl: "skipped-ref-consume", tmpl: "consume" }
        when: "false"
        arguments:
          parameters:
          - name: in
            value: "{{steps.producer.outputs.parameters.msg}}"
  - name: produce
    outputs:
      parameters:
      - name: msg
        valueFrom:
          path: /tmp/out.txt
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo hello > /tmp/out.txt"]
`

// TestStepsWhenFalseSkipsDropPass verifies that a step whose when-clause evaluates to false does
// not fail on an absent-optional argument: a when-false step never substitutes its body (and item
// expansion strips nils for never-executing steps), so the absent optional is never hit and the
// step group proceeds with the step Skipped.
func TestStepsWhenFalseSkipsDropPass(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wfv1.MustUnmarshalWorkflow(stepsWhenFalseSkippedRefDrop), wfv1.MustUnmarshalWorkflowTemplate(skippedRefConsumeWorkflowTemplate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wfv1.MustUnmarshalWorkflow(stepsWhenFalseSkippedRefDrop), controller)
	woc.operate(ctx)

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	for _, node := range woc.wf.Status.Nodes {
		assert.NotEqual(t, wfv1.NodeError, node.Phase, "node %q should not error: %s", node.DisplayName, node.Message)
	}
	assert.NotEqual(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.NotEqual(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var skippedRefConsumeWorkflowTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: skipped-ref-consume
  namespace: default
spec:
  templates:
  - name: consume
    inputs:
      parameters:
      - name: in
        default: "FALLBACK"
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.in}}"]
`
