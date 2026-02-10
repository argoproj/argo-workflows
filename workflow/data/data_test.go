package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type testDataSourceProcessor struct{}

var _ v1alpha1.DataSourceProcessor = &testDataSourceProcessor{}

func (t testDataSourceProcessor) ProcessArtifactPaths(context.Context, *v1alpha1.ArtifactPaths) (any, error) {
	return []any{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}, nil
}

type nullTestDataSourceProcessor struct{}

var _ v1alpha1.DataSourceProcessor = &nullTestDataSourceProcessor{}

func (t nullTestDataSourceProcessor) ProcessArtifactPaths(context.Context, *v1alpha1.ArtifactPaths) (any, error) {
	return nil, fmt.Errorf("cannot get artifacts")
}

func TestProcessSource(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	artifactPathsSource := v1alpha1.DataSource{ArtifactPaths: &v1alpha1.ArtifactPaths{}}
	data, err := processSource(ctx, artifactPathsSource, &testDataSourceProcessor{})
	require.NoError(t, err)
	assert.Equal(t, []any{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}, data)

	_, err = processSource(ctx, artifactPathsSource, &nullTestDataSourceProcessor{})
	require.Error(t, err)

	_, err = processSource(ctx, v1alpha1.DataSource{}, &nullTestDataSourceProcessor{})
	require.Error(t, err)
}

func TestProcessTransformation(t *testing.T) {
	files := []any{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}

	filterFiles := &v1alpha1.Transformation{{Expression: `filter(data, {# endsWith '.py'})`}}
	filtered, err := processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"foo.py", "goo/foo.py"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains '/'})`}}
	filtered, err = processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"goo/foo.py", "moo/bar.pdf"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains 'foo'})`}}
	filtered, err = processTransformation(filtered, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"goo/foo.py"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains '/'})`}, {Expression: `filter(data, {# contains 'foo'})`}}
	filtered, err = processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"goo/foo.py"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}}
	filtered, err = processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"foo.py", "bar.pdf"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `map(data, {# + '.processed'})`}}
	filtered, err = processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"foo.py.processed", "bar.pdf.processed", "goo/foo.py.processed", "moo/bar.pdf.processed"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}, {Expression: `map(data, {# + '.processed'})`}}
	filtered, err = processTransformation(files, filterFiles)
	require.NoError(t, err)
	assert.Equal(t, []any{"foo.py.processed", "bar.pdf.processed"}, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}, {Expression: `map(data, {# + '.processed'})`}, {}}
	_, err = processTransformation(files, filterFiles)
	require.NoError(t, err)

	filtered, err = processTransformation(files, nil)
	require.NoError(t, err)
	assert.Equal(t, files, filtered)

	filtered, err = processTransformation(files, &v1alpha1.Transformation{})
	require.NoError(t, err)
	assert.Equal(t, files, filtered)

	filterFiles = &v1alpha1.Transformation{{Expression: `map(data, {# + '.processed'}`}}
	_, err = processTransformation(files, filterFiles)
	require.Error(t, err)
}
