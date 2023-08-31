package dispatch

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func Test_metaData(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		data := metaData(context.TODO())
		assert.Empty(t, data)
	})
	t.Run("Headers", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.TODO(), metadata.MD{
			"x-valid": []string{"true"},
			"ignored": []string{"false"},
		})
		data := metaData(ctx)
		if assert.Len(t, data, 1) {
			assert.Equal(t, []string{"true"}, data["x-valid"])
		}
	})
}

func TestNewOperation(t *testing.T) {
	// set-up
	client := fake.NewSimpleClientset(
		&wfv1.ClusterWorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-cwft", Labels: map[string]string{common.LabelKeyControllerInstanceID: "my-instanceid"}},
		},
		&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wft", Namespace: "my-ns", Labels: map[string]string{common.LabelKeyControllerInstanceID: "my-instanceid"}},
		},
		&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wft-2", Namespace: "my-ns", Labels: map[string]string{common.LabelKeyControllerInstanceID: "my-instanceid"}},
		},
		&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wft-3", Namespace: "my-ns"},
		},
		&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wft-4", Namespace: "my-ns", Labels: map[string]string{common.LabelKeyControllerInstanceID: "my-instanceid"}},
		},
	)
	ctx := context.WithValue(context.WithValue(context.Background(), auth.WfKey, client), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}})
	recorder := record.NewFakeRecorder(6)

	// act
	operation, err := NewOperation(ctx, instanceid.NewService("my-instanceid"), recorder, []wfv1.WorkflowEventBinding{
		// test a malformed binding
		{
			ObjectMeta: metav1.ObjectMeta{Name: "malformed", Namespace: "my-ns"},
		},
		// test a binding that cannot match
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-0", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "false"},
			},
		},
		// test a binding with a cluster template
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-1", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-cwft", ClusterScope: true},
					Arguments:           &wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "my-param", ValueFrom: &wfv1.ValueFrom{Event: `"bar"`}}}},
				},
			},
		},
		// test a binding with a namespace template
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-2", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					Arguments:           &wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "my-param", ValueFrom: &wfv1.ValueFrom{Event: `"bar"`}}}},
				},
			},
		},
		// test a bind with a payload and expression with a map in it
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-3", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "payload.foo.bar == 'baz'"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft-2"},
					Arguments:           &wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "my-param", ValueFrom: &wfv1.ValueFrom{Event: "payload.foo"}}}},
				},
			},
		},
		// test a binding that errors due to missing workflow template
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-4", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "not-found"},
				},
			},
		},
		// test a binding that errors match due to wrong instance ID on template
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-5", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft-3"},
				},
			},
		},
		// test a binding with an invalid event selector
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-6", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "garbage!!!!!!"},
			},
		},
		// test a binding with a non-bool event selector
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-6", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: `"garbage"`},
			},
		},
		// test a binding with an invalid param expression
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-7", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					Arguments:           &wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "my-param", ValueFrom: &wfv1.ValueFrom{Event: "rubbish!!!"}}}},
				},
			},
		},
		// test a bind with a payload and fmt expression
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-8", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft-4"},
					Arguments:           &wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "my-param", ValueFrom: &wfv1.ValueFrom{Event: "payload.formatted"}}}},
				},
			},
		},
	}, "my-ns", "my-discriminator", &wfv1.Item{Value: json.RawMessage(`{"foo": {"bar": "baz"}, "formatted": "My%Test%"}`)})
	assert.NoError(t, err)
	err = operation.Dispatch(ctx)
	assert.Error(t, err)

	expectedParamValues := []string{
		`My%Test%`,
		"bar",
		"bar",
		`{"bar":"baz"}`,
	}
	var paramValues []string
	// assert
	list, err := client.ArgoprojV1alpha1().Workflows("my-ns").List(ctx, metav1.ListOptions{})
	if assert.NoError(t, err) && assert.Len(t, list.Items, 4) {
		for _, wf := range list.Items {
			assert.Equal(t, "my-instanceid", wf.Labels[common.LabelKeyControllerInstanceID])
			assert.Equal(t, "my-sub", wf.Labels[common.LabelKeyCreator])
			assert.Contains(t, wf.Labels, common.LabelKeyWorkflowEventBinding)
			assert.Contains(t, "my-param", wf.Spec.Arguments.Parameters[0].Name)
			paramValues = append(paramValues, string(*wf.Spec.Arguments.Parameters[0].Value))
		}
		sort.Strings(paramValues)
		assert.Equal(t, expectedParamValues, paramValues)
	}
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow template expression: unable to evaluate expression '': unexpected token EOF (1:1)", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to get workflow template: workflowtemplates.argoproj.io \"not-found\" not found", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to validate workflow template instanceid: 'my-wft-3' is not managed by the current Argo Server", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow template expression: unable to evaluate expression 'garbage!!!!!!': unexpected token Operator(\"!\") (1:8)\n | garbage!!!!!!\n | .......^", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow template expression: unable to cast expression result 'garbage' to bool", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow template parameter \"my-param\" expression: unexpected token Operator(\"!\") (1:8)\n | rubbish!!!\n | .......^", <-recorder.Events)
}

func Test_populateWorkflowMetadata(t *testing.T) {
	// set-up
	client := fake.NewSimpleClientset(
		&wfv1.WorkflowTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wft", Namespace: "my-ns", Labels: map[string]string{common.LabelKeyControllerInstanceID: "my-instanceid"}},
		},
	)
	ctx := context.WithValue(context.WithValue(context.Background(), auth.WfKey, client), auth.ClaimsKey, &types.Claims{Claims: jwt.Claims{Subject: "my-sub"}})
	recorder := record.NewFakeRecorder(10)

	// act
	operation, err := NewOperation(ctx, instanceid.NewService("my-instanceid"), recorder, []wfv1.WorkflowEventBinding{
		{
			// No name specified
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-1", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
				},
			},
		},
		{
			// Fixed name, label and annotation given
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-2", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "\"my-wfeb-2\"",
						Labels:      map[string]string{"aLabel": "\"someValue\""},
						Annotations: map[string]string{"anAnnotation": "\"otherValue\""},
					},
				},
			},
		},
		{
			// Name, label and annotations using expressions
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-3", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "\"my-wfeb-\" + payload.foo.bar",
						Labels:      map[string]string{"aLabel": "payload.list[0]"},
						Annotations: map[string]string{"anAnnotation": "payload.list[1]"},
					},
				},
			},
		},
		{
			// Invalid name expression
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-4", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.......foo[.numeric]"},
				},
			},
		},
		{
			// Invalid label expression
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-4", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"invalidLabel": "foo...bar"},
					},
				},
			},
		},
		{
			// Invalid annotation expression
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-4", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"invalidAnnotation": "foo.[..]bar"},
					},
				},
			},
		},
		{
			// Name expression evaluates to invalid type (float)
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-5", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.foo.numeric"},
				},
			},
		},
		{
			// Name expression evaluates to invalid type (boolean)
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-6", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.foo.bool"},
				},
			},
		},
		{
			// Name expression evaluates to invalid type (map)
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-7", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.foo"},
				},
			},
		},
		{
			// Name expression evaluates to invalid type (list)
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-8", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.list"},
				},
			},
		},
		{
			// Name expression evaluates to non existent value (nil)
			ObjectMeta: metav1.ObjectMeta{Name: "my-wfeb-9", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{Name: "payload.nothing"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "my-wfeb-10", Namespace: "my-ns"},
			Spec: wfv1.WorkflowEventBindingSpec{
				Event: wfv1.Event{Selector: "true"},
				Submit: &wfv1.Submit{
					WorkflowTemplateRef: wfv1.WorkflowTemplateRef{Name: "my-wft"},
					ObjectMeta:          metav1.ObjectMeta{GenerateName: `"my-wft-pr-"+sprig.toString(payload.foo.pr)+"-"`},
				},
			},
		},
	}, "my-ns", "my-discriminator",
		&wfv1.Item{Value: json.RawMessage(`{"foo": {"bar": "baz", "numeric": 8675309, "bool": true, "pr": 112}, "list": ["one", "two"]}`)})

	assert.NoError(t, err)
	err = operation.Dispatch(ctx)
	assert.Error(t, err)

	list, err := client.ArgoprojV1alpha1().Workflows("my-ns").List(ctx, metav1.ListOptions{})

	assert.NoError(t, err)
	assert.Len(t, list.Items, 4)

	expectedNames := []string{
		"my-wfeb-2",
		"my-wfeb-baz",
	}
	actualNames := []string{}
	var hasExpectGenerateName bool
	for _, item := range list.Items {
		actualNames = append(actualNames, item.Name)

		// find the generateName
		if strings.HasPrefix(item.Name, "my-wft-pr-112-") {
			hasExpectGenerateName = true
		}
	}

	assert.True(t, hasExpectGenerateName)

	// ordering not guaranteed
	assert.Subset(t, actualNames, expectedNames)

	for _, item := range list.Items {
		if _, ok := item.Labels["aLabel"]; !ok {
			assert.Contains(t, item.Name, "my-wft")
			continue
		}

		label := item.Labels["aLabel"]
		annotation := item.Annotations["anAnnotation"]
		assert.True(t, label == "someValue" || label == "one")
		assert.True(t, annotation == "otherValue" || annotation == "two")
	}

	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow name expression: unexpected token Operator(\"..\") (1:10)\n | payload.......foo[.numeric]\n | .........^", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow label \"invalidLabel\" expression: cannot use pointer accessor outside closure (1:6)\n | foo...bar\n | .....^", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: failed to evaluate workflow annotation \"invalidAnnotation\" expression: expected name (1:6)\n | foo.[..]bar\n | .....^", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: workflow name expression must evaluate to a string, not a float64", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: workflow name expression must evaluate to a string, not a bool", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: workflow name expression must evaluate to a string, not a map[string]interface {}", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: workflow name expression must evaluate to a string, not a []interface {}", <-recorder.Events)
	assert.Equal(t, "Warning WorkflowEventBindingError failed to dispatch event: workflow name expression must evaluate to a string, not a <nil>", <-recorder.Events)
}

func Test_expressionEnvironment(t *testing.T) {
	env, err := expressionEnvironment(context.TODO(), "my-ns", "my-d", &wfv1.Item{Value: []byte(`{"foo":"bar"}`)})
	if assert.NoError(t, err) {
		assert.Equal(t, "my-ns", env["namespace"])
		assert.Equal(t, "my-d", env["discriminator"])
		assert.Contains(t, env, "metadata")
		assert.Equal(t, map[string]interface{}{"foo": "bar"}, env["payload"], "make sure we parse an object as a map")
	}
}
