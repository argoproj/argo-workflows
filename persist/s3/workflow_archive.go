package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/minio/minio-go/v7"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

type workflowArchive struct {
	storage
}

func NewWorkflowArchive(secretInterface corev1.SecretInterface, clusterName string, config config.S3ArtifactRepository, migrate bool) (*workflowArchive, error) {
	s, err := newStorage(secretInterface, clusterName, config)
	if err != nil {
		return nil, err
	}
	if migrate {
		log.WithFields(log.Fields{"bucket": s.bucket}).Info("Archiving workflows to S3")
	}
	return &workflowArchive{*s}, nil
}

func (a *workflowArchive) ArchiveWorkflow(wf *wfv1.Workflow) error {
	data, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	// should be idempotent replace
	// write meta-data can allow us to change key in the future
	_, err = a.client.PutObject(
		context.Background(),
		a.bucket,
		a.objectName(string(wf.UID)),
		bytes.NewBuffer(data),
		int64(len(data)),
		minio.PutObjectOptions{
			UserMetadata: map[string]string{
				// meta-data must be HTTP header format
				"Namespace":   wf.Namespace,
				"Name":        wf.Name,
				"Uid":         string(wf.UID),
				"Started-At":  wf.Status.StartedAt.Format(time.RFC3339),
				"Finished-At": wf.Status.FinishedAt.Format(time.RFC3339),
				"Labels":      labels.Set(wf.Labels).String(),
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (a *workflowArchive) objectName(uid string) string {
	return fmt.Sprintf("%s/%v-workflow.json", a.prefix, uid)
}

func (a *workflowArchive) ListWorkflows(namespace string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int) (wfv1.Workflows, error) {
	out := make(wfv1.Workflows, 0)
WorkflowForEach:
	for object := range a.client.ListObjects(
		context.Background(),
		a.bucket,
		minio.ListObjectsOptions{WithMetadata: true, Prefix: a.prefix + "/", Recursive: true},
	) {
		userMetadata := object.UserMetadata
		if namespace != "" && userMetadata["X-Amz-Meta-Namespace"] != namespace {
			continue
		}
		// we treat missing started at as zero
		startedAt, _ := time.Parse(time.RFC3339, userMetadata["X-Amz-Meta-Started-At"])
		if !minStartAt.IsZero() && startedAt.Before(minStartAt) {
			continue
		}
		if !maxStartAt.IsZero() && startedAt.After(maxStartAt) {
			continue
		}
		labels, _ := labels.ConvertSelectorToLabelsMap(userMetadata["X-Amz-Meta-Labels"])
		for _, r := range labelRequirements {
			if !r.Matches(labels) {
				continue WorkflowForEach
			}
		}
		finishedAt, _ := time.Parse(time.RFC3339, userMetadata["X-Amz-Meta-Finished-At"])
		out = append(out, wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userMetadata["X-Amz-Meta-Name"],
				Namespace: userMetadata["X-Amz-Meta-Namespace"],
				UID:       types.UID(userMetadata["X-Amz-Meta-Uid"]),
			},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.NodePhase(labels.Get(common.LabelKeyPhase)),
				StartedAt:  metav1.NewTime(startedAt),
				FinishedAt: metav1.NewTime(finishedAt),
			},
		})
	}
	sort.Sort(out)
	if offset > 0 && offset < len(out) {
		out = out[offset:]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (a *workflowArchive) GetWorkflow(uid string) (*wfv1.Workflow, error) {
	object, err := a.client.GetObject(context.Background(), a.bucket, a.objectName(uid), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	wf := &wfv1.Workflow{}
	err = json.NewDecoder(object).Decode(&wf)
	if noSuchKeyErr(err) { // yes, this can be returned from Decode!
		return nil, nil
	}
	return wf, err
}

func (a *workflowArchive) DeleteWorkflow(uid string) error {
	wf, err := a.GetWorkflow(uid)
	if err != nil {
		return err
	}
	if wf == nil {
		return nil
	}
	return a.client.RemoveObject(context.Background(), a.bucket, a.objectName(uid), minio.RemoveObjectOptions{})
}

func (a *workflowArchive) Run(<-chan struct{}) {}

func (a *workflowArchive) IsEnabled() bool {
	return true
}
