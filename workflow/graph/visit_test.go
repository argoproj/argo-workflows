package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type testVisitor struct {
	inited bool
	visted []string
}

func (t *testVisitor) Init() {
	t.inited = true
}

func (t *testVisitor) Visit(nodeID string) {
	t.visted = append(t.visted, nodeID)
}

var _ Visitor = &testVisitor{}

func TestVisit(t *testing.T) {
	v := &testVisitor{}
	err := Visit(wfv1.Nodes{
		"root-1": {Children: []string{"child", "missing"}},
		"child":  {},
		"root-2": {},
	}, v)
	if assert.NoError(t, err) {
		assert.True(t, v.inited)
		assert.Equal(t, []string{"root-2", "child", "root-1"}, v.visted, "we visit all nodes except missing")
	}
}
