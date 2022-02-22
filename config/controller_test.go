package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_parseConfigMap(t *testing.T) {
	cc := controller{emptyConfigFunc: EmptyConfigFunc}
	t.Run("Empty", func(t *testing.T) {
		_, err := cc.parseConfigMap(&apiv1.ConfigMap{})
		assert.NoError(t, err)
	})
	t.Run("Config", func(t *testing.T) {
		c, err := cc.parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"config": "containerRuntimeExecutor: pns"}})
		if assert.NoError(t, err) {
			assert.Equal(t, "pns", c.(*Config).ContainerRuntimeExecutor)
		}
	})
	t.Run("Complex", func(t *testing.T) {
		c, err := cc.parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"containerRuntimeExecutor": "pns", "artifactRepository": `    archiveLogs: true
    s3:
      bucket: my-bucket
      endpoint: minio:9000
      insecure: true
      accessKeySecret:
        name: my-minio-cred
        key: accesskey
      secretKeySecret:
        name: my-minio-cred
        key: secretkey`}})
		if assert.NoError(t, err) {
			assert.Equal(t, "pns", c.(*Config).ContainerRuntimeExecutor)
			assert.NotEmpty(t, c.(*Config).ArtifactRepository)
		}
	})
	t.Run("IgnoreGarbage", func(t *testing.T) {
		_, err := cc.parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"garbage": "garbage"}})
		assert.NoError(t, err)
	})
}

func Test_controller_Get(t *testing.T) {
	kube := fake.NewSimpleClientset()
	c := controller{configMap: "my-config-map", kubeclientset: kube, emptyConfigFunc: EmptyConfigFunc}
	ctx := context.Background()
	config, err := c.Get(ctx)
	if assert.NoError(t, err) {
		assert.Empty(t, config)
	}
}
