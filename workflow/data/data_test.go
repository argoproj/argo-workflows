package data

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type testDataSourceProcessor struct{}

var _ v1alpha1.DataSourceProcessor = &testDataSourceProcessor{}

func (t testDataSourceProcessor) ProcessArtifactPaths(*v1alpha1.ArtifactPaths) (interface{}, error) {
	return []interface{}{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}, nil
}

type nullTestDataSourceProcessor struct{}

var _ v1alpha1.DataSourceProcessor = &nullTestDataSourceProcessor{}

func (t nullTestDataSourceProcessor) ProcessArtifactPaths(*v1alpha1.ArtifactPaths) (interface{}, error) {
	return nil, fmt.Errorf("cannot get artifacts")
}

func TestProcessSource(t *testing.T) {
	artifactPathsSource := v1alpha1.DataSource{ArtifactPaths: &v1alpha1.ArtifactPaths{}}
	data, err := processSource(artifactPathsSource, &testDataSourceProcessor{})
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}, data)
	}

	_, err = processSource(artifactPathsSource, &nullTestDataSourceProcessor{})
	assert.Error(t, err)

	_, err = processSource(v1alpha1.DataSource{}, &nullTestDataSourceProcessor{})
	assert.Error(t, err)
}

func TestProcessTransformation(t *testing.T) {
	files := []interface{}{"foo.py", "bar.pdf", "goo/foo.py", "moo/bar.pdf"}

	filterFiles := &v1alpha1.Transformation{{Expression: `filter(data, {# endsWith '.py'})`}}
	filtered, err := processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"foo.py", "goo/foo.py"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains '/'})`}}
	filtered, err = processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"goo/foo.py", "moo/bar.pdf"}, filtered)
	}
	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains 'foo'})`}}
	filtered, err = processTransformation(filtered, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"goo/foo.py"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {# contains '/'})`}, {Expression: `filter(data, {# contains 'foo'})`}}
	filtered, err = processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"goo/foo.py"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}}
	filtered, err = processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"foo.py", "bar.pdf"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `map(data, {# + '.processed'})`}}
	filtered, err = processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"foo.py.processed", "bar.pdf.processed", "goo/foo.py.processed", "moo/bar.pdf.processed"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}, {Expression: `map(data, {# + '.processed'})`}}
	filtered, err = processTransformation(files, filterFiles)
	if assert.NoError(t, err) {
		assert.Equal(t, []interface{}{"foo.py.processed", "bar.pdf.processed"}, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `filter(data, {not(# contains '/')})`}, {Expression: `map(data, {# + '.processed'})`}, {}}
	_, err = processTransformation(files, filterFiles)
	assert.NoError(t, err)

	filtered, err = processTransformation(files, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, files, filtered)
	}

	filtered, err = processTransformation(files, &v1alpha1.Transformation{})
	if assert.NoError(t, err) {
		assert.Equal(t, files, filtered)
	}

	filterFiles = &v1alpha1.Transformation{{Expression: `map(data, {# + '.processed'}`}}
	_, err = processTransformation(files, filterFiles)
	assert.Error(t, err)
}
