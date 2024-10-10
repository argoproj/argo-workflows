package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow missing a workflow template ref", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowPath, clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow missing a cluster workflow template ref", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowPath, wftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow template on its own", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{wftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a cluster workflow template on its own", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a workflow and templates", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowPath, wftmplPath, clusterWftmplPath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting a directory", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{dir}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

	t.Run("linting one file from stdin", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }() // Restore original Stdin
		os.Stdin, err = os.Open(clusterWftmplPath)
		require.NoError(t, err)
		defer func() { _ = os.Stdin.Close() }() // close previously opened path to avoid errors trying to remove the file.

		err = runLint(context.Background(), []string{workflowPath, wftmplPath, "-"}, true, nil, "pretty", true)

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
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowCaseSensitivePath}, true, nil, "pretty", true)

		require.NoError(t, err)
		assert.True(t, fatal, "should have exited")
	})

	t.Run("linting a workflow with case sensitive fields and strict disabled", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		err = runLint(context.Background(), []string{workflowCaseSensitivePath}, true, nil, "pretty", false)

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
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }
		err = runLint(context.Background(), []string{workflowMultiDocsPath}, true, nil, "pretty", false)

		require.NoError(t, err)
		assert.False(t, fatal, "should not have exited")
	})

}
