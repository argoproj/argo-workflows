package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceDirPath(t *testing.T) {
	newPath, err := ReplaceDirPath("/foo", "/foo", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo")

	newPath, err = ReplaceDirPath("/foo", "/foo/", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo")

	newPath, err = ReplaceDirPath("/foo/", "/foo", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo")

	newPath, err = ReplaceDirPath("/foo/", "/foo/", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo")

	newPath, err = ReplaceDirPath("/foo/bar", "/foo", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo/bar")

	newPath, err = ReplaceDirPath("/foo/bar", "/foo/", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo/bar")

	newPath, err = ReplaceDirPath("/foo/bar/", "/foo", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo/bar")

	newPath, err = ReplaceDirPath("/foo/bar/", "/foo/", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/argo/bar")

	newPath, err = ReplaceDirPath("/foo/bar", "/xxx", "/argo")
	assert.Nil(t, err)
	assert.Equal(t, newPath, "/foo/bar")
}
