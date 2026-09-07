package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiClusterDocsExist(t *testing.T) {
	docPath := filepath.Join("..", "docs", "multi-cluster-workflows.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)

	// Check file exists and is not empty
	assert.Greater(t, len(content), 0, "multi-cluster-workflows.md should not be empty")

	// Check for AI disclosure line per CNCF policy
	assert.Contains(t, string(content), "AI assistance", "should contain AI disclosure")
}

func TestMultiClusterDocsHasProperNavigation(t *testing.T) {
	properdocsPath := filepath.Join("..", "properdocs.yml")
	content, err := os.ReadFile(properdocsPath)
	require.NoError(t, err)

	// Check that multi-cluster-workflows.md is referenced in Features navigation
	assert.Contains(t, string(content), "multi-cluster-workflows.md", "should reference multi-cluster-workflows.md in properdocs.yml")
}

func TestMultiClusterDocsValidMarkdown(t *testing.T) {
	docPath := filepath.Join("..", "docs", "multi-cluster-workflows.md")
	content, err := os.ReadFile(docPath)
	require.NoError(t, err)

	// Check that it's valid markdown (basic check)
	assert.True(t, strings.Contains(string(content), "#"), "should have heading")
	assert.True(t, strings.Contains(string(content), "`"), "should have code examples")
}
