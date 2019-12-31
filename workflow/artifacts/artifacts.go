package executor

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/artifacts/artifactory"
	"github.com/argoproj/argo/workflow/artifacts/git"
	"github.com/argoproj/argo/workflow/artifacts/hdfs"
	"github.com/argoproj/argo/workflow/artifacts/http"
	"github.com/argoproj/argo/workflow/artifacts/raw"
	"github.com/argoproj/argo/workflow/artifacts/resource"
	"github.com/argoproj/argo/workflow/artifacts/s3"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// Load accepts an artifact source URL and places it at specified path
	Load(inputArtifact *wfv1.Artifact, path string) error

	// Save uploads the path to artifact destination
	Save(path string, outputArtifact *wfv1.Artifact) error
}

var ErrUnsupportedDriver = fmt.Errorf("unsupported artifact driver")

// NewDriver initializes an instance of an artifact driver
func NewDriver(art *wfv1.Artifact, ri resource.Interface) (ArtifactDriver, error) {
	if art.S3 != nil {
		var accessKey string
		var secretKey string

		if art.S3.AccessKeySecret.Name != "" {
			accessKeyBytes, err := ri.GetSecret(art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = accessKeyBytes
			secretKeyBytes, err := ri.GetSecret(art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = secretKeyBytes
		}

		driver := s3.S3ArtifactDriver{
			Endpoint:  art.S3.Endpoint,
			AccessKey: accessKey,
			SecretKey: secretKey,
			Secure:    art.S3.Insecure == nil || !*art.S3.Insecure,
			Region:    art.S3.Region,
			RoleARN:   art.S3.RoleARN,
		}
		return &driver, nil
	}
	if art.HTTP != nil {
		return &http.HTTPArtifactDriver{}, nil
	}
	if art.Git != nil {
		gitDriver := git.GitArtifactDriver{
			InsecureIgnoreHostKey: art.Git.InsecureIgnoreHostKey,
		}
		if art.Git.UsernameSecret != nil {
			usernameBytes, err := ri.GetSecret(art.Git.UsernameSecret.Name, art.Git.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Username = usernameBytes
		}
		if art.Git.PasswordSecret != nil {
			passwordBytes, err := ri.GetSecret(art.Git.PasswordSecret.Name, art.Git.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Password = passwordBytes
		}
		if art.Git.SSHPrivateKeySecret != nil {
			sshPrivateKeyBytes, err := ri.GetSecret(art.Git.SSHPrivateKeySecret.Name, art.Git.SSHPrivateKeySecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.SSHPrivateKey = sshPrivateKeyBytes
		}

		return &gitDriver, nil
	}
	if art.Artifactory != nil {
		usernameBytes, err := ri.GetSecret(art.Artifactory.UsernameSecret.Name, art.Artifactory.UsernameSecret.Key)
		if err != nil {
			return nil, err
		}
		passwordBytes, err := ri.GetSecret(art.Artifactory.PasswordSecret.Name, art.Artifactory.PasswordSecret.Key)
		if err != nil {
			return nil, err
		}
		driver := artifactory.ArtifactoryArtifactDriver{
			Username: usernameBytes,
			Password: passwordBytes,
		}
		return &driver, nil

	}
	if art.HDFS != nil {
		return hdfs.CreateDriver(ri, art.HDFS)
	}
	if art.Raw != nil {
		return &raw.RawArtifactDriver{}, nil
	}

	return nil, ErrUnsupportedDriver
}
