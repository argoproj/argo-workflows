package sqldb

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func Test_archivedWorkflowMetadata_argumentsUnmarshal(t *testing.T) {
	tests := []struct {
		name          string
		argumentsJSON string
		wantParams    int
		wantFirstName string
	}{
		{
			name:          "empty arguments",
			argumentsJSON: `{}`,
			wantParams:    0,
		},
		{
			name:          "with parameters",
			argumentsJSON: `{"parameters":[{"name":"message","value":"hello world"},{"name":"env","value":"production"}]}`,
			wantParams:    2,
			wantFirstName: "message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := archivedWorkflowMetadata{Arguments: tt.argumentsJSON}
			arguments := wfv1.Arguments{}
			err := json.Unmarshal([]byte(md.Arguments), &arguments)
			require.NoError(t, err)
			assert.Len(t, arguments.Parameters, tt.wantParams)
			if tt.wantParams > 0 {
				assert.Equal(t, tt.wantFirstName, arguments.Parameters[0].Name)
			}
		})
	}
}

func Test_sortListedWorkflows(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	wfs := []archivedWorkflowMetadata{
		{UID: "b", StartedAt: base},
		{UID: "d", StartedAt: base.Add(time.Minute)},
		{UID: "c", StartedAt: base},
		{UID: "a", StartedAt: base},
	}
	sortListedWorkflows(wfs)
	uids := make([]string, len(wfs))
	for i, wf := range wfs {
		uids[i] = wf.UID
	}
	// Newest first, then uid ascending, as BuildArchivedWorkflowSelector orders them.
	assert.Equal(t, []string{"d", "a", "b", "c"}, uids)
}
