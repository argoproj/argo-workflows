package retention

import (
	"context"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

var retentionGCPeriod = envutil.LookupEnvDurationOr("RETENTION_GC_PERIOD", time.Minute)

type Interface interface {
	Run(ctx context.Context)
}

type retention struct {
	policy            config.RetentionPolicy
	wfInformer        cache.SharedIndexInformer
	workflowInterface workflow.Interface
}

func (r retention) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			objs, err := r.wfInformer.GetIndexer().ByIndex(indexes.WorkflowCompletedIndex, "true")
			if err != nil {
				panic(err)
			}
			var uns []*unstructured.Unstructured
			for _, obj := range objs {
				un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					panic("nok")
				}
				uns = append(uns, un)
			}
			// sort youngest ... oldest - we priorities keeping newer workflows
			sort.Slice(uns, func(i, j int) bool {
				return uns[i].GetCreationTimestamp().Time.After(uns[j].GetCreationTimestamp().Time)
			})
			retain := make(map[types.UID]bool) // which one we should retain

			// firstly, we prioritise keeping errored and failed
			failed, errored := 0, 0
			for _, un := range uns {
				if failed >= r.policy.Failed && errored >= r.policy.Errored {
					break
				}
				switch wfv1.WorkflowPhase(un.GetLabels()[common.LabelKeyPhase]) {
				case wfv1.WorkflowError:
					if errored < r.policy.Errored {
						errored++
						retain[un.GetUID()] = true
					}
				case wfv1.WorkflowFailed:
					if failed < r.policy.Failed {
						failed++
						retain[un.GetUID()] = true
					}
				}
			}
			// the we add any completed until we have enough
			for _, un := range uns {
				if len(retain) >= r.policy.Completed {
					break
				}
				retain[un.GetUID()] = true
			}
			log.WithFields(log.Fields{
				"policy":    log.Fields{"failed": r.policy.Failed, "errored": r.policy.Errored, "completed": r.policy.Completed},
				"retention": log.Fields{"failed": failed, "errored": errored, "completed": len(retain)},
				"total":     len(uns),
			}).Info("Performing retention GC")
			for _, un := range uns {
				println("ALEX", "retain", un.GetCreationTimestamp().String(), un.GetName(), retain[un.GetUID()])
				if retain[un.GetUID()] {
					continue
				}
				err := r.workflowInterface.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Delete(ctx, un.GetName(), metav1.DeleteOptions{
					PropagationPolicy: util.GetDeletePropagation(),
				})
				if err != nil && !apierr.IsNotFound(err) {
					log.WithError(err).Warn("failed to delete workflow for retention")
				}
			}
			time.Sleep(retentionGCPeriod)
		}
	}
}

func New(f config.RetentionPolicy, i cache.SharedIndexInformer, wi workflow.Interface) Interface {
	return &retention{f, i, wi}
}
