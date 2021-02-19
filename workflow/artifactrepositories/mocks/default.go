package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func DummyArtifactRepositories(repo *config.ArtifactRepository) *Interface {
	i := &Interface{}
	i.On("Resolve", mock.Anything, mock.Anything, mock.Anything).Return(wfv1.DefaultArtifactRepositoryRefStatus, nil)
	i.On("Get", mock.Anything, mock.Anything).Return(repo, nil)
	return i
}
