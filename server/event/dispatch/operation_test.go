package dispatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/auth/jws"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/common"
)

func TestOperation(t *testing.T) {
	client := fake.NewSimpleClientset(&wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "my-template", Namespace: "my-ns"},
	})
	keyLister := &cache.FakeCustomStore{
		ListKeysFunc: func() []string { return []string{"my-ns/my-template"} },
	}
	event, err := wfv1.ParseItem(`{"type":"test"}`)
	assert.NoError(t, err)
	ctx := context.WithValue(context.WithValue(context.TODO(), auth.WfKey, client), auth.ClaimSetKey, &jws.ClaimSet{Sub: "my-sub"})
	operation := NewOperation(ctx, instanceid.NewService("my-instanceid"), keyLister, &event)

	t.Run("EventSpecMissing", func(t *testing.T) {
		_, err := operation.submitWorkflowFromWorkflowTemplate("my-ns", "my-template")
		assert.EqualError(t, err, "event spec is missing (should be impossible)")
	})

	t.Run("MalformedExpression", func(t *testing.T) {
		_, err := client.ArgoprojV1alpha1().WorkflowTemplates("my-ns").Update(&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-template", Namespace: "my-ns"},
			Spec:       wfv1.WorkflowTemplateSpec{Event: &wfv1.Event{Expression: "="}},
		})
		assert.NoError(t, err)
		_, err = operation.submitWorkflowFromWorkflowTemplate("my-ns", "my-template")
		assert.EqualError(t, err, "failed to evaluate workflow template expression: unexpected token Operator(\"=\") (1:1)\n | =\n | ^")
	})

	t.Run("NonBooleanExpression", func(t *testing.T) {
		_, err := client.ArgoprojV1alpha1().WorkflowTemplates("my-ns").Update(&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-template", Namespace: "my-ns"},
			Spec:       wfv1.WorkflowTemplateSpec{Event: &wfv1.Event{Expression: "invalid"}},
		})
		assert.NoError(t, err)
		_, err = operation.submitWorkflowFromWorkflowTemplate("my-ns", "my-template")
		assert.EqualError(t, err, "malformed workflow template expression: did not evaluate to boolean")
	})

	t.Run("UnMatchedExpression", func(t *testing.T) {
		_, err := client.ArgoprojV1alpha1().WorkflowTemplates("my-ns").Update(&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-template", Namespace: "my-ns"},
			Spec:       wfv1.WorkflowTemplateSpec{Event: &wfv1.Event{Expression: "false"}},
		})
		assert.NoError(t, err)
		wf, err := operation.submitWorkflowFromWorkflowTemplate("my-ns", "my-template")
		if assert.NoError(t, err) {
			assert.Nil(t, wf, "no workflow created")
		}
	})

	t.Run("MatchedExpression", func(t *testing.T) {
		_, err := client.ArgoprojV1alpha1().WorkflowTemplates("my-ns").Update(&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-template", Namespace: "my-ns"},
			Spec: wfv1.WorkflowTemplateSpec{
				// note the non-trival expression
				Event: &wfv1.Event{Expression: "event.type == \"test\""},
				WorkflowSpec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{
						{
							Name:      "my-param",
							ValueFrom: &wfv1.ValueFrom{Expression: "event.type"},
						}, {
							Name:  "other-param",
							Value: &intstr.IntOrString{StrVal: "bar"},
						},
					}},
				},
			},
		})
		assert.NoError(t, err)
		wf, err := operation.submitWorkflowFromWorkflowTemplate("my-ns", "my-template")
		if assert.NoError(t, err) && assert.NotNil(t, wf, "workflow created") {
			assert.Contains(t, wf.Labels, common.LabelKeyControllerInstanceID, "instance ID labels is applied")
			assert.Contains(t, wf.Labels, common.LabelKeyCreator, "creator label is applied")
			parameters := wf.Spec.Arguments.Parameters
			if assert.Len(t, parameters, 1) {
				assert.Equal(t, "my-param", parameters[0].Name)
				assert.Equal(t, "test", parameters[0].Value.String())
			}
		}
	})
}

func Test_metaData(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		data := metaData(context.TODO())
		if assert.Len(t, data, 1) {
			assert.Nil(t, data["claimSet"], "always has claimSet, even if nil")
		}
	})
	t.Run("Headers", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.TODO(), metadata.MD{
			"x-valid": []string{"true"},
			"ignored": []string{"false"},
		})
		data := metaData(ctx)
		if assert.Len(t, data, 2) {
			assert.Equal(t, []string{"true"}, data["x-valid"])
		}
	})
}
