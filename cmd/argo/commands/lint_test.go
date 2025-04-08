package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func Test_OfflineLint(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0755))
	wftmplPath := filepath.Join(subdir, "wftmpl.yaml")
	err := os.WriteFile(wftmplPath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template-local-arg
  namespace: test
spec:
  templates:
    - name: hello-world
      inputs:
        parameters:
          - name: msg
            value: hello world
      container:
        image: docker/whalesay
        command:
          - cowsay
        args:
          - '{{inputs.parameters.msg}}'
`), 0644)
	require.NoError(t, err)

	clusterWftmplPath := filepath.Join(subdir, "cluster-workflow-template.yaml")
	err = os.WriteFile(clusterWftmplPath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: hello-world-cluster
spec:
  templates:
  - name: hello-world
    inputs:
      parameters:
        - name: msg
          value: hello world
    container:
      image: docker/whalesay
      command:
        - cowsay
      args:
        - '{{inputs.parameters.msg}}'
`), 0644)
	require.NoError(t, err)

	workflowPath := filepath.Join(subdir, "workflow.yaml")
	err = os.WriteFile(workflowPath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-local-arg-
  namespace: test
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      steps:
        - - name: hello-world
            templateRef:
              name: hello-world-template-local-arg
              template: hello-world
          - name: hello-world-cluster
            templateRef:
              name: hello-world-cluster
              template: hello-world
              clusterScope: true
`), 0644)
	require.NoError(t, err)

	t.Run("linting a workflow missing references", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow missing a workflow template ref", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowPath, clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow missing a cluster workflow template ref", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowPath, wftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow template on its own", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{wftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a cluster workflow template on its own", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a workflow and templates", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowPath, wftmplPath, clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a directory", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{dir}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting one file from stdin", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin, err = os.Open(clusterWftmplPath)
		require.NoError(t, err)
		defer func() { _ = os.Stdin.Close() }()

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowPath, wftmplPath, "-"}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	workflowCaseSensitivePath := filepath.Join(subdir, "workflowCaseSensitive.yaml")
	err = os.WriteFile(workflowCaseSensitivePath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoInt: whalesay
  templates:
    - name: whalesay
      container:
        image: docker/whalesay
        command: [ cowsay ]
        args: [ "hello world" ]
        resources:
          limits:
            memory: 32Mi
            cpu: 100m
`), 0644)
	require.NoError(t, err)

	t.Run("linting a workflow with case sensitive fields and strict enabled", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowCaseSensitivePath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow with case sensitive fields and strict disabled", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })

		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowCaseSensitivePath}, true, nil, "pretty", false)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	workflowMultiDocsPath := filepath.Join(subdir, "workflowMultiDocs.yaml")
	err = os.WriteFile(workflowMultiDocsPath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template-local-arg-1
spec:
  templates:
    - name: hello-world
      inputs:
        parameters:
          - name: msg
            value: 'hello world'
      container:
        image: busybox
        command: [echo]
        args: ['{{inputs.parameters.msg}}']
---
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template-local-arg-2
spec:
  templates:
    - name: hello-world
      inputs:
        parameters:
          - name: msg
            value: 'hello world'
      container:
        image: busybox
        command: [echo]
        args: ['{{inputs.parameters.msg}}']
---
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-local-arg-
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      steps:
        - - name: hello-world
            templateRef:
              name: hello-world-template-local-arg-2
              template: hello-world
`), 0644)
	require.NoError(t, err)

	t.Run("linting a workflow in multi-documents yaml", func(t *testing.T) {
		defer func() { logging.SetExitFunc(nil) }()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })
		ctx := logging.TestContext(t.Context())
		err = runLint(ctx, []string{workflowMultiDocsPath}, true, nil, "pretty", false)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting an invalid YAML file with offline mode logs once", func(t *testing.T) {
		invalidYAMLPath := filepath.Join(subdir, "invalid-yaml.yaml")
		err = os.WriteFile(invalidYAMLPath, []byte(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
	 generateName: hello-world-
	 labels:
	   workflows.argoproj.io/archive-strategy: "false"
	 annotations:
	   workflows.argoproj.io/description: |
	     This is a simple hello world example.
spec:
	 entrypoint: hello-world
	 templates:
	 - name: hello-world
	   container:
	     image: busybox
	command: [echo]
	args: ["hello world"]
`), 0644)
		require.NoError(t, err)

		hook := logging.NewTestHook()
		ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text, hook))

		originalExitFunc := logging.GetExitFunc()
		var fatal bool
		logging.SetExitFunc(func(int) { fatal = true })
		defer logging.SetExitFunc(originalExitFunc)

		err = runLint(ctx, []string{invalidYAMLPath}, true, nil, "pretty", true)
		require.NoError(t, err)

		yamlErrCount := 0
		for _, entry := range hook.AllEntries() {
			if entry.Level == logging.Error && strings.Contains(entry.Msg, "yaml file at index 0 is not valid") {
				yamlErrCount++
			}
		}

		assert.Equal(t, 1, yamlErrCount, "parse errors should only be logged once in offline mode")
		assert.True(t, fatal, "Should have exited with error code for parse error")
	})
}
