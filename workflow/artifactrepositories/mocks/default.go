package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var DummyArtifactRepositories = &Interface{}

func init() {
	DummyArtifactRepositories.On("Resolve", mock.Anything, mock.Anything).Return(wfv1.DefaultArtifactRepositoryRef, nil)
	DummyArtifactRepositories.On("Get", mock.Anything).Return(&config.ArtifactRepository{}, nil)
}
