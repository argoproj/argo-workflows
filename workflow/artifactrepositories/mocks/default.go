package mocks

import (
	"github.com/stretchr/testify/mock"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func DummyArtifactRepositories(repo *wfv1.ArtifactRepository) *Interface {
	i := &Interface{}
	i.On("Resolve", mock.Anything, mock.Anything, mock.Anything).Return(&wfv1.ArtifactRepositoryRefStatus{}, nil)
	i.On("Get", mock.Anything, mock.Anything).Return(repo, nil)
	return i
}
