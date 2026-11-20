package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfsync "github.com/argoproj/argo-workflows/v4/workflow/sync"
)

func TestUpdateNSAppliesParallelismLimit(t *testing.T) {
	throttler := wfsync.NewMultiThrottler(0, 0, func(string) {})
	namespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argo",
			Labels: map[string]string{
				common.LabelParallelismLimit: "1",
			},
		},
	}

	updateNS(
		logging.TestContext(t.Context()),
		namespace,
		throttler.UpdateNamespaceParallelism,
		throttler.ResetNamespaceParallelism,
	)

	created := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	throttler.Add("argo/first", 0, created)
	throttler.Add("argo/second", 0, created.Add(time.Second))

	assert.True(t, throttler.Admit("argo/first"))
	assert.False(t, throttler.Admit("argo/second"))
}
