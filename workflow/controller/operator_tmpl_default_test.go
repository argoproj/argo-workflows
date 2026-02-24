package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const stepWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello
  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello1"}]
    - - name: hello2a
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello2a"}]
      - name: hello2b
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello2b"}]

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

const dagWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-coinflip-
spec:
  entrypoint: diamond
  templates:
  - name: diamond
    dag:
      tasks:
      - name: A
        template: coinflip
      - name: B
        dependencies: [A]
        template: coinflip
      - name: C
        dependencies: [A]
        template: coinflip
      - name: D
        dependencies: [B, C]
        template: coinflip

  - name: coinflip
    steps:
    - - name: flip-coin
        template: flip-coin
    - - name: heads
        template: heads
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: coinflip
        when: "{{steps.flip-coin.outputs.result}} == tails"

  - name: flip-coin
    script:
      image: python:alpine3.23
      command: [python]
      source: |
        import random
        result = "heads" if random.randint(0,1) == 0 else "tails"
        print(result)

  - name: heads
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo \"it was heads\""]
`

const defaultWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  labels:
    workflows.argoproj.io/archive-strategy: "false"
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

const httpWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: http-
spec:
  entrypoint: http-template
  templates:
  - name: http-template
    http:
      url: http://dummy.rest.api/endpoint
`

func TestSetTemplateDefault(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.Config.WorkflowDefaults = &wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			TemplateDefaults: &wfv1.Template{
				ActiveDeadlineSeconds: intstrutil.ParsePtr("110"),
				Container: &apiv1.Container{
					ImagePullPolicy: "Never",
				},
			},
		},
	}
	t.Run("tmplDefaultInConfig", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(defaultWf)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		tmpl := woc.execWf.Spec.Templates[0]
		err = woc.mergedTemplateDefaultsInto(&tmpl)
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, intstrutil.ParsePtr("110"), tmpl.ActiveDeadlineSeconds)
		assert.Equal(t, apiv1.PullNever, tmpl.Container.ImagePullPolicy)
	})
	t.Run("tmplDefaultInWf", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(defaultWf)
		envs := []apiv1.EnvVar{
			{
				Name: "test",
			},
		}
		wf.Spec.TemplateDefaults = &wfv1.Template{
			ActiveDeadlineSeconds: intstrutil.ParsePtr("150"),
			Script: &wfv1.ScriptTemplate{
				Source: "Test",
			},
			Container: &apiv1.Container{
				ImagePullPolicy: apiv1.PullIfNotPresent,
				Env:             envs,
			},
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		tmpl := woc.execWf.Spec.Templates[0]
		err = woc.mergedTemplateDefaultsInto(&tmpl)
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl.ActiveDeadlineSeconds)
		assert.Equal(t, apiv1.PullIfNotPresent, tmpl.Container.ImagePullPolicy)
		assert.Len(t, tmpl.Container.Env, 1)
		assert.Equal(t, "test", tmpl.Container.Env[0].Name)
		assert.Nil(t, tmpl.Script)
	})
	t.Run("stepTmplDefaultWf", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(stepWf)
		envs := []apiv1.EnvVar{
			{
				Name: "test",
			},
		}
		wf.Spec.TemplateDefaults = &wfv1.Template{
			ActiveDeadlineSeconds: intstrutil.ParsePtr("150"),
			Script: &wfv1.ScriptTemplate{
				Source: "Test",
			},
			Container: &apiv1.Container{
				ImagePullPolicy: apiv1.PullIfNotPresent,
				Env:             envs,
			},
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		tmpl := woc.execWf.Spec.Templates[0]
		err = woc.mergedTemplateDefaultsInto(&tmpl)

		require.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl.ActiveDeadlineSeconds)
		assert.Nil(t, tmpl.Container)
		assert.Equal(t, wfv1.TemplateTypeSteps, tmpl.GetType())

		tmpl1 := woc.execWf.Spec.Templates[1]
		err = woc.mergedTemplateDefaultsInto(&tmpl1)
		require.NoError(t, err)
		assert.NotNil(t, tmpl1)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl1.ActiveDeadlineSeconds)
		assert.Equal(t, apiv1.PullIfNotPresent, tmpl1.Container.ImagePullPolicy)
		assert.Len(t, tmpl1.Container.Env, 1)
		assert.Equal(t, "test", tmpl1.Container.Env[0].Name)
		assert.Nil(t, tmpl1.Script)
	})
	t.Run("DagTmplDefaultWf", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(dagWf)
		envs := []apiv1.EnvVar{
			{
				Name: "test",
			},
		}
		wf.Spec.TemplateDefaults = &wfv1.Template{
			ActiveDeadlineSeconds: intstrutil.ParsePtr("150"),
			Script: &wfv1.ScriptTemplate{
				Container: apiv1.Container{
					Env: envs,
				},
			},
			Container: &apiv1.Container{
				ImagePullPolicy: apiv1.PullIfNotPresent,
				Env:             envs,
			},
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		tmpl := woc.execWf.Spec.Templates[0]
		err = woc.mergedTemplateDefaultsInto(&tmpl)

		require.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl.ActiveDeadlineSeconds)
		assert.Nil(t, tmpl.Container)
		assert.Equal(t, wfv1.TemplateTypeDAG, tmpl.GetType())

		tmpl1 := woc.execWf.Spec.Templates[2]
		err = woc.mergedTemplateDefaultsInto(&tmpl1)
		require.NoError(t, err)
		assert.NotNil(t, tmpl1)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl1.ActiveDeadlineSeconds)
		assert.NotNil(t, tmpl1.Script)
		assert.Len(t, tmpl1.Script.Env, 1)
		assert.Equal(t, "test", tmpl1.Script.Env[0].Name)

		tmpl2 := woc.execWf.Spec.Templates[3]
		err = woc.mergedTemplateDefaultsInto(&tmpl2)
		require.NoError(t, err)
		assert.NotNil(t, tmpl2)
		assert.Equal(t, intstrutil.ParsePtr("150"), tmpl2.ActiveDeadlineSeconds)
		assert.Equal(t, apiv1.PullIfNotPresent, tmpl2.Container.ImagePullPolicy)
		assert.Len(t, tmpl2.Container.Env, 1)
		assert.Equal(t, "test", tmpl2.Container.Env[0].Name)
		assert.Nil(t, tmpl2.Script)
	})
	t.Run("HTTPTmplDefaultWf", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(httpWf)
		wf.Spec.TemplateDefaults = &wfv1.Template{
			Container: &apiv1.Container{
				ImagePullPolicy: apiv1.PullIfNotPresent,
			},
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err := woc.setExecWorkflow(ctx)
		require.NoError(t, err)
		tmpl := woc.execWf.Spec.Templates[0]
		err = woc.mergedTemplateDefaultsInto(&tmpl)
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, wfv1.TemplateTypeHTTP, tmpl.GetType())
		assert.Nil(t, tmpl.Container)
	})
}
