package namespacedefaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

// configMap builds a ConfigMap carrying the label the controller's typed informer
// selects on, since discovery is by label rather than by a well-known name.
func configMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapWorkflowDefaults,
			},
		},
		Data: data,
	}
}

// indexerFor builds the same index the controller's typed ConfigMap informer uses, so
// these tests exercise the real lookup rather than a stub.
func indexerFor(t *testing.T, cms ...*corev1.ConfigMap) func() cache.Indexer {
	t.Helper()
	idx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		indexes.ConfigMapLabelsIndex: indexes.ConfigMapIndexFunc,
	})
	for _, cm := range cms {
		require.NoError(t, idx.Add(cm))
	}
	return func() cache.Indexer { return idx }
}

func TestNamespaceDefaults(t *testing.T) {
	t.Run("NoConfigMap", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())

		wf, err := New(indexerFor(t), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, wf, "a namespace without a labelled ConfigMap configures no defaults")
	})

	t.Run("Defaults", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAccountName: my-sa\n  entrypoint: my-entrypoint\n",
		})

		wf, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, wf)
		assert.Equal(t, "my-sa", wf.Spec.ServiceAccountName)
		assert.Equal(t, "my-entrypoint", wf.Spec.Entrypoint)
	})

	t.Run("OtherNamespaceIsIgnored", func(t *testing.T) {
		// ConfigMapLabelsIndex is keyed on the label value alone, so the lookup sees
		// every namespace's ConfigMaps and has to filter. Without that filter another
		// namespace's defaults would leak into this one.
		ctx := logging.TestContext(t.Context())
		other := configMap("other-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAccountName: not-mine\n",
		})

		wf, err := New(indexerFor(t, other), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, wf, "defaults from another namespace must not apply")
	})

	t.Run("StatusAndControllerOwnedMetadataAreDropped", func(t *testing.T) {
		// MergeTo copies the whole object, so a defaults document must not be able to
		// carry controller-owned state onto every Workflow in the namespace.
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "any-name", map[string]string{
			Key: "metadata:\n" +
				"  labels:\n    team: platform\n" +
				"  annotations:\n    owner: platform\n" +
				"  finalizers:\n  - workflows.argoproj.io/bogus\n" +
				"  ownerReferences:\n  - apiVersion: v1\n    kind: ConfigMap\n    name: evil\n    uid: abc\n" +
				"status:\n  phase: Succeeded\n" +
				"spec:\n  entrypoint: my-entrypoint\n",
		})

		wf, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, wf)

		assert.Empty(t, wf.Status.Phase, "a defaults document must not set status")
		assert.Empty(t, wf.Finalizers, "a defaults document must not set finalizers")
		assert.Empty(t, wf.OwnerReferences, "a defaults document must not set ownerReferences")

		assert.Equal(t, "platform", wf.Labels["team"], "labels are a supported default")
		assert.Equal(t, "platform", wf.Annotations["owner"], "annotations are a supported default")
		assert.Equal(t, "my-entrypoint", wf.Spec.Entrypoint)
	})
}

func TestNamespaceDefaultsFailLoudly(t *testing.T) {
	t.Run("MissingKey", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "any-name", map[string]string{
			"workflowDefault": "spec:\n  entrypoint: my-entrypoint\n",
		})

		_, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err, "a ConfigMap with a mistyped key must fail rather than silently apply nothing")
		assert.Contains(t, err.Error(), "missing key")
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "any-name", map[string]string{
			Key: "spec: [this is not a workflow spec",
		})

		_, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("MisspelledField", func(t *testing.T) {
		// Strict decoding, to match config/controller.go. Without it this parses fine
		// and applies nothing, which is the failure mode the feature exists to avoid.
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAcccountName: my-sa\n",
		})

		_, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err, "a misspelled field must be rejected, not ignored")
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("MoreThanOneConfigMap", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		first := configMap("my-ns", "a-defaults", map[string]string{
			Key: "spec:\n  serviceAccountName: from-a\n",
		})
		second := configMap("my-ns", "b-defaults", map[string]string{
			Key: "spec:\n  serviceAccountName: from-b\n",
		})

		_, err := New(indexerFor(t, first, second), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err, "which of two labelled ConfigMaps wins is not something to arbitrate")
		assert.Contains(t, err.Error(), "a-defaults")
		assert.Contains(t, err.Error(), "b-defaults")
	})

	t.Run("ControllerOwnedLabel", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cm := configMap("my-ns", "my-defaults", map[string]string{
			Key: "metadata:\n  labels:\n    " + common.LabelKeyCompleted + ": \"true\"\n",
		})

		_, err := New(indexerFor(t, cm), indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err, "a completed label from defaults would stop every workflow in the namespace running")
		assert.Contains(t, err.Error(), common.LabelKeyCompleted)
		assert.Contains(t, err.Error(), "my-defaults")
	})

	t.Run("InformerNotRunning", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())

		_, err := New(func() cache.Indexer { return nil }, indexes.ConfigMapLabelsIndex).Get(ctx, "my-ns")
		require.Error(t, err, "silently applying no defaults is the failure mode to avoid")
	})
}

// NewLister is what the argo-server uses, since it has no ConfigMap informer. It has to
// agree with the indexed lookup on every rule, so these mirror the tests above.
func TestNamespaceDefaultsFromLister(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset(configMap("my-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAccountName: my-sa\n",
		}))

		wf, err := NewLister(k).Get(ctx, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, wf)
		assert.Equal(t, "my-sa", wf.Spec.ServiceAccountName)
	})

	t.Run("NoConfigMap", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())

		wf, err := NewLister(kubefake.NewClientset()).Get(ctx, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, wf)
	})

	t.Run("UnlabelledConfigMapIsIgnored", func(t *testing.T) {
		// Discovery is by label, so an unlabelled ConfigMap must not be picked up no
		// matter what it is called.
		ctx := logging.TestContext(t.Context())
		unlabelled := configMap("my-ns", "workflow-defaults", map[string]string{
			Key: "spec:\n  serviceAccountName: should-not-apply\n",
		})
		unlabelled.Labels = nil

		wf, err := NewLister(kubefake.NewClientset(unlabelled)).Get(ctx, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, wf, "only labelled ConfigMaps are namespace defaults")
	})

	t.Run("MoreThanOneConfigMap", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset(
			configMap("my-ns", "a-defaults", map[string]string{Key: "spec:\n  serviceAccountName: from-a\n"}),
			configMap("my-ns", "b-defaults", map[string]string{Key: "spec:\n  serviceAccountName: from-b\n"}),
		)

		_, err := NewLister(k).Get(ctx, "my-ns")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "a-defaults")
		assert.Contains(t, err.Error(), "b-defaults")
	})

	t.Run("MisspelledField", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset(configMap("my-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAcccountName: my-sa\n",
		}))

		_, err := NewLister(k).Get(ctx, "my-ns")
		require.Error(t, err, "the server must reject a misspelling exactly as the controller does")
	})

	t.Run("ControllerOwnedLabel", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset(configMap("my-ns", "any-name", map[string]string{
			Key: "metadata:\n  labels:\n    " + common.LabelKeyCompleted + ": \"true\"\n",
		}))

		_, err := NewLister(k).Get(ctx, "my-ns")
		require.Error(t, err, "the server must reject a controller-owned label exactly as the controller does")
		assert.Contains(t, err.Error(), common.LabelKeyCompleted)
	})
	t.Run("EmptyNamespaceDoesNotListEveryNamespace", func(t *testing.T) {
		// A List with an empty namespace spans the cluster, so without a guard a caller
		// that has not resolved its namespace would pick up another namespace's defaults,
		// or trip the at-most-one rule on ConfigMaps it has nothing to do with.
		ctx := logging.TestContext(t.Context())
		k := kubefake.NewClientset(
			configMap("ns-a", "a-defaults", map[string]string{Key: "spec:\n  serviceAccountName: from-a\n"}),
			configMap("ns-b", "b-defaults", map[string]string{Key: "spec:\n  serviceAccountName: from-b\n"}),
		)

		wf, err := NewLister(k).Get(ctx, "")
		require.NoError(t, err, "an empty namespace must not trip the at-most-one rule")
		assert.Nil(t, wf)
	})
}

// Merged is what puts namespace defaults above controller defaults, and both the
// controller and the server rely on it for that ordering.
func TestMerged(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controllerDefaults := &wfv1.Workflow{Spec: wfv1.WorkflowSpec{
		ServiceAccountName: "from-controller",
		Entrypoint:         "from-controller",
	}}

	t.Run("NamespaceWinsAndControllerFillsTheRest", func(t *testing.T) {
		i := New(indexerFor(t, configMap("my-ns", "any-name", map[string]string{
			Key: "spec:\n  serviceAccountName: from-namespace\n",
		})), indexes.ConfigMapLabelsIndex)

		merged, err := Merged(ctx, i, controllerDefaults, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, merged)
		assert.Equal(t, "from-namespace", merged.Spec.ServiceAccountName, "namespace defaults must win")
		assert.Equal(t, "from-controller", merged.Spec.Entrypoint, "controller defaults must still fill the rest")
	})

	t.Run("NeitherLayerConfiguresAnything", func(t *testing.T) {
		merged, err := Merged(ctx, New(indexerFor(t), indexes.ConfigMapLabelsIndex), nil, "my-ns")
		require.NoError(t, err)
		assert.Nil(t, merged)
	})

	t.Run("NilInterfaceIsTolerated", func(t *testing.T) {
		merged, err := Merged(ctx, nil, controllerDefaults, "my-ns")
		require.NoError(t, err)
		require.NotNil(t, merged)
		assert.Equal(t, "from-controller", merged.Spec.ServiceAccountName)
	})
}
