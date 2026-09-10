package executor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// TestResourceFlags tests whether Resource Flags
// are properly passed to `kubectl` command
func TestResourceFlags(t *testing.T) {
	manifestPath := "../../examples/hello-world.yaml"
	fakeClientset := fake.NewClientset()
	fakeFlags := []string{"--fake=true"}

	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	template := wfv1.Template{
		Resource: &wfv1.ResourceTemplate{
			Action: "fake",
			Flags:  fakeFlags,
		},
	}

	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        template,
		ClientSet:       fakeClientset,
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mockRuntimeExecutor,
	}
	args, err := we.getKubectlArguments("fake", manifestPath, fakeFlags)
	require.NoError(t, err)
	assert.Contains(t, args, fakeFlags[0])

	_, err = we.getKubectlArguments("fake", manifestPath, nil)
	require.NoError(t, err)
	_, err = we.getKubectlArguments("fake", "unknown-location", fakeFlags)
	if runtime.GOOS == "windows" {
		require.EqualError(t, err, "open unknown-location: The system cannot find the file specified.")
	} else {
		require.EqualError(t, err, "open unknown-location: no such file or directory")
	}

	emptyFile, err := os.CreateTemp("/tmp", "empty-manifest")
	require.NoError(t, err)
	defer func() { _ = os.Remove(emptyFile.Name()) }()
	_, err = we.getKubectlArguments("fake", emptyFile.Name(), nil)
	require.EqualError(t, err, "Must provide at least one of flags or manifest.")
}

// TestResourcePatchFlags tests whether Resource Flags
// are properly passed to `kubectl patch` command
func TestResourcePatchFlags(t *testing.T) {
	fakeFlags := []string{"pod", "mypod"}
	fakeClientset := fake.NewClientset()
	mockRuntimeExecutor := mocks.ContainerRuntimeExecutor{}

	tests := []struct {
		name           string
		patchType      string
		appendFileFlag bool
		manifestPath   string
	}{
		{
			name:           "strategic -f --patch-file",
			patchType:      "strategic",
			appendFileFlag: true,
			manifestPath:   "../../examples/hello-world.yaml", // any YAML with a `kind`
		},
		{
			name:           "json --patch-file",
			patchType:      "json",
			appendFileFlag: false,
			manifestPath:   "../../.golangci.yml", // any YAML without a `kind`
		},
		{
			name:           "merge --patch-file",
			patchType:      "merge",
			appendFileFlag: false,
			manifestPath:   "../../.golangci.yml", // any YAML without a `kind`
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedArgs := []string{"kubectl", "patch", "--type", tt.patchType, "--patch-file", tt.manifestPath}
			expectedArgs = append(expectedArgs, fakeFlags...)
			if tt.appendFileFlag {
				expectedArgs = append(expectedArgs, "-f", tt.manifestPath)
			}
			expectedArgs = append(expectedArgs, "-o", "json")

			template := wfv1.Template{
				Resource: &wfv1.ResourceTemplate{
					Action:        "patch",
					Flags:         fakeFlags,
					MergeStrategy: tt.patchType,
				},
			}
			we := WorkflowExecutor{
				PodName:         fakePodName,
				Template:        template,
				ClientSet:       fakeClientset,
				Namespace:       fakeNamespace,
				RuntimeExecutor: &mockRuntimeExecutor,
			}
			args, err := we.getKubectlArguments("patch", tt.manifestPath, fakeFlags)

			require.NoError(t, err)
			assert.Equal(t, expectedArgs, args)
		})
	}
}

// TestResourceConditionsMatching tests whether the JSON response match
// with either success or failure conditions.
func TestResourceConditionsMatching(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var successReqs labels.Requirements
	successSelector, err := labels.Parse("status.phase == Succeeded")
	require.NoError(t, err)
	successReqs, _ = successSelector.Requirements()
	require.NoError(t, err)
	var failReqs labels.Requirements
	failSelector, err := labels.Parse("status.phase == Error")
	require.NoError(t, err)
	failReqs, _ = failSelector.Requirements()
	require.NoError(t, err)

	jsonBytes := []byte(`{"name": "test","status":{"phase":"Error"}`)
	finished, err := matchConditions(ctx, jsonBytes, successReqs, failReqs)
	require.Error(t, err, `failure condition '{status.phase == [Error]}' evaluated true`)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Succeeded"}`)
	finished, err = matchConditions(ctx, jsonBytes, successReqs, failReqs)
	require.NoError(t, err)
	assert.False(t, finished)

	jsonBytes = []byte(`{"name": "test","status":{"phase":"Pending"}`)
	finished, err = matchConditions(ctx, jsonBytes, successReqs, failReqs)
	require.Error(t, err, "Neither success condition nor the failure condition has been matched. Retrying...")
	assert.True(t, finished)
}

// TestInferSelfLink tests whether the inferred self link for k8s objects are correct.
func TestInferSelfLink(t *testing.T) {
	obj := unstructured.Unstructured{}
	obj.SetNamespace("test-namespace")
	obj.SetName("test-name")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	})
	assert.Equal(t, "api/v1/namespaces/test-namespace/pods/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "TestKind",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/testkinds/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "Duty",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/duties/test-name", inferObjectSelfLink(obj))

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "IngressGateway",
	})
	assert.Equal(t, "apis/test.group/v1/namespaces/test-namespace/ingressgateways/test-name", inferObjectSelfLink(obj))

	obj.SetNamespace("")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	})
	assert.Equal(t, "api/v1/namespaces/test-name", inferObjectSelfLink(obj))
}

// TestResourceExecRetry tests whether Exec retries transitive errors
func TestResourceExecRetry(t *testing.T) {
	we := WorkflowExecutor{
		PodName:         fakePodName,
		Template:        wfv1.Template{},
		ClientSet:       fake.NewClientset(),
		Namespace:       fakeNamespace,
		RuntimeExecutor: &mocks.ContainerRuntimeExecutor{},
	}

	_, filename, _, _ := runtime.Caller(0)
	dirname := path.Dir(filename)
	duration := retry.DefaultBackoff.Duration
	defer func() {
		retry.DefaultBackoff.Duration = duration
	}()
	retry.DefaultBackoff.Duration = 0
	t.Setenv("PATH", dirname+"/testdata")
	ctx := logging.TestContext(t.Context())
	_, _, _, err := we.ExecResource(ctx, "", "../../examples/hello-world.yaml", nil)
	require.ErrorContains(t, err, "no more retries")
}

func Test_jqFilter(t *testing.T) {
	for _, testCase := range []struct {
		input  []byte
		filter string
		want   string
	}{
		{[]byte(`{"metadata": {"name": "foo"}}`), ".metadata.name", "foo"},
		{[]byte(`{"items": [{"key": "foo"}, {"key": "bar"}]}`), ".items.[].key", "foo\nbar"},
	} {
		t.Run(string(testCase.input), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			got, err := jqFilter(ctx, testCase.input, testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

const resourceJSON = `{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "my-pod",
    "namespace": "my-ns",
    "labels": {"app": "my-app"},
    "managedFields": [{"manager": "kubectl"}]
  },
  "spec": {"containers": [{"name": "a"}, {"name": "b"}]}
}`

// fakeKubectl simulates `kubectl get -o json`, which omits managedFields unless
// --show-managed-fields=true is passed. The read gating depends on that behaviour, so the fake has
// to honour the flag rather than always returning the same body. Every call is recorded in calls.
func fakeKubectl(calls *[][]string) kubectlRunner {
	return func(_ context.Context, args ...string) ([]byte, error) {
		*calls = append(*calls, args)
		if slices.Contains(args, "--show-managed-fields=true") {
			return []byte(resourceJSON), nil
		}
		obj := &unstructured.Unstructured{}
		if err := json.Unmarshal([]byte(resourceJSON), obj); err != nil {
			return nil, err
		}
		obj.SetManagedFields(nil)
		return obj.MarshalJSON()
	}
}

func Test_jsonPathFilter(t *testing.T) {
	obj := &unstructured.Unstructured{}
	require.NoError(t, json.Unmarshal([]byte(resourceJSON), obj))

	for _, tc := range []struct {
		expression string
		want       string
		wantErr    bool
	}{
		{expression: "{.metadata.name}", want: "my-pod"},
		{expression: "{.metadata.labels.app}", want: "my-app"},
		{expression: "{.spec.containers[*].name}", want: "a b"},
		// kubectl passes `-o jsonpath=` templates to the JSONPath printer verbatim: text outside
		// braces is literal, it is not relaxed into `{.metadata.name}`.
		{expression: "metadata.name", want: "metadata.name"},
		// kubectl defaults to --allow-missing-template-keys=true.
		{expression: "{.metadata.nope}", want: ""},
		// managedFields are still visible, as they were with `-o jsonpath=`.
		{expression: "{.metadata.managedFields[0].manager}", want: "kubectl"},
		// Whitespace in the template is part of the output. `-o jsonpath=` emits it verbatim, and
		// jqFilter's TrimSpace must not be copied here: `{range}`-style expressions rely on the
		// trailing newline surviving. Without these two cases, adding strings.TrimSpace to
		// jsonPathFilter leaves the whole table green.
		{expression: "{.metadata.name}{\"\\n\"}", want: "my-pod\n"},
		{expression: "{range .spec.containers[*]}{.name}{\"\\n\"}{end}", want: "a\nb\n"},
		{expression: "{unparseable", wantErr: true},
		// A valid template that cannot be executed against this object.
		{expression: "{.spec.containers[9].name}", wantErr: true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			got, err := jsonPathFilter(obj, tc.expression)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_saveResourceParameters(t *testing.T) {
	tracingObj, err := tracing.New(logging.TestContext(t.Context()), `argoexec`)
	require.NoError(t, err)
	newExecutor := func(params ...wfv1.Parameter) *WorkflowExecutor {
		return &WorkflowExecutor{
			PodName:         fakePodName,
			Namespace:       fakeNamespace,
			nodeID:          fakeNodeID,
			ClientSet:       fake.NewClientset(),
			RuntimeExecutor: &mocks.ContainerRuntimeExecutor{},
			Tracing:         tracingObj,
			Template:        wfv1.Template{Outputs: wfv1.Outputs{Parameters: params}},
		}
	}
	param := func(name string, valueFrom *wfv1.ValueFrom) wfv1.Parameter {
		return wfv1.Parameter{Name: name, ValueFrom: valueFrom}
	}

	t.Run("reads the resource once for all parameters", func(t *testing.T) {
		we := newExecutor(
			param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}),
			param("app", &wfv1.ValueFrom{JSONPath: "{.metadata.labels.app}"}),
			param("jq", &wfv1.ValueFrom{JQFilter: ".spec.containers[].name"}),
		)
		var calls [][]string
		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", fakeKubectl(&calls)))

		require.Len(t, calls, 1, "the resource must be read exactly once, not once per parameter")
		assert.Equal(t, []string{"kubectl", "-n", "my-ns", "get", "pod./my-pod", "-o", "json", "--show-managed-fields=true"}, calls[0])

		out := we.Template.Outputs.Parameters
		assert.Equal(t, "my-pod", out[0].Value.String())
		assert.Equal(t, "my-app", out[1].Value.String())
		assert.Equal(t, "a\nb", out[2].Value.String())
	})

	// managedFields are asked for only when a jsonPath expression could read them. A jqFilter never
	// sees them, so requesting them for a jqFilter-only template would inflate the response and then
	// throw the extra away - the opposite of what this change is for.
	t.Run("managedFields are requested only when a jsonPath parameter needs them", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			valueFrom *wfv1.ValueFrom
			want      bool
		}{
			{"jsonPath only", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}, true},
			{"jqFilter only", &wfv1.ValueFrom{JQFilter: ".metadata.name"}, false},
			{"jsonPath wins when a parameter sets both", &wfv1.ValueFrom{JSONPath: "{.metadata.name}", JQFilter: ".metadata.name"}, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				we := newExecutor(param("p", tc.valueFrom))
				var calls [][]string
				ctx := logging.TestContext(t.Context())
				require.NoError(t, we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", fakeKubectl(&calls)))
				require.Len(t, calls, 1)
				assert.Equal(t, tc.want, slices.Contains(calls[0], "--show-managed-fields=true"))
			})
		}
	})

	// Whichever way the resource was fetched, a jqFilter must see it without managedFields, exactly
	// as it did when it was fed plain `kubectl get -o json`.
	t.Run("jqFilter does not see managedFields", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			params []wfv1.Parameter
			at     int
		}{
			{"jqFilter alone", []wfv1.Parameter{
				param("keys", &wfv1.ValueFrom{JQFilter: ".metadata | keys | join(\",\")"}),
			}, 0},
			// Here the response does carry managedFields, because the jsonPath parameter asked for
			// them; they have to be stripped back out before the jqFilter runs.
			{"jqFilter alongside a jsonPath parameter", []wfv1.Parameter{
				param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}),
				param("keys", &wfv1.ValueFrom{JQFilter: ".metadata | keys | join(\",\")"}),
			}, 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				we := newExecutor(tc.params...)
				var calls [][]string
				ctx := logging.TestContext(t.Context())
				require.NoError(t, we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", fakeKubectl(&calls)))
				assert.Equal(t, "labels,name,namespace", we.Template.Outputs.Parameters[tc.at].Value.String())
			})
		}
	})

	// A valueFrom that names neither jsonPath nor jqFilter resolves to nothing, so it neither
	// triggers a read nor overwrites the value the parameter already has.
	t.Run("parameter with neither jsonPath nor jqFilter is left alone", func(t *testing.T) {
		we := newExecutor(wfv1.Parameter{
			Name:      "untouched",
			Value:     wfv1.AnyStringPtr("keep me"),
			ValueFrom: &wfv1.ValueFrom{Default: wfv1.AnyStringPtr("unused")},
		})
		var calls [][]string
		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", fakeKubectl(&calls)))
		assert.Empty(t, calls)
		assert.Equal(t, "keep me", we.Template.Outputs.Parameters[0].Value.String())
	})

	// A parameter with a literal value and no valueFrom is the honest fixture for "nothing to read":
	// validateOutputParameter rejects Supplied for resource templates, so it never reaches here.
	t.Run("no read when no parameter needs the resource", func(t *testing.T) {
		we := newExecutor(wfv1.Parameter{Name: "literal", Value: wfv1.AnyStringPtr("hello")})
		var calls [][]string
		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", fakeKubectl(&calls)))
		assert.Empty(t, calls)
		assert.Equal(t, "hello", we.Template.Outputs.Parameters[0].Value.String())
	})

	t.Run("no resource falls back to the default", func(t *testing.T) {
		we := newExecutor(param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}", Default: wfv1.AnyStringPtr("fallback")}))
		var calls [][]string
		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.saveResourceParameters(ctx, "", "", fakeKubectl(&calls)))
		assert.Empty(t, calls)
		assert.Equal(t, "fallback", we.Template.Outputs.Parameters[0].Value.String())
	})

	t.Run("errors are returned, not swallowed", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			param   wfv1.Parameter
			kubectl kubectlRunner
			wantErr string
		}{
			{
				name:    "kubectl fails",
				param:   param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}),
				kubectl: func(context.Context, ...string) ([]byte, error) { return nil, errors.New("boom") },
				wantErr: "boom",
			},
			{
				// New failure mode: a jsonPath-only template never JSON-decoded kubectl's output
				// before this change, because raw stdout was the value. #3037 exists because
				// `kubectl get` can exit 0 with empty stdout.
				name:    "kubectl exits 0 with empty stdout",
				param:   param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}),
				kubectl: func(context.Context, ...string) ([]byte, error) { return nil, nil },
				wantErr: "unexpected end of JSON input",
			},
			{
				name:    "kubectl exits 0 with output that is not JSON",
				param:   param("name", &wfv1.ValueFrom{JSONPath: "{.metadata.name}"}),
				kubectl: func(context.Context, ...string) ([]byte, error) { return []byte("No resources found"), nil },
				wantErr: "invalid character",
			},
			{
				name:    "jsonPath cannot be parsed",
				param:   param("name", &wfv1.ValueFrom{JSONPath: "{unparseable"}),
				wantErr: "error parsing jsonpath",
			},
			{
				name:    "jqFilter cannot be parsed",
				param:   param("name", &wfv1.ValueFrom{JQFilter: "{{"}),
				wantErr: "unexpected token",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				we := newExecutor(tc.param)
				kubectl := tc.kubectl
				if kubectl == nil {
					var calls [][]string
					kubectl = fakeKubectl(&calls)
				}
				ctx := logging.TestContext(t.Context())
				err := we.saveResourceParameters(ctx, "my-ns", "pod./my-pod", kubectl)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		}
	})
}

// SaveResourceParameters owns the wiring that saveResourceParameters does not: the short-circuit on
// a template with no output parameters, and the ReportOutputs call. Neither is reachable from
// saveResourceParameters, so both are pinned here - without this, deleting the ReportOutputs call
// passes the whole unit suite.
func TestSaveResourceParameters(t *testing.T) {
	tracingObj, err := tracing.New(logging.TestContext(t.Context()), `argoexec`)
	require.NoError(t, err)
	newExecutor := func(params ...wfv1.Parameter) (*WorkflowExecutor, *argofake.Clientset) {
		argoClientset := argofake.NewClientset()
		we := &WorkflowExecutor{
			PodName:          fakePodName,
			Namespace:        fakeNamespace,
			nodeID:           fakeNodeID,
			ClientSet:        fake.NewClientset(),
			RuntimeExecutor:  &mocks.ContainerRuntimeExecutor{},
			Tracing:          tracingObj,
			Template:         wfv1.Template{Outputs: wfv1.Outputs{Parameters: params}},
			taskResultClient: argoClientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace),
		}
		return we, argoClientset
	}

	t.Run("reports the resolved output parameters", func(t *testing.T) {
		we, argoClientset := newExecutor(wfv1.Parameter{
			Name:      "name",
			ValueFrom: &wfv1.ValueFrom{JSONPath: "{.metadata.name}"},
		})
		var calls [][]string
		original := kubectlRunnerFn
		kubectlRunnerFn = fakeKubectl(&calls)
		t.Cleanup(func() { kubectlRunnerFn = original })

		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.SaveResourceParameters(ctx, "my-ns", "pod./my-pod"))

		require.Len(t, calls, 1)
		assert.Equal(t, "my-pod", we.Template.Outputs.Parameters[0].Value.String())
		assert.NotEmpty(t, argoClientset.Actions(), "the resolved parameters must be reported")
	})

	// A failure to resolve the parameters must propagate and must not report a partial result.
	t.Run("resolution failure propagates and reports nothing", func(t *testing.T) {
		we, argoClientset := newExecutor(wfv1.Parameter{
			Name:      "name",
			ValueFrom: &wfv1.ValueFrom{JSONPath: "{.metadata.name}"},
		})
		original := kubectlRunnerFn
		kubectlRunnerFn = func(context.Context, ...string) ([]byte, error) { return nil, errors.New("boom") }
		t.Cleanup(func() { kubectlRunnerFn = original })

		ctx := logging.TestContext(t.Context())
		err := we.SaveResourceParameters(ctx, "my-ns", "pod./my-pod")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
		assert.Empty(t, argoClientset.Actions(), "nothing should be reported when resolution failed")
	})

	// Must keep short-circuiting before ReportOutputs when the template has no output parameters,
	// otherwise every resource template would start writing a task result.
	t.Run("no output parameters reports nothing", func(t *testing.T) {
		we, argoClientset := newExecutor()
		ctx := logging.TestContext(t.Context())
		require.NoError(t, we.SaveResourceParameters(ctx, "my-ns", "pod./my-pod"))
		assert.Empty(t, argoClientset.Actions(), "no task result should be written when there are no output parameters")
	})
}

func Test_runKubectl(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	out, err := runKubectl(ctx, "kubectl", "version", "--client=true", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, string(out), "clientVersion")
}
