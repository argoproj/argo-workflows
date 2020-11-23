package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type offloadNodeStatusRepo struct {
	storage
	ttl time.Duration
}

func NewOffloadNodeStatusRepo(secretInterface corev1.SecretInterface, clusterName string, config config.S3ArtifactRepository, migrate bool, ttl time.Duration) (*offloadNodeStatusRepo, error) {
	s, err := newStorage(secretInterface, clusterName, config)
	if err != nil {
		return nil, err
	}
	if migrate {
		log.WithFields(log.Fields{"bucket": s.bucket, "ttl": ttl}).Info("Offloading node status to S3")
	}
	return &offloadNodeStatusRepo{*s, ttl}, nil
}

func (r *offloadNodeStatusRepo) Save(uid, namespace string, nodes wfv1.Nodes) (string, error) {
	marshalled, version, err := persist.NodeStatusVersion(nodes)
	if err != nil {
		return "", err
	}
	_, err = r.client.PutObject(
		context.Background(),
		r.bucket,
		r.objectName(uid, version),
		bytes.NewBufferString(marshalled),
		int64(len(marshalled)),
		minio.PutObjectOptions{
			UserMetadata: map[string]string{
				// must be HTTP header format
				"Namespace": namespace,
				"Uid":       uid,
				"Version":   version,
			},
		},
	)
	if err != nil {
		return "", err
	}
	return version, nil
}

func (r *offloadNodeStatusRepo) objectName(uid, version string) string {
	return r.prefix + "/" + uid + "-" + version + "-node-status.json"
}

func (r *offloadNodeStatusRepo) Get(uid, version string) (wfv1.Nodes, error) {
	object, err := r.client.GetObject(
		context.Background(),
		r.bucket,
		r.objectName(uid, version),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	nodes := wfv1.Nodes{}
	return nodes, json.NewDecoder(object).Decode(&nodes)
}

func (r *offloadNodeStatusRepo) List(namespace string) (map[persist.UUIDVersion]wfv1.Nodes, error) {
	out := make(map[persist.UUIDVersion]wfv1.Nodes)
	for object := range r.client.ListObjects(context.Background(), r.bucket, minio.ListObjectsOptions{WithMetadata: true}) {
		userMetadata := object.UserMetadata
		if userMetadata["X-Amz-Meta-Namespace"] != namespace {
			continue
		}
		uid := userMetadata["X-Amz-Meta-Uid"]
		version := userMetadata["X-Amz-Meta-Version"]
		nodes, err := r.Get(uid, version)
		if err != nil {
			return nil, err
		}
		out[persist.UUIDVersion{UID: uid, Version: version}] = nodes
	}
	return out, nil
}

func (r *offloadNodeStatusRepo) ListOldOffloads(string) ([]persist.UUIDVersion, error) {
	var out []persist.UUIDVersion
	for object := range r.client.ListObjects(context.Background(), r.bucket, minio.ListObjectsOptions{WithMetadata: true}) {
		if object.LastModified.After(time.Now().Add(r.ttl)) {
			continue
		}
		uid := object.UserMetadata["X-Amz-Meta-Uid"]
		version := object.UserMetadata["X-Amz-Meta-Version"]
		out = append(out, persist.UUIDVersion{UID: uid, Version: version})
	}
	return out, nil
}

func (r *offloadNodeStatusRepo) Delete(uid, version string) error {
	return r.client.RemoveObject(
		context.Background(),
		r.bucket,
		r.objectName(uid, version),
		minio.RemoveObjectOptions{VersionID: version},
	)
}

func (r *offloadNodeStatusRepo) IsEnabled() bool {
	return true
}
