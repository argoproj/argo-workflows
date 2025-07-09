package auth

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestCanI(t *testing.T) {
	kubeClient := &kubefake.Clientset{}

	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		selfSubjectAccessReview := reflect.ValueOf(action).FieldByName("Object").Elem().Elem().Field(2).Field(0).Elem()
		resource := selfSubjectAccessReview.FieldByName("Resource").String()
		verb := selfSubjectAccessReview.FieldByName("Verb").String()
		allowed := resource == "workflow" && verb == "get"
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	allowed, err := CanIArgo(ctx, kubeClient, "get", "workflow", "", "")
	require.NoError(t, err)
	assert.True(t, allowed)
	notAllowed, err := CanIArgo(ctx, kubeClient, "list", "workflow", "", "")
	require.NoError(t, err)
	assert.False(t, notAllowed)
}
