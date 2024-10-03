package v1alpha1

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkflowTemplates(t *testing.T) {
	tmpls := WorkflowTemplates{
		{ObjectMeta: v1.ObjectMeta{Name: "1"}},
		{ObjectMeta: v1.ObjectMeta{Name: "2"}},
		{ObjectMeta: v1.ObjectMeta{Name: "0"}},
	}
	sort.Sort(tmpls)
	require.Len(t, tmpls, 3)
	assert.Equal(t, "0", tmpls[0].Name)
	assert.Equal(t, "1", tmpls[1].Name)
	assert.Equal(t, "2", tmpls[2].Name)
}
