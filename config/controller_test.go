package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
)

func Test_parseConfigMap(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		c := &Config{}
		err := parseConfigMap(&apiv1.ConfigMap{}, c)
		require.NoError(t, err)
	})
	t.Run("Complex", func(t *testing.T) {
		c := &Config{}
		err := parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"artifactRepository": `    archiveLogs: true
    s3:
      bucket: my-bucket
      endpoint: minio:9000
      insecure: true
      accessKeySecret:
        name: my-minio-cred
        key: accesskey
      secretKeySecret:
        name: my-minio-cred
        key: secretkey`}}, c)
		require.NoError(t, err)
		assert.NotEmpty(t, c.ArtifactRepository)
	})
	t.Run("Garbage", func(t *testing.T) {
		c := &Config{}
		err := parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"garbage": "garbage"}}, c)
		require.Error(t, err)
	})
}
