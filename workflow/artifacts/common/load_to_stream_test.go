//go:build !windows

package common

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type fakeArtifactDriver struct {
	ArtifactDriver
	data []byte
	// mockedErrs is a map where key is the function name and value is the mocked error of that function
	mockedErrs map[string]error
}

func (a *fakeArtifactDriver) getMockedErr(funcName string) error {
	err, ok := a.mockedErrs[funcName]
	if !ok {
		return nil
	}
	return err
}

func (a *fakeArtifactDriver) Load(_ *wfv1.Artifact, path string) error {
	err := a.getMockedErr("Load")
	if err == nil {
		// actually write a file to disk
		_, err := os.Create(path)
		if err != nil {
			panic(fmt.Sprintf("can't create file at path %s", path))
		}
		return nil
	} else {
		return err
	}

}

func (a *fakeArtifactDriver) OpenStream(_ *wfv1.Artifact) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *fakeArtifactDriver) Save(_ string, _ *wfv1.Artifact) error {
	return fmt.Errorf("not implemented")
}

func (a *fakeArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func filteredFiles(t *testing.T) ([]os.DirEntry, error) {
	t.Helper()

	filtered := make([]os.DirEntry, 0)
	entries, err := os.ReadDir("/tmp/")
	if err != nil {
		return filtered, err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), loadToStreamPrefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, err
}

func TestLoadToStream(t *testing.T) {
	tests := map[string]struct {
		artifactDriver ArtifactDriver
		errMsg         string
	}{
		"Success": {
			artifactDriver: &fakeArtifactDriver{
				data:       []byte("my-data"),
				mockedErrs: map[string]error{},
			},
			errMsg: "",
		},
		"LoadFailure": {
			artifactDriver: &fakeArtifactDriver{
				data:       []byte("my-data"),
				mockedErrs: map[string]error{"Load": fmt.Errorf("failed to find file")},
			},
			errMsg: "failed to find file",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			// need to verify that a new file doesn't get written so check the /tmp directory ahead of time
			filesBefore, err := filteredFiles(t)
			if err != nil {
				panic(err)
			}

			stream, err := LoadToStream(&wfv1.Artifact{}, tc.artifactDriver)
			if tc.errMsg == "" {
				require.NoError(t, err)
				assert.NotNil(t, stream)
				stream.Close()

				// make sure the new file got deleted when we called stream.Close() above
				filesAfter, err := filteredFiles(t)
				if err != nil {
					panic(err)
				}
				assert.Equal(t, len(filesBefore), len(filesAfter))
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.errMsg, err.Error())
			}
		})
	}

}
