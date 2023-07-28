package executor

import (
	"context"
	"fmt"
	gohttp "net/http"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/azure"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/gcs"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/git"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/hdfs"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/http"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/oss"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/raw"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/s3"
)

var ErrUnsupportedDriver = fmt.Errorf("unsupported artifact driver")

type NewDriverFunc func(ctx context.Context, art *wfv1.Artifact, ri resource.Interface) (common.ArtifactDriver, error)

// NewDriver initializes an instance of an artifact driver
func NewDriver(ctx context.Context, art *wfv1.Artifact, ri resource.Interface) (common.ArtifactDriver, error) {
	drv, err := newDriver(ctx, art, ri)
	if err != nil {
		return nil, err
	}
	return logging.New(drv), nil

}
func newDriver(ctx context.Context, art *wfv1.Artifact, ri resource.Interface) (common.ArtifactDriver, error) {
	if art.S3 != nil {
		var accessKey string
		var secretKey string
		var serverSideCustomerKey string
		var kmsKeyId string
		var kmsEncryptionContext string
		var enableEncryption bool

		if art.S3.AccessKeySecret != nil && art.S3.AccessKeySecret.Name != "" {
			accessKeyBytes, err := ri.GetSecret(ctx, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = accessKeyBytes
			secretKeyBytes, err := ri.GetSecret(ctx, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = secretKeyBytes
		}

		if art.S3.EncryptionOptions != nil {
			if art.S3.EncryptionOptions.ServerSideCustomerKeySecret != nil {
				if art.S3.EncryptionOptions.KmsKeyId != "" {
					return nil, fmt.Errorf("serverSideCustomerKeySecret and kmsKeyId cannot be set together")
				}

				serverSideCustomerKeyBytes, err := ri.GetSecret(ctx, art.S3.EncryptionOptions.ServerSideCustomerKeySecret.Name, art.S3.EncryptionOptions.ServerSideCustomerKeySecret.Key)
				if err != nil {
					return nil, err
				}
				serverSideCustomerKey = serverSideCustomerKeyBytes
			}

			enableEncryption = art.S3.EncryptionOptions.EnableEncryption
			kmsKeyId = art.S3.EncryptionOptions.KmsKeyId
			kmsEncryptionContext = art.S3.EncryptionOptions.KmsEncryptionContext
		}

		driver := s3.ArtifactDriver{
			Endpoint:              art.S3.Endpoint,
			AccessKey:             accessKey,
			SecretKey:             secretKey,
			Secure:                art.S3.Insecure == nil || !*art.S3.Insecure,
			Region:                art.S3.Region,
			RoleARN:               art.S3.RoleARN,
			UseSDKCreds:           art.S3.UseSDKCreds,
			KmsKeyId:              kmsKeyId,
			KmsEncryptionContext:  kmsEncryptionContext,
			EnableEncryption:      enableEncryption,
			ServerSideCustomerKey: serverSideCustomerKey,
		}

		return &driver, nil
	}
	if art.HTTP != nil {
		var client *gohttp.Client
		driver := http.ArtifactDriver{}
		if art.HTTP.Auth != nil && art.HTTP.Auth.BasicAuth.UsernameSecret != nil {
			usernameBytes, err := ri.GetSecret(ctx, art.HTTP.Auth.BasicAuth.UsernameSecret.Name, art.HTTP.Auth.BasicAuth.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			driver.Username = usernameBytes
		}
		if art.HTTP.Auth != nil && art.HTTP.Auth.BasicAuth.PasswordSecret != nil {
			passwordBytes, err := ri.GetSecret(ctx, art.HTTP.Auth.BasicAuth.PasswordSecret.Name, art.HTTP.Auth.BasicAuth.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			driver.Password = passwordBytes
		}
		if art.HTTP.Auth != nil && art.HTTP.Auth.OAuth2.ClientIDSecret != nil && art.HTTP.Auth.OAuth2.ClientSecretSecret != nil && art.HTTP.Auth.OAuth2.TokenURLSecret != nil {
			clientId, err := ri.GetSecret(ctx, art.HTTP.Auth.OAuth2.ClientIDSecret.Name, art.HTTP.Auth.OAuth2.ClientIDSecret.Key)
			if err != nil {
				return nil, err
			}
			clientSecret, err := ri.GetSecret(ctx, art.HTTP.Auth.OAuth2.ClientSecretSecret.Name, art.HTTP.Auth.OAuth2.ClientSecretSecret.Key)
			if err != nil {
				return nil, err
			}
			tokenURL, err := ri.GetSecret(ctx, art.HTTP.Auth.OAuth2.TokenURLSecret.Name, art.HTTP.Auth.OAuth2.TokenURLSecret.Key)
			if err != nil {
				return nil, err
			}
			client = http.CreateOauth2Client(clientId, clientSecret, tokenURL, art.HTTP.Auth.OAuth2.Scopes, art.HTTP.Auth.OAuth2.EndpointParams)
		}
		if art.HTTP.Auth != nil && art.HTTP.Auth.ClientCert.ClientCertSecret != nil && art.HTTP.Auth.ClientCert.ClientKeySecret != nil {
			clientCert, err := ri.GetSecret(ctx, art.HTTP.Auth.ClientCert.ClientCertSecret.Name, art.HTTP.Auth.ClientCert.ClientCertSecret.Key)
			if err != nil {
				return nil, err
			}
			clientKey, err := ri.GetSecret(ctx, art.HTTP.Auth.ClientCert.ClientKeySecret.Name, art.HTTP.Auth.ClientCert.ClientKeySecret.Key)
			if err != nil {
				return nil, err
			}
			client, err = http.CreateClientWithCertificate([]byte(clientCert), []byte(clientKey))
			if err != nil {
				return nil, err
			}
		}
		if client == nil {
			client = &gohttp.Client{}
		}
		driver.Client = client
		return &driver, nil
	}
	if art.Git != nil {
		gitDriver := git.ArtifactDriver{
			InsecureIgnoreHostKey: art.Git.InsecureIgnoreHostKey,
			DisableSubmodules:     art.Git.DisableSubmodules,
		}
		if art.Git.UsernameSecret != nil {
			usernameBytes, err := ri.GetSecret(ctx, art.Git.UsernameSecret.Name, art.Git.UsernameSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Username = usernameBytes
		}
		if art.Git.PasswordSecret != nil {
			passwordBytes, err := ri.GetSecret(ctx, art.Git.PasswordSecret.Name, art.Git.PasswordSecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.Password = passwordBytes
		}
		if art.Git.SSHPrivateKeySecret != nil {
			sshPrivateKeyBytes, err := ri.GetSecret(ctx, art.Git.SSHPrivateKeySecret.Name, art.Git.SSHPrivateKeySecret.Key)
			if err != nil {
				return nil, err
			}
			gitDriver.SSHPrivateKey = sshPrivateKeyBytes
		}

		return &gitDriver, nil
	}
	if art.Artifactory != nil {
		usernameBytes, err := ri.GetSecret(ctx, art.Artifactory.UsernameSecret.Name, art.Artifactory.UsernameSecret.Key)
		if err != nil {
			return nil, err
		}
		passwordBytes, err := ri.GetSecret(ctx, art.Artifactory.PasswordSecret.Name, art.Artifactory.PasswordSecret.Key)
		if err != nil {
			return nil, err
		}
		driver := http.ArtifactDriver{
			Username: usernameBytes,
			Password: passwordBytes,
			Client:   &gohttp.Client{},
		}
		return &driver, nil

	}
	if art.HDFS != nil {
		return hdfs.CreateDriver(ctx, ri, art.HDFS)
	}
	if art.Raw != nil {
		return &raw.ArtifactDriver{}, nil
	}

	if art.OSS != nil {
		var accessKey string
		var secretKey string

		if !art.OSS.UseSDKCreds && art.OSS.AccessKeySecret != nil && art.OSS.AccessKeySecret.Name != "" {
			accessKeyBytes, err := ri.GetSecret(ctx, art.OSS.AccessKeySecret.Name, art.OSS.AccessKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accessKey = string(accessKeyBytes)
			secretKeyBytes, err := ri.GetSecret(ctx, art.OSS.SecretKeySecret.Name, art.OSS.SecretKeySecret.Key)
			if err != nil {
				return nil, err
			}
			secretKey = string(secretKeyBytes)
		}

		driver := oss.ArtifactDriver{
			Endpoint:      art.OSS.Endpoint,
			AccessKey:     accessKey,
			SecretKey:     secretKey,
			SecurityToken: art.OSS.SecurityToken,
			UseSDKCreds:   art.OSS.UseSDKCreds,
		}
		return &driver, nil
	}

	if art.GCS != nil {
		driver := gcs.ArtifactDriver{}
		if art.GCS.ServiceAccountKeySecret != nil && art.GCS.ServiceAccountKeySecret.Name != "" {
			serviceAccountKeyBytes, err := ri.GetSecret(ctx, art.GCS.ServiceAccountKeySecret.Name, art.GCS.ServiceAccountKeySecret.Key)
			if err != nil {
				return nil, err
			}
			serviceAccountKey := string(serviceAccountKeyBytes)
			driver.ServiceAccountKey = serviceAccountKey
		}
		// key is not set, assume it is using Workload Idendity
		return &driver, nil
	}

	if art.Azure != nil {
		var accountKey string

		if !art.Azure.UseSDKCreds && art.Azure.AccountKeySecret != nil && art.Azure.AccountKeySecret.Name != "" {
			accountKeyBytes, err := ri.GetSecret(ctx, art.Azure.AccountKeySecret.Name, art.Azure.AccountKeySecret.Key)
			if err != nil {
				return nil, err
			}
			accountKey = accountKeyBytes
		}
		driver := azure.ArtifactDriver{
			AccountKey:  accountKey,
			Container:   art.Azure.Container,
			Endpoint:    art.Azure.Endpoint,
			UseSDKCreds: art.Azure.UseSDKCreds,
		}
		return &driver, nil
	}

	return nil, ErrUnsupportedDriver
}
