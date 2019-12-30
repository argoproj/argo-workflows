package artifactory_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	art "github.com/argoproj/argo/workflow/artifacts/artifactory"
)

const (
	LoadFileName string = "argo_artifactory_test_load.txt"
	SaveFileName string = "argo_artifactory_test_save.txt"
	RepoName     string = "generic-local"
	URL          string = "http://localhost:8081/artifactory/" + RepoName + "/" + LoadFileName
	Username     string = "admin"
	Password     string = "password"
)

func TestSaveAndLoad(t *testing.T) {

	t.Skip("This test is skipped since it depends on external service")
	fileContent := "time: " + string(time.Now().UnixNano())

	// create file to test save
	lf, err := ioutil.TempFile("", LoadFileName)
	assert.NoError(t, err)
	defer os.Remove(lf.Name())
	// load file with test content
	content := []byte(fileContent)
	_, err = lf.Write(content)
	assert.NoError(t, err)
	err = lf.Close()
	assert.NoError(t, err)

	// create file to test load
	sf, err := ioutil.TempFile("", SaveFileName)
	assert.NoError(t, err)
	defer os.Remove(sf.Name())

	artL := &wfv1.Artifact{}
	artL.Artifactory = &wfv1.ArtifactoryArtifact{
		URL: URL,
	}
	driver := &art.ArtifactoryArtifactDriver{
		Username: Username,
		Password: Password,
	}
	err = driver.Save(lf.Name(), artL)
	assert.NoError(t, err)
	err = driver.Load(artL, sf.Name())
	assert.NoError(t, err)

	dat, err := ioutil.ReadFile(sf.Name())
	assert.NoError(t, err)
	assert.Equal(t, fileContent, string(dat))
}
