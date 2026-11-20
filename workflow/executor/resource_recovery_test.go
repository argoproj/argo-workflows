package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestExecResourceRecoveryPreservesFirstCreateError(t *testing.T) {
	for _, recovering := range []bool{false, true} {
		t.Run(map[bool]string{false: "first execution", true: "replacement executor"}[recovering], func(t *testing.T) {
			manifest := filepath.Join(t.TempDir(), "manifest.yaml")
			require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: recover-me\n"), 0o600))
			we := WorkflowExecutor{
				nodeID:                     "resource-recovery",
				workflowUID:                "workflow-uid",
				Template:                   wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "create"}},
				resourceTemplateRecovering: recovering,
			}
			var calls [][]string
			run := func(_ context.Context, args ...string) ([]byte, error) {
				calls = append(calls, append([]string(nil), args...))
				if args[1] == "create" {
					return nil, &exec.ExitError{Stderr: []byte(`Error from server (AlreadyExists): configmaps "recover-me" already exists`)}
				}
				return []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"recover-me","namespace":"target","annotations":{"workflows.argoproj.io/workflow-uid":"workflow-uid","workflows.argoproj.io/node-id":"resource-recovery"}}}`), nil
			}
			flags := []string{"--namespace", "target", "--context", "target-cluster", "--validate=false", "--field-manager=resource-test", "--save-config"}
			namespace, name, _, err := we.execResource(logging.TestContext(t.Context()), "create", manifest, flags, run)
			if !recovering {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "AlreadyExists")
				require.Len(t, calls, 1, "first-run conflicts must not adopt existing resources")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "target", namespace)
			assert.Equal(t, "configmap./recover-me", name)
			require.Len(t, calls, 2)
			stagedManifest := calls[0][len(calls[0])-3]
			assert.NotEqual(t, manifest, stagedManifest)
			assert.Equal(t, []string{"kubectl", "get", "--context=target-cluster", "--namespace=target", "-f", stagedManifest, "-o", "json"}, calls[1])
			_, statErr := os.Stat(stagedManifest)
			assert.ErrorIs(t, statErr, os.ErrNotExist, "remove the staged manifest after kubectl completes")
		})
	}
}

func TestExecResourceSuccessfulActionsRunOnce(t *testing.T) {
	for _, action := range []string{"create", "get", "patch"} {
		t.Run(action, func(t *testing.T) {
			manifest := filepath.Join(t.TempDir(), "manifest.yaml")
			require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ordinary-resource\n"), 0o600))
			we := WorkflowExecutor{
				nodeID: "node", workflowUID: "workflow",
				Template: wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: action}},
			}
			calls := 0
			namespace, name, _, err := we.execResource(logging.TestContext(t.Context()), action, manifest, nil,
				func(_ context.Context, args ...string) ([]byte, error) {
					calls++
					assert.Equal(t, action, args[1])
					return []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"ordinary-resource","namespace":"target"}}`), nil
				})
			require.NoError(t, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, "target", namespace)
			assert.Equal(t, "configmap./ordinary-resource", name)
		})
	}
}

func TestExecResourceRecoveryRejectsUnrelatedObject(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: recover-me\n"), 0o600))
	for name, annotations := range map[string]map[string]string{
		"unmarked":       nil,
		"other workflow": {common.AnnotationKeyWorkflowUID: "other", common.AnnotationKeyNodeID: "resource-recovery"},
		"other node":     {common.AnnotationKeyWorkflowUID: "workflow-uid", common.AnnotationKeyNodeID: "other"},
	} {
		t.Run(name, func(t *testing.T) {
			we := WorkflowExecutor{nodeID: "resource-recovery", workflowUID: "workflow-uid", resourceTemplateRecovering: true}
			obj := unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap"}}
			obj.SetName("recover-me")
			obj.SetAnnotations(annotations)
			output, err := obj.MarshalJSON()
			require.NoError(t, err)
			_, _, _, err = we.execResource(logging.TestContext(t.Context()), "create", manifest, nil,
				func(_ context.Context, args ...string) ([]byte, error) {
					if args[1] == "create" {
						return nil, &exec.ExitError{Stderr: []byte(`Error from server (AlreadyExists): configmaps "recover-me" already exists`)}
					}
					return output, nil
				})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not created by this workflow node")
		})
	}
}

func TestExecResourceStopsIfDeterministicManifestCannotBeStaged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses the Unix TMPDIR variable")
	}
	manifest := filepath.Join(t.TempDir(), "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  generateName: recover-\n"), 0o600))
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing-directory"))
	we := WorkflowExecutor{nodeID: "resource-recovery"}
	called := false
	_, _, _, err := we.execResource(logging.TestContext(t.Context()), "create", manifest, nil,
		func(context.Context, ...string) ([]byte, error) {
			called = true
			return nil, nil
		})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage deterministic resource manifest")
	assert.False(t, called, "never create a randomly named resource after failing to persist its identity")
}

func TestExecResourcePreservesReadOnlyManifestArtifact(t *testing.T) {
	for _, mode := range []os.FileMode{0o400, 0o444} {
		t.Run(mode.String(), func(t *testing.T) {
			manifest := filepath.Join(t.TempDir(), "manifest.yaml")
			source := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  generateName: recover-\n")
			require.NoError(t, os.WriteFile(manifest, source, mode))
			we := WorkflowExecutor{nodeID: "node", workflowUID: "workflow"}
			var staged string
			_, _, _, err := we.execResource(logging.TestContext(t.Context()), "create", manifest, nil,
				func(_ context.Context, args ...string) ([]byte, error) {
					staged = args[len(args)-3]
					require.NotEqual(t, manifest, staged)
					info, err := os.Stat(staged)
					require.NoError(t, err)
					if runtime.GOOS != "windows" {
						assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
					}
					return []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"recover-me","namespace":"target"}}`), nil
				})
			require.NoError(t, err)
			after, err := os.ReadFile(manifest)
			require.NoError(t, err)
			assert.Equal(t, source, after)
			info, err := os.Stat(manifest)
			require.NoError(t, err)
			if runtime.GOOS != "windows" {
				assert.Equal(t, mode, info.Mode().Perm())
			}
			_, err = os.Stat(staged)
			assert.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestExecResourceRecoveryRejectsAdditionalSourcesBeforeCreate(t *testing.T) {
	for _, flags := range [][]string{
		{"-f", "other.yaml"}, {"-fother.yaml"}, {"--filename=other.yaml"},
		{"-Rfother.yaml"}, {"-k", "directory"}, {"--kustomize=directory"},
		{"--raw=/api/v1/namespaces/default/configmaps"},
	} {
		t.Run(flags[0], func(t *testing.T) {
			manifest := filepath.Join(t.TempDir(), "manifest.yaml")
			require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: recover-me\n"), 0o600))
			we := WorkflowExecutor{nodeID: "node", workflowUID: "workflow", resourceTemplateRecovering: true}
			called := false
			_, _, _, err := we.execResource(logging.TestContext(t.Context()), "create", manifest, flags,
				func(context.Context, ...string) ([]byte, error) {
					called = true
					return nil, nil
				})
			require.Error(t, err)
			assert.False(t, called, "unmarked additional inputs must not be replayed")
		})
	}
}

func TestExecResourceRecoveryRejectsEmptyManifestBeforeCreate(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, nil, 0o600))
	we := WorkflowExecutor{nodeID: "node", workflowUID: "workflow", resourceTemplateRecovering: true}
	called := false
	_, _, _, err := we.execResource(logging.TestContext(t.Context()), "create", manifest, nil,
		func(context.Context, ...string) ([]byte, error) {
			called = true
			return nil, nil
		})
	require.ErrorContains(t, err, "empty manifest")
	assert.False(t, called)
}

func TestAlreadyExistsErrorUsesKubectlStderr(t *testing.T) {
	assert.True(t, isAlreadyExistsErr(&exec.ExitError{
		Stderr: []byte(`Error from server (AlreadyExists): configmaps "recover-me" already exists`),
	}))
	assert.False(t, isAlreadyExistsErr(&exec.ExitError{
		Stderr: []byte("Error from server (Forbidden): access denied"),
	}))
}

func TestResourceManifestIdentityIsScopedToWorkflowRun(t *testing.T) {
	prepare := func(workflowUID types.UID) unstructured.Unstructured {
		manifest := filepath.Join(t.TempDir(), "manifest.yaml")
		require.NoError(t, os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  generateName: recover-\n  annotations:\n    custom: preserved\n"), 0o600))
		we := WorkflowExecutor{nodeID: "same-workflow-name", workflowUID: workflowUID}
		firstPath, err := we.prepareResourceManifest(manifest)
		require.NoError(t, err)
		defer func() { _ = os.Remove(firstPath) }()
		first, err := os.ReadFile(firstPath)
		require.NoError(t, err)
		secondPath, err := we.prepareResourceManifest(manifest)
		require.NoError(t, err)
		defer func() { _ = os.Remove(secondPath) }()
		second, err := os.ReadFile(secondPath)
		require.NoError(t, err)
		assert.Equal(t, first, second, "replacement executors preserve the resource's identity")
		var obj unstructured.Unstructured
		require.NoError(t, yaml.Unmarshal(first, &obj))
		assert.Equal(t, "preserved", obj.GetAnnotations()["custom"])
		assert.Equal(t, string(workflowUID), obj.GetAnnotations()[common.AnnotationKeyWorkflowUID])
		assert.Equal(t, we.nodeID, obj.GetAnnotations()[common.AnnotationKeyNodeID])
		return obj
	}
	first := prepare("first-workflow")
	second := prepare("second-workflow")
	assert.NotEqual(t, first.GetName(), second.GetName(), "reusing a workflow name must not reuse an earlier run's resource")
}
