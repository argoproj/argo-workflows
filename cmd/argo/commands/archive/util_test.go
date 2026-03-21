package archive

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestIsUID(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		forceUID  bool
		forceName bool
		want      bool
	}{
		{
			name:      "Valid UID",
			s:         "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			forceUID:  false,
			forceName: false,
			want:      true,
		},
		{
			name:      "Valid UID with forceUID",
			s:         "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			forceUID:  true,
			forceName: false,
			want:      true,
		},
		{
			name:      "Valid UID with forceName",
			s:         "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			forceUID:  false,
			forceName: true,
			want:      false,
		},
		{
			name:      "Invalid UID (Name)",
			s:         "my-workflow",
			forceUID:  false,
			forceName: false,
			want:      false,
		},
		{
			name:      "Invalid UID with forceUID",
			s:         "my-workflow",
			forceUID:  true,
			forceName: false,
			want:      true,
		},
		{
			name:      "Invalid UID with forceName",
			s:         "my-workflow",
			forceUID:  false,
			forceName: true,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isUID(tt.s, tt.forceUID, tt.forceName))
		})
	}
}

type mockArchivedWorkflowServiceClient struct {
	mock.Mock
}

func (m *mockArchivedWorkflowServiceClient) ListArchivedWorkflows(ctx context.Context, in *workflowarchivepkg.ListArchivedWorkflowsRequest, opts ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wfv1.WorkflowList), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) GetArchivedWorkflow(ctx context.Context, in *workflowarchivepkg.GetArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*wfv1.Workflow), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) DeleteArchivedWorkflow(ctx context.Context, in *workflowarchivepkg.DeleteArchivedWorkflowRequest, opts ...grpc.CallOption) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*workflowarchivepkg.ArchivedWorkflowDeletedResponse), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) ListArchivedWorkflowLabelKeys(ctx context.Context, in *workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest, opts ...grpc.CallOption) (*wfv1.LabelKeys, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*wfv1.LabelKeys), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) ListArchivedWorkflowLabelValues(ctx context.Context, in *workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest, opts ...grpc.CallOption) (*wfv1.LabelValues, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*wfv1.LabelValues), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) RetryArchivedWorkflow(ctx context.Context, in *workflowarchivepkg.RetryArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*wfv1.Workflow), args.Error(1)
}

func (m *mockArchivedWorkflowServiceClient) ResubmitArchivedWorkflow(ctx context.Context, in *workflowarchivepkg.ResubmitArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*wfv1.Workflow), args.Error(1)
}

func TestResolveUID(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		forceUID   bool
		forceName  bool
		mockSetup  func(*mockArchivedWorkflowServiceClient)
		wantUID    string
		wantErr    string
	}{
		{
			name:       "Already UID",
			identifier: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			mockSetup:  func(m *mockArchivedWorkflowServiceClient) {},
			wantUID:    "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		},
		{
			name:       "Name with single match",
			identifier: "my-wf",
			mockSetup: func(m *mockArchivedWorkflowServiceClient) {
				m.On("ListArchivedWorkflows", mock.Anything, mock.MatchedBy(func(req *workflowarchivepkg.ListArchivedWorkflowsRequest) bool {
					return req.NamePrefix == "my-wf" && req.NameFilter == "Exact"
				}), mock.Anything).Return(&wfv1.WorkflowList{
					Items: wfv1.Workflows{{
						ObjectMeta: metav1.ObjectMeta{UID: "uid-1", Name: "my-wf"},
					}},
				}, nil)
			},
			wantUID: "uid-1",
		},
		{
			name:       "Name with no match",
			identifier: "my-wf",
			mockSetup: func(m *mockArchivedWorkflowServiceClient) {
				m.On("ListArchivedWorkflows", mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.WorkflowList{
					Items: wfv1.Workflows{},
				}, nil)
			},
			wantErr: "archived workflow 'my-wf' not found",
		},
		{
			name:       "Name with multiple matches",
			identifier: "my-wf",
			mockSetup: func(m *mockArchivedWorkflowServiceClient) {
				m.On("ListArchivedWorkflows", mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.WorkflowList{
					Items: wfv1.Workflows{
						{ObjectMeta: metav1.ObjectMeta{UID: "uid-1", Name: "my-wf"}},
						{ObjectMeta: metav1.ObjectMeta{UID: "uid-2", Name: "my-wf"}},
					},
				}, nil)
			},
			wantErr: "Multiple archived workflows found with name 'my-wf'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockArchivedWorkflowServiceClient{}
			tt.mockSetup(m)
			got, err := resolveUID(context.Background(), m, tt.identifier, "default", tt.forceUID, tt.forceName)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUID, got)
			}
		})
	}
}
