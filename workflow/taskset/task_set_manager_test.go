package taskset

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

func TestUpdateTaskSet(t *testing.T) {
	config, err := clientcmd.DefaultClientConfig.ClientConfig()
	assert.NoError(t, err)

	wfclientset := wfclientset.NewForConfigOrDie(config)

	taskSet, err := wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("argo").Get(context.Background(), "steps-5j6px", v1.GetOptions{})
	fmt.Println(taskSet, err)
}
