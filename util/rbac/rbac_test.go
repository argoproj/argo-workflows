package rbac

import (
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestRBAC_AccessClusterWorkflowTemplates(t *testing.T) {
	t.Run("has full access", func(t *testing.T) {
		kubeclientset := &kubefake.Clientset{}
		allowedVerbs := []string{"get", "list", "watch"}
		kubeclientset.AddReactor("create", "selfsubjectaccessreviews", reactionFuncWithAllowedVerbs(allowedVerbs))
		ctx := logging.TestContext(t.Context())
		allowed := HasAccessToClusterWorkflowTemplates(ctx, kubeclientset)
		assert.True(t, allowed)
	})

	t.Run("only has get and list access", func(t *testing.T) {
		kubeclientset := &kubefake.Clientset{}
		allowedVerbs := []string{"get", "list"}
		kubeclientset.AddReactor("create", "selfsubjectaccessreviews", reactionFuncWithAllowedVerbs(allowedVerbs))
		ctx := logging.TestContext(t.Context())
		allowed := HasAccessToClusterWorkflowTemplates(ctx, kubeclientset)
		assert.False(t, allowed)
	})

	t.Run("only has get access", func(t *testing.T) {
		kubeclientset := &kubefake.Clientset{}
		allowedVerbs := []string{"get"}
		kubeclientset.AddReactor("create", "selfsubjectaccessreviews", reactionFuncWithAllowedVerbs(allowedVerbs))
		ctx := logging.TestContext(t.Context())
		allowed := HasAccessToClusterWorkflowTemplates(ctx, kubeclientset)
		assert.False(t, allowed)
	})
}

func reactionFuncWithAllowedVerbs(allowedVerbs []string) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		selfSubjectAccessReview := reflect.ValueOf(action).FieldByName("Object").Elem().Elem().Field(2).Field(0).Elem()
		resource := selfSubjectAccessReview.FieldByName("Resource").String()
		verb := selfSubjectAccessReview.FieldByName("Verb").String()
		allowed := resource == "clusterworkflowtemplates" && slices.Contains(allowedVerbs, verb)
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	}
}
