package artifactory_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	art "github.com/argoproj/argo/workflow/artifacts/artifactory"
	"github.com/stretchr/testify/assert"
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
	assert.Nil(t, err)
	defer os.Remove(lf.Name())
	// load file with test content
	content := []byte(fileContent)
	_, err = lf.Write(content)
	assert.Nil(t, err)
	err = lf.Close()
	assert.Nil(t, err)

	// create file to test load
	sf, err := ioutil.TempFile("", SaveFileName)
	assert.Nil(t, err)
	defer os.Remove(sf.Name())

	artL := &wfv1.Artifact{}
	artL.Artifactory = &wfv1.ArtifactoryArtifact{
		URL: URL,
	}
	driver := &art.ArtifactoryArtifactDriver{
		Username: Username,
		Password: Password,
	}
	driver.Save(lf.Name(), artL)
	driver.Load(artL, sf.Name())

	dat, err := ioutil.ReadFile(sf.Name())
	assert.Nil(t, err)
	assert.Equal(t, fileContent, string(dat))
}
