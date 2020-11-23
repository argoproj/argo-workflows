package s3

import (
	"os"

	argos3 "github.com/argoproj/pkg/s3"
	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/workflow/common"
)

type storage struct {
	bucket string
	prefix string
	client *minio.Client
}

func newStorage(secretInterface corev1.SecretInterface, clusterName string, config config.S3ArtifactRepository) (*storage, error) {
	secret, err := secretInterface.Get(config.AccessKeySecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	accessKey := secret.Data[config.AccessKeySecret.Key]
	secret, err = secretInterface.Get(config.SecretKeySecret.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	secretKey := secret.Data[config.SecretKeySecret.Key]
	opts := argos3.S3ClientOpts{
		Endpoint:    config.Endpoint,
		Region:      config.Region,
		Secure:      config.Secure(),
		AccessKey:   string(accessKey),
		SecretKey:   string(secretKey),
		RoleARN:     config.RoleARN,
		Trace:       os.Getenv(common.EnvVarArgoTrace) == "1",
		UseSDKCreds: config.UseSDKCreds,
	}
	credentials, err := argos3.GetCredentials(opts)
	if err != nil {
		return nil, err
	}
	minioOpts := &minio.Options{Creds: credentials, Secure: opts.Secure, Region: opts.Region}
	client, err := minio.New(opts.Endpoint, minioOpts)
	if err != nil {
		return nil, err
	}
	if opts.Trace {
		client.TraceOn(log.StandardLogger().Out)
	}
	return &storage{client: client, bucket: config.Bucket, prefix: clusterName}, nil
}

func noSuchKeyErr(err error) bool {
	switch v := err.(type) {
	case minio.ErrorResponse:
		return v.Code == "NoSuchKey"
	}
	return false
}
