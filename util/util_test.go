package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestGenerateFieldSelectorFromWorkflowName(t *testing.T) {
	type args struct {
		wfName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"TestGenerateFieldSelectorFromWorkflowName", args{"whalesay"}, "metadata.name=whalesay"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateFieldSelectorFromWorkflowName(tt.args.wfName); got != tt.want {
				t.Errorf("GenerateFieldSelectorFromWorkflowName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverWorkflowNameFromSelectorString(t *testing.T) {
	type args struct {
		selector string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"TestRecoverWorkflowNameFromSelectorString", args{"metadata.name=whalesay"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name="}, ""},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name=whalesay,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=whalesay,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=whalesay"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name= whalesay ,other=hello"}, "whalesay"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,other=hello"}, ""},
		{"TestRecoverWorkflowNameFromSelectorString", args{"metadata.name=@latest"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name="}, ""},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"metadata.name=@latest,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=@latest,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name=@latest"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,metadata.name= @latest ,other=hello"}, "@latest"},
		{"TestRecoverWorkflowNameFromSelectorStringEmptyWf", args{"foo=bar,other=hello"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecoverWorkflowNameFromSelectorStringIfAny(tt.args.selector)
			if got != tt.want {
				t.Errorf("RecoverWorkflowNameFromSelectorStringIfAny() = %v, want %v", got, tt.want)
			}
		})
	}
	name := RecoverWorkflowNameFromSelectorStringIfAny("whatever=whalesay")
	assert.Equal(t, "", name)
	assert.NotPanics(t, func() {
		_ = RecoverWorkflowNameFromSelectorStringIfAny("whatever")
	})
}

func TestGetDeletePropagation(t *testing.T) {
	t.Run("GetDefaultPolicy", func(t *testing.T) {
		assert.Equal(t, metav1.DeletePropagationBackground, *GetDeletePropagation())
	})
	t.Run("GetEnvPolicy", func(t *testing.T) {
		t.Setenv("WF_DEL_PROPAGATION_POLICY", "Foreground")
		assert.Equal(t, metav1.DeletePropagationForeground, *GetDeletePropagation())
	})
	t.Run("GetEnvPolicyWithEmpty", func(t *testing.T) {
		t.Setenv("WF_DEL_PROPAGATION_POLICY", "")
		assert.Equal(t, metav1.DeletePropagationBackground, *GetDeletePropagation())
	})
}

func TestMergeArtifacts(t *testing.T) {
	type args struct {
		artifactSlices [][]wfv1.Artifact
	}
	tests := []struct {
		name string
		args args
		want []wfv1.Artifact
	}{
		{
			name: "test merge artifacts",
			args: args{
				artifactSlices: [][]wfv1.Artifact{
					{
						{
							Name: "artifact1",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{
									S3Bucket: wfv1.S3Bucket{
										Bucket: "bucket1",
									},
								},
							},
						},
						{
							Name: "artifact2",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{
									S3Bucket: wfv1.S3Bucket{
										Bucket: "bucket2",
									},
								},
							},
						},
					},
					{
						{
							Name: "artifact3",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{
									S3Bucket: wfv1.S3Bucket{
										Bucket: "bucket3",
									},
								},
							},
						},
						{
							Name: "artifact1",
							ArtifactLocation: wfv1.ArtifactLocation{
								S3: &wfv1.S3Artifact{

									S3Bucket: wfv1.S3Bucket{
										Bucket: "bucket1",
									},
								},
							},
						},
					},
				},
			},
			want: []wfv1.Artifact{
				{
					Name: "artifact1",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket: "bucket1",
							},
						},
					},
				},
				{
					Name: "artifact2",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket: "bucket2",
							},
						},
					},
				},
				{
					Name: "artifact3",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Bucket: "bucket3",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, MergeArtifacts(tt.args.artifactSlices...), "MergeArtifacts(%v)", tt.args.artifactSlices)
		})
	}
}

func TestMergeParameters(t *testing.T) {
	type args struct {
		params [][]wfv1.Parameter
	}
	tests := []struct {
		name string
		args args
		want []wfv1.Parameter
	}{
		{
			name: "test merge parameters",
			args: args{
				params: [][]wfv1.Parameter{
					{
						{
							Name:  "param1",
							Value: wfv1.AnyStringPtr("value1"),
						},
						{
							Name:  "param2",
							Value: wfv1.AnyStringPtr("value2"),
						},
					},
					{
						{
							Name:  "param3",
							Value: wfv1.AnyStringPtr("value3"),
						},
						{
							Name:  "param2",
							Value: wfv1.AnyStringPtr("value2"),
						},
					},
				},
			},
			want: []wfv1.Parameter{
				{
					Name:  "param1",
					Value: wfv1.AnyStringPtr("value1"),
				},
				{
					Name:  "param2",
					Value: wfv1.AnyStringPtr("value2"),
				},
				{
					Name:  "param3",
					Value: wfv1.AnyStringPtr("value3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, MergeParameters(tt.args.params...), "MergeParameters(%v)", tt.args.params)
		})
	}
}
