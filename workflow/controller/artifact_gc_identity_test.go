package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestArtifactGCSeparatesSameNamedTemplates(t *testing.T) {
	archiveLocation := func(bucket, secret string) *wfv1.ArtifactLocation {
		return &wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{
			S3Bucket: wfv1.S3Bucket{
				Endpoint: "minio:9000",
				Bucket:   bucket,
				AccessKeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secret},
					Key:                  "accesskey",
				},
				SecretKeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secret},
					Key:                  "secretkey",
				},
			},
			Key: "archive",
		}}
	}
	first := wfv1.Template{Name: "main", ArchiveLocation: archiveLocation("first-bucket", "first-secret")}
	second := wfv1.Template{Name: "main", ArchiveLocation: archiveLocation("second-bucket", "second-secret")}
	node := func(id, resource string) wfv1.NodeStatus {
		return wfv1.NodeStatus{
			ID:          id,
			Name:        "template-identity." + id,
			Type:        wfv1.NodeTypePod,
			Phase:       wfv1.NodeSucceeded,
			TemplateRef: &wfv1.TemplateRef{Name: resource, Template: "main"},
			Outputs: &wfv1.Outputs{Artifacts: wfv1.Artifacts{{
				Name:             "result",
				ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: id + "/result"}},
			}}},
		}
	}
	useLocalTemplate := func(wf *wfv1.Workflow) {
		localNode := wf.Status.Nodes["first"]
		localNode.TemplateName = "main"
		localNode.TemplateRef = nil
		localNode.TemplateScope = "local/" + wf.Name
		wf.Status.Nodes["first"] = localNode
		delete(wf.Status.StoredTemplates, "namespaced/first/main")
	}

	for _, tt := range []struct {
		name            string
		setup           func(*wfv1.Workflow) *wfv1.WorkflowTemplate
		missingTemplate string
	}{
		{name: "two WorkflowTemplates"},
		{
			name: "local and referenced",
			setup: func(wf *wfv1.Workflow) *wfv1.WorkflowTemplate {
				useLocalTemplate(wf)
				wf.Spec.Templates = []wfv1.Template{first}
				return nil
			},
		},
		{
			name: "namespaced and cluster",
			setup: func(wf *wfv1.Workflow) *wfv1.WorkflowTemplate {
				clusterNode := wf.Status.Nodes["second"]
				clusterNode.TemplateRef = &wfv1.TemplateRef{Name: "first", Template: "main", ClusterScope: true}
				wf.Status.Nodes["second"] = clusterNode
				delete(wf.Status.StoredTemplates, "namespaced/second/main")
				wf.Status.StoredTemplates["cluster/first/main"] = second
				return nil
			},
		},
		{
			name: "inherited template scopes",
			setup: func(wf *wfv1.Workflow) *wfv1.WorkflowTemplate {
				for id, n := range wf.Status.Nodes {
					n.TemplateName = "main"
					n.TemplateRef = nil
					n.TemplateScope = "namespaced/" + id
					wf.Status.Nodes[id] = n
				}
				return nil
			},
		},
		{
			name: "workflowTemplateRef and referenced",
			setup: func(wf *wfv1.Workflow) *wfv1.WorkflowTemplate {
				useLocalTemplate(wf)
				wf.Spec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "source"}
				wf.Status.Phase = wfv1.WorkflowSucceeded
				return &wfv1.WorkflowTemplate{
					ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: wf.Namespace},
					Spec: wfv1.WorkflowSpec{
						Entrypoint: "main",
						Templates:  []wfv1.Template{first},
					},
				}
			},
		},
		{
			name: "missing referenced template",
			setup: func(wf *wfv1.Workflow) *wfv1.WorkflowTemplate {
				wf.Spec.Templates = []wfv1.Template{second}
				delete(wf.Status.StoredTemplates, "namespaced/second/main")
				return nil
			},
			missingTemplate: "namespaced/second/main",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "template-identity", Namespace: "default", UID: "workflow-uid"},
				Spec: wfv1.WorkflowSpec{
					ArtifactGC: &wfv1.WorkflowLevelArtifactGC{ArtifactGC: wfv1.ArtifactGC{Strategy: wfv1.ArtifactGCOnWorkflowCompletion}},
				},
				Status: wfv1.WorkflowStatus{
					StoredTemplates: map[string]wfv1.Template{
						"namespaced/first/main":  first,
						"namespaced/second/main": second,
					},
					Nodes: wfv1.Nodes{
						"first":  node("first", "first"),
						"second": node("second", "second"),
					},
				},
			}
			options := []any{wf}
			var source *wfv1.WorkflowTemplate
			if tt.setup != nil {
				source = tt.setup(wf)
				if source != nil {
					options = append(options, source)
				}
			}
			cancel, controller := newController(ctx, options...)
			defer cancel()
			assignTaskUIDs(controller)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.wf.Status.ArtifactGCStatus = &wfv1.ArtGCStatus{}
			if source != nil {
				var err error
				ctx, err = woc.setExecWorkflow(ctx)
				require.NoError(t, err)
				require.NotNil(t, woc.wf.Status.StoredWorkflowSpec)
			}

			strategy := wfv1.ArtifactGCOnWorkflowCompletion
			err := woc.processArtifactGCStrategy(ctx, strategy)
			if tt.missingTemplate != "" {
				require.ErrorContains(t, err, tt.missingTemplate)
			} else {
				require.NoError(t, err)
			}

			tasks, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(wf.Namespace).List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			pods, err := controller.kubeclientset.CoreV1().Pods(wf.Namespace).List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			if tt.missingTemplate != "" {
				assert.False(t, woc.wf.Status.ArtifactGCStatus.IsArtifactGCStrategyProcessed(strategy))
				assert.Empty(t, tasks.Items)
				assert.Empty(t, pods.Items)
				return
			}

			require.Len(t, tasks.Items, 1)
			require.Len(t, tasks.Items[0].Spec.ArtifactsByNode, 2)
			assert.Equal(t, first.ArchiveLocation, tasks.Items[0].Spec.ArtifactsByNode["first"].ArchiveLocation)
			assert.Equal(t, second.ArchiveLocation, tasks.Items[0].Spec.ArtifactsByNode["second"].ArchiveLocation)
			require.Len(t, pods.Items, 1)
			var secretNames []string
			for _, volume := range pods.Items[0].Spec.Volumes {
				if volume.Secret != nil {
					secretNames = append(secretNames, volume.Secret.SecretName)
				}
			}
			assert.ElementsMatch(t, []string{"first-secret", "second-secret"}, secretNames)
		})
	}
}
