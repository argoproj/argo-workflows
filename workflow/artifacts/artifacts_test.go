package executor

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/s3"
)

type mockResourceInterface struct{}

func (*mockResourceInterface) GetSecret(ctx context.Context, name, key string) (string, error) {
	// Mock getSecret that doesn't actually read from a store, just return a different value to assert it was called
	return key + "-secret", nil
}

func (*mockResourceInterface) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	// We don't need GetConfigMapKey for initialising the driver
	return "", nil
}

func TestNewDriverS3(t *testing.T) {
	art := &wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{
			S3Bucket: wfv1.S3Bucket{
				Endpoint: "endpoint",
				Bucket:   "bucket",
				Region:   "us-east-1",
				AccessKeySecret: &apiv1.SecretKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: "accesskey",
					},
					Key: "access-key",
				},
				SecretKeySecret: &apiv1.SecretKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: "secretkey",
					},
					Key: "secret-key",
				},
				SessionTokenSecret: &apiv1.SecretKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: "sessiontoken",
					},
					Key: "session-token",
				},
			},
			Key: "art",
		}},
	}

	got, err := newDriver(logging.TestContext(t.Context()), art, &mockResourceInterface{})
	require.NoError(t, err)

	artDriver := got.(*s3.ArtifactDriver)
	assert.Equal(t, art.S3.AccessKeySecret.Key+"-secret", artDriver.AccessKey)
	assert.Equal(t, art.S3.SecretKeySecret.Key+"-secret", artDriver.SecretKey)
	assert.Equal(t, art.S3.SessionTokenSecret.Key+"-secret", artDriver.SessionToken)
}
