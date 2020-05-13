package config

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type ArtifactRepositoryCredential struct {
	Name        string                `json:"name"`
	Artifactory *wfv1.ArtifactoryAuth `json:"artifactory,omitempty"`
	GCS         *wfv1.GCSBucket       `json:"gcs,omitempty"`
	Git         *wfv1.GitArtifact     `json:"git,omitempty"`
	OSS         *wfv1.OSSBucket       `json:"oss,omitempty"`
	S3          *wfv1.S3Bucket        `json:"s3,omitempty"`
}

func (c ArtifactRepositoryCredential) MergeInto(a *wfv1.Artifact) {
	if a == nil {
		return
	}
	c.S3.MergeInto(a.S3)
	// TODO
}

type ArtifactRepositoryCredentials []ArtifactRepositoryCredential

func (cs ArtifactRepositoryCredentials) Find(predicate func(c ArtifactRepositoryCredential) bool) (ArtifactRepositoryCredential, bool) {
	for _, c := range cs {
		if predicate(c) {
			return c, true
		}
	}
	return ArtifactRepositoryCredential{}, false
}

func (cs ArtifactRepositoryCredentials) Merge(as wfv1.Artifacts) (wfv1.Artifacts, error) {
	out := make(wfv1.Artifacts, len(as))
	for i, a := range as {
		if a.CredentialName != "" {
			c, ok := cs.Find(func(c ArtifactRepositoryCredential) bool { return c.Name == a.CredentialName })
			if !ok {
				return nil, fmt.Errorf("could not find credential named \"%s\"", a.CredentialName)
			}
			c.MergeInto(&a)
		}
		out[i] = a
	}
	return out, nil
}
