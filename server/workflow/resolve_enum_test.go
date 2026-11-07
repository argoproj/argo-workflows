package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	v1alpha "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func newUnstructured(apiVersion, kind, namespace, name string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetNamespace(namespace)
	u.SetName(name)
	u.SetLabels(labels)
	return u
}

func enumParam(name, apiVersion, kind, namespace, labelSelector string) v1alpha1.Parameter {
	return v1alpha1.Parameter{Name: name, EnumFrom: &v1alpha1.EnumFrom{
		Resource:      v1alpha1.ResourceSelector{APIVersion: apiVersion, Kind: kind, Namespace: namespace},
		JSONPath:      "{.metadata.name}",
		LabelSelector: labelSelector,
	}}
}

// enumSpec defines every enumFrom case these tests resolve.
func enumSpec() v1alpha1.WorkflowSpec {
	badPath := enumParam("bad-path", "v1", "Namespace", "", "")
	badPath.EnumFrom.JSONPath = "{.metadata.name"
	labelValues := enumParam("label-values", "apps/v1", "Deployment", "team-a", "")
	labelValues.EnumFrom.JSONPath = "{.metadata.labels.*}"
	return v1alpha1.WorkflowSpec{
		Entrypoint: "main",
		Arguments: v1alpha1.Arguments{Parameters: []v1alpha1.Parameter{
			enumParam("namespaces", "v1", "Namespace", "", ""),
			enumParam("deployments", "apps/v1", "Deployment", "", ""),
			enumParam("endpoints", "v1", "Endpoints", "", ""),
			enumParam("web-deployments", "apps/v1", "Deployment", "", "tier=web"),
			enumParam("cascading", "apps/v1", "Deployment", "{{workflow.parameters.ns}}", "tier={{workflow.parameters.tier}}"),
			enumParam("cascading-spaces", "apps/v1", "Deployment", "{{ workflow.parameters.ns }}", ""),
			enumParam("unsupported-reference", "apps/v1", "Deployment", "{{workflow.namespace}}", ""),
			enumParam("unknown-kind", "apps/v1", "Nope", "", ""),
			enumParam("unknown-group", "example.com/v1", "Thing", "", ""),
			enumParam("secrets", "v1", "Secret", "", ""),
			badPath,
			labelValues,
			{Name: "plain", Value: v1alpha1.AnyStringPtr("x")},
		}},
		Templates: []v1alpha1.Template{{
			Name:   "main",
			Inputs: v1alpha1.Inputs{Parameters: []v1alpha1.Parameter{enumParam("input-ns", "v1", "Namespace", "", "")}},
		}},
	}
}

func resolveEnumServer(t *testing.T) (*workflowServer, context.Context, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	kube := fake.NewClientset()
	kube.Resources = []*metav1.APIResourceList{
		{GroupVersion: "v1", APIResources: []metav1.APIResource{
			{Name: "namespaces", Kind: "Namespace", Namespaced: false},
			{Name: "namespaces/status", Kind: "Namespace", Namespaced: false},
			{Name: "endpoints", Kind: "Endpoints", Namespaced: true},
			{Name: "secrets", Kind: "Secret", Namespaced: true},
		}},
		{GroupVersion: "apps/v1", APIResources: []metav1.APIResource{
			{Name: "deployments", Kind: "Deployment", Namespaced: true},
		}},
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Version: "v1", Resource: "namespaces"}:                 "NamespaceList",
			{Version: "v1", Resource: "endpoints"}:                  "EndpointsList",
			{Version: "v1", Resource: "secrets"}:                    "SecretList",
			{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		},
		newUnstructured("v1", "Namespace", "", "team-b", nil),
		newUnstructured("v1", "Namespace", "", "team-a", nil),
		newUnstructured("v1", "Endpoints", "argo", "argo-server", nil),
		newUnstructured("v1", "Secret", "argo", "my-secret", nil),
		newUnstructured("apps/v1", "Deployment", "argo", "argo-server", map[string]string{"tier": "web"}),
		newUnstructured("apps/v1", "Deployment", "argo", "minio", map[string]string{"tier": "storage"}),
		newUnstructured("apps/v1", "Deployment", "team-a", "api", map[string]string{"tier": "web", "team": "a"}),
	)
	wfClient := v1alpha.NewClientset(
		&v1alpha1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "wftmpl", Namespace: "argo"}, Spec: enumSpec()},
		&v1alpha1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "cwftmpl"}, Spec: enumSpec()},
		&v1alpha1.CronWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "cronwf", Namespace: "argo"}, Spec: v1alpha1.CronWorkflowSpec{WorkflowSpec: enumSpec()}},
		// A workflow submitted from a template carries the template's parameters in its stored spec.
		&v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "argo"},
			Spec:       v1alpha1.WorkflowSpec{WorkflowTemplateRef: &v1alpha1.WorkflowTemplateRef{Name: "wftmpl"}},
			Status:     v1alpha1.WorkflowStatus{StoredWorkflowSpec: new(enumSpec())},
		},
		&v1alpha1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "other-instance", Namespace: "argo", Labels: map[string]string{"workflows.argoproj.io/controller-instanceid": "other"}}, Spec: enumSpec()},
	)
	ctx := logging.TestContext(t.Context())
	ctx = context.WithValue(ctx, auth.KubeKey, kube)
	ctx = context.WithValue(ctx, auth.DynamicKey, dyn)
	ctx = context.WithValue(ctx, auth.WfKey, wfClient)
	s := &workflowServer{instanceIDService: instanceid.NewService(""), enumResolver: newEnumResolver(ctx)}
	return s, ctx, dyn
}

func TestResolveParameterEnum(t *testing.T) {
	s, ctx, _ := resolveEnumServer(t)
	resolve := func(req *workflowpkg.ResolveParameterEnumRequest) ([]string, error) {
		if req.Namespace == "" {
			req.Namespace = "argo"
		}
		if req.ResourceKind == "" {
			req.ResourceKind = "WorkflowTemplate"
			req.ResourceName = "wftmpl"
		}
		resp, err := s.ResolveParameterEnum(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.Values, nil
	}

	t.Run("ClusterScopedIgnoresRequestNamespace", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "namespaces"})
		require.NoError(t, err)
		assert.Equal(t, []string{"team-a", "team-b"}, values)
	})
	t.Run("NamespacedDefaultsToRequestNamespace", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "deployments"})
		require.NoError(t, err)
		assert.Equal(t, []string{"argo-server", "minio"}, values)
	})
	t.Run("IrregularPlural", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "endpoints"})
		require.NoError(t, err)
		assert.Equal(t, []string{"argo-server"}, values)
	})
	t.Run("LabelSelector", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "web-deployments"})
		require.NoError(t, err)
		assert.Equal(t, []string{"argo-server"}, values)
	})
	t.Run("Cascading", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "cascading", Parameters: map[string]string{"ns": "team-a", "tier": "web"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"api"}, values)
	})
	// The UI sends every referenced parameter, as "" while it is empty, so empty must mean "no value yet",
	// the same as missing; otherwise an empty labelSelector reference would match unlabelled resources.
	for name, params := range map[string]map[string]string{
		"CascadingParentMissing":      nil,
		"CascadingParentEmpty":        {"ns": "", "tier": ""},
		"CascadingNamespaceEmpty":     {"ns": "", "tier": "web"},
		"CascadingLabelSelectorEmpty": {"ns": "team-a", "tier": ""},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "cascading", Parameters: params})
			assert.Equal(t, codes.FailedPrecondition, status.Code(err), "%v", err)
		})
	}
	t.Run("WaitingNamesTheParameters", func(t *testing.T) {
		_, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "cascading", Parameters: map[string]string{"ns": "team-a", "tier": ""}})
		assert.Equal(t, "waiting for a value for tier", status.Convert(err).Message())
	})
	t.Run("CascadingWithSpacesInReference", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "cascading-spaces", Parameters: map[string]string{"ns": "team-a"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"api"}, values)
	})
	t.Run("UnsupportedReference", func(t *testing.T) {
		_, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "unsupported-reference"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "%v", err)
	})
	t.Run("TemplateInput", func(t *testing.T) {
		values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ParameterName: "input-ns", TemplateName: "main"})
		require.NoError(t, err)
		assert.Equal(t, []string{"team-a", "team-b"}, values)
	})

	for _, src := range []struct{ kind, name string }{
		{"ClusterWorkflowTemplate", "cwftmpl"},
		{"CronWorkflow", "cronwf"},
		{"Workflow", "wf"},
	} {
		t.Run("From"+src.kind, func(t *testing.T) {
			values, err := resolve(&workflowpkg.ResolveParameterEnumRequest{ResourceKind: src.kind, ResourceName: src.name, ParameterName: "namespaces"})
			require.NoError(t, err)
			assert.Equal(t, []string{"team-a", "team-b"}, values)
		})
	}

	for name, tc := range map[string]struct {
		req  *workflowpkg.ResolveParameterEnumRequest
		code codes.Code
	}{
		"SecretsNeverListed":     {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "secrets"}, codes.PermissionDenied},
		"UnknownKind":            {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "unknown-kind"}, codes.InvalidArgument},
		"UnknownGroupVersion":    {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "unknown-group"}, codes.InvalidArgument},
		"InvalidJSONPath":        {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "bad-path"}, codes.InvalidArgument},
		"ParameterWithoutEnum":   {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "plain"}, codes.InvalidArgument},
		"ParameterNotFound":      {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "nope"}, codes.InvalidArgument},
		"ParameterNameMissing":   {&workflowpkg.ResolveParameterEnumRequest{}, codes.InvalidArgument},
		"TemplateNotFound":       {&workflowpkg.ResolveParameterEnumRequest{ParameterName: "input-ns", TemplateName: "nope"}, codes.InvalidArgument},
		"UnsupportedKind":        {&workflowpkg.ResolveParameterEnumRequest{ResourceKind: "Pod", ResourceName: "x", ParameterName: "namespaces"}, codes.InvalidArgument},
		"ResourceNameMissing":    {&workflowpkg.ResolveParameterEnumRequest{ResourceKind: "WorkflowTemplate", ParameterName: "namespaces"}, codes.InvalidArgument},
		"TemplateNotFoundInKube": {&workflowpkg.ResolveParameterEnumRequest{ResourceKind: "WorkflowTemplate", ResourceName: "missing", ParameterName: "namespaces"}, codes.NotFound},
		"OtherInstanceID":        {&workflowpkg.ResolveParameterEnumRequest{ResourceKind: "WorkflowTemplate", ResourceName: "other-instance", ParameterName: "namespaces"}, codes.InvalidArgument},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolve(tc.req)
			assert.Equal(t, tc.code, status.Code(err), "%v", err)
		})
	}
}

// listCalls counts list requests that reached the (fake) Kubernetes API.
func listCalls(dyn *dynamicfake.FakeDynamicClient) int {
	n := 0
	for _, a := range dyn.Actions() {
		if a.GetVerb() == "list" {
			n++
		}
	}
	return n
}

func withCaller(ctx context.Context, authorization string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", authorization))
}

func TestResolveParameterEnumCache(t *testing.T) {
	req := func() *workflowpkg.ResolveParameterEnumRequest {
		return &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "deployments"}
	}

	t.Run("RepeatedRequestsHitCache", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		ctx = withCaller(ctx, "Bearer alice")
		for range 3 {
			resp, err := s.ResolveParameterEnum(ctx, req())
			require.NoError(t, err)
			assert.Equal(t, []string{"argo-server", "minio"}, resp.Values)
		}
		assert.Equal(t, 1, listCalls(dyn))
	})
	t.Run("CallersDoNotShareEntries", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		_, err := s.ResolveParameterEnum(withCaller(ctx, "Bearer alice"), req())
		require.NoError(t, err)
		_, err = s.ResolveParameterEnum(withCaller(ctx, "Bearer bob"), req())
		require.NoError(t, err)
		assert.Equal(t, 2, listCalls(dyn))
	})
	t.Run("ParameterValuesArePartOfTheKey", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		for _, ns := range []string{"argo", "team-a", "argo"} {
			_, err := s.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{
				Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "cascading",
				Parameters: map[string]string{"ns": ns, "tier": "web"},
			})
			require.NoError(t, err)
		}
		assert.Equal(t, 2, listCalls(dyn))
	})
	t.Run("ZeroTTLDisablesCache", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		s.enumResolver.cacheTTL = 0
		for range 2 {
			_, err := s.ResolveParameterEnum(ctx, req())
			require.NoError(t, err)
		}
		assert.Equal(t, 2, listCalls(dyn))
	})
	t.Run("ErrorsAreNotCached", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		dyn.PrependReactor("list", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierr.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "", nil)
		})
		for range 2 {
			_, err := s.ResolveParameterEnum(ctx, req())
			assert.Equal(t, codes.PermissionDenied, status.Code(err))
		}
		assert.Equal(t, 2, listCalls(dyn))
	})
}

func TestResolveParameterEnumLimit(t *testing.T) {
	t.Run("TooManyItems", func(t *testing.T) {
		s, ctx, _ := resolveEnumServer(t)
		s.enumResolver.maxItems = 1
		resp, err := s.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "deployments"})
		require.NoError(t, err)
		assert.Len(t, resp.Values, 1)
		assert.True(t, resp.Truncated)
	})
	t.Run("TooManyValuesFromOneItem", func(t *testing.T) {
		s, ctx, _ := resolveEnumServer(t)
		s.enumResolver.maxItems = 1
		resp, err := s.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "label-values"})
		require.NoError(t, err)
		assert.Len(t, resp.Values, 1)
		assert.True(t, resp.Truncated)
	})
	t.Run("UnderLimit", func(t *testing.T) {
		s, ctx, _ := resolveEnumServer(t)
		resp, err := s.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "label-values"})
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "web"}, resp.Values)
		assert.False(t, resp.Truncated)
	})
	t.Run("RequestsOnePageFromAPIServer", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		s.enumResolver.maxItems = 7
		_, err := s.ResolveParameterEnum(ctx, &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: "deployments"})
		require.NoError(t, err)
		for _, a := range dyn.Actions() {
			if l, ok := a.(ktesting.ListActionImpl); ok {
				assert.Equal(t, int64(7), l.ListOptions.Limit)
			}
		}
	})
}

// kubeCalls counts the Kubernetes calls made so far, by kind of call.
type kubeCalls struct{ templateGets, discovery, lists int }

func countKubeCalls(ctx context.Context, dyn *dynamicfake.FakeDynamicClient) kubeCalls {
	var c kubeCalls
	for _, a := range auth.GetWfClient(ctx).(*v1alpha.Clientset).Actions() {
		if a.GetVerb() == "get" {
			c.templateGets++
		}
	}
	for _, a := range auth.GetKubeClient(ctx).(*fake.Clientset).Actions() {
		if a.GetVerb() == "get" && a.GetResource().Resource == "resource" {
			c.discovery++
		}
	}
	c.lists = listCalls(dyn)
	return c
}

func TestResolveParameterEnumCacheLayers(t *testing.T) {
	req := func(param string, params map[string]string) *workflowpkg.ResolveParameterEnumRequest {
		return &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "wftmpl", ParameterName: param, Parameters: params}
	}

	t.Run("CacheHitMakesNoKubernetesCalls", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		ctx = withCaller(ctx, "Bearer alice")
		_, err := s.ResolveParameterEnum(ctx, req("deployments", nil))
		require.NoError(t, err)
		first := countKubeCalls(ctx, dyn)
		assert.Equal(t, kubeCalls{templateGets: 1, discovery: 1, lists: 1}, first)
		for range 3 {
			_, err := s.ResolveParameterEnum(ctx, req("deployments", nil))
			require.NoError(t, err)
		}
		assert.Equal(t, first, countKubeCalls(ctx, dyn))
	})
	t.Run("DiscoveryIsSharedBetweenCallers", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		for _, caller := range []string{"Bearer alice", "Bearer bob"} {
			_, err := s.ResolveParameterEnum(withCaller(ctx, caller), req("deployments", nil))
			require.NoError(t, err)
		}
		// Each caller reads the template and lists with their own credentials.
		assert.Equal(t, kubeCalls{templateGets: 2, discovery: 1, lists: 2}, countKubeCalls(ctx, dyn))
	})
	t.Run("ListIsSharedBetweenRequestsWithTheSameQuery", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		// Same query, but a parameter the lookup doesn't use differs, so the request key differs.
		for _, extra := range []string{"1", "2"} {
			_, err := s.ResolveParameterEnum(ctx, req("deployments", map[string]string{"unrelated": extra}))
			require.NoError(t, err)
		}
		assert.Equal(t, kubeCalls{templateGets: 2, discovery: 1, lists: 1}, countKubeCalls(ctx, dyn))
	})
	t.Run("FailedDiscoveryIsNotCached", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		for range 2 {
			_, err := s.ResolveParameterEnum(ctx, req("unknown-group", nil))
			require.Error(t, err)
		}
		assert.Equal(t, 2, countKubeCalls(ctx, dyn).discovery)
	})
	t.Run("ZeroTTLDisablesAllLayers", func(t *testing.T) {
		s, ctx, dyn := resolveEnumServer(t)
		s.enumResolver.cacheTTL = 0
		for range 2 {
			_, err := s.ResolveParameterEnum(ctx, req("deployments", nil))
			require.NoError(t, err)
		}
		assert.Equal(t, kubeCalls{templateGets: 2, discovery: 2, lists: 2}, countKubeCalls(ctx, dyn))
	})
}

func TestRequestCacheKey(t *testing.T) {
	base := &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "t", ParameterName: "p", Parameters: map[string]string{"a": "1", "b": "2"}}
	key := requestCacheKey("caller", base)
	same := &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: "WorkflowTemplate", ResourceName: "t", ParameterName: "p", Parameters: map[string]string{"b": "2", "a": "1"}}
	assert.Equal(t, key, requestCacheKey("caller", same), "map order must not matter")
	assert.NotEqual(t, key, requestCacheKey("other-caller", base))
	for name, change := range map[string]func(r *workflowpkg.ResolveParameterEnumRequest){
		"Namespace":    func(r *workflowpkg.ResolveParameterEnumRequest) { r.Namespace = "x" },
		"ResourceName": func(r *workflowpkg.ResolveParameterEnumRequest) { r.ResourceName = "x" },
		"TemplateName": func(r *workflowpkg.ResolveParameterEnumRequest) { r.TemplateName = "x" },
		"Parameter":    func(r *workflowpkg.ResolveParameterEnumRequest) { r.ParameterName = "x" },
		"Value":        func(r *workflowpkg.ResolveParameterEnumRequest) { r.Parameters = map[string]string{"a": "1", "b": "3"} },
		// "a=1b" + "" must not collide with "a=1" + "b".
		"Boundary": func(r *workflowpkg.ResolveParameterEnumRequest) { r.Parameters = map[string]string{"a": "1b", "": "2"} },
	} {
		t.Run(name, func(t *testing.T) {
			r := *base
			change(&r)
			assert.NotEqual(t, key, requestCacheKey("caller", &r))
		})
	}
}

func TestNewEnumResolverSettings(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Run("Defaults", func(t *testing.T) {
		r := newEnumResolver(ctx)
		assert.Equal(t, defaultEnumMaxItems, r.maxItems)
		assert.Equal(t, defaultEnumCacheTTL, r.cacheTTL)
	})
	t.Run("FromEnvironment", func(t *testing.T) {
		t.Setenv("ARGO_SERVER_RESOLVE_ENUM_MAX_ITEMS", "7")
		t.Setenv("ARGO_SERVER_RESOLVE_ENUM_CACHE_TTL", "0s")
		r := newEnumResolver(ctx)
		assert.Equal(t, 7, r.maxItems)
		assert.Equal(t, time.Duration(0), r.cacheTTL)
	})
	for _, v := range []string{"0", "-1"} {
		t.Run("MaxItems="+v+"MeansDefault", func(t *testing.T) {
			t.Setenv("ARGO_SERVER_RESOLVE_ENUM_MAX_ITEMS", v)
			assert.Equal(t, defaultEnumMaxItems, newEnumResolver(ctx).maxItems)
		})
	}
}

func TestResolveParameterEnumErrors(t *testing.T) {
	req := func(kind, name string) *workflowpkg.ResolveParameterEnumRequest {
		return &workflowpkg.ResolveParameterEnumRequest{Namespace: "argo", ResourceKind: kind, ResourceName: name, ParameterName: "deployments"}
	}

	for _, kind := range []string{"ClusterWorkflowTemplate", "CronWorkflow", "Workflow"} {
		t.Run(kind+"NotFound", func(t *testing.T) {
			s, ctx, _ := resolveEnumServer(t)
			_, err := s.ResolveParameterEnum(ctx, req(kind, "missing"))
			assert.Equal(t, codes.NotFound, status.Code(err), "%v", err)
		})
	}

	for name, tc := range map[string]struct {
		err  error
		code codes.Code
	}{
		// The cluster couldn't answer: not the caller's fault, so not InvalidArgument.
		"DiscoveryUnavailable": {errors.New("connection refused"), codes.Internal},
		"DiscoveryForbidden":   {apierr.NewForbidden(schema.GroupResource{}, "", nil), codes.PermissionDenied},
	} {
		t.Run(name, func(t *testing.T) {
			s, ctx, _ := resolveEnumServer(t)
			auth.GetKubeClient(ctx).(*fake.Clientset).PrependReactor("get", "resource", func(ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, tc.err
			})
			_, err := s.ResolveParameterEnum(ctx, req("WorkflowTemplate", "wftmpl"))
			assert.Equal(t, tc.code, status.Code(err), "%v", err)
		})
	}
}

func TestSubstituteEnumParameters(t *testing.T) {
	for name, tc := range map[string]struct {
		in, out string
		params  map[string]string
		code    codes.Code
		message string
	}{
		"NoReferences":          {in: "tier=web", out: "tier=web"},
		"Substituted":           {in: "app={{workflow.parameters.a}},tier={{workflow.parameters.b}}", params: map[string]string{"a": "x", "b": "y"}, out: "app=x,tier=y"},
		"SpacesInsideBraces":    {in: "{{  workflow.parameters.a  }}", params: map[string]string{"a": "x"}, out: "x"},
		"Missing":               {in: "{{workflow.parameters.a}}", code: codes.FailedPrecondition, message: "waiting for a value for a"},
		"Empty":                 {in: "{{workflow.parameters.a}}", params: map[string]string{"a": ""}, code: codes.FailedPrecondition, message: "waiting for a value for a"},
		"NamesEachOnce":         {in: "{{workflow.parameters.a}}-{{workflow.parameters.b}}-{{workflow.parameters.a}}", code: codes.FailedPrecondition, message: "waiting for a value for a, b"},
		"ValueWithBraces":       {in: "{{workflow.parameters.a}}", params: map[string]string{"a": "{{x}}"}, out: "{{x}}"},
		"OtherVariable":         {in: "{{workflow.namespace}}", code: codes.InvalidArgument},
		"OtherVariableAndEmpty": {in: "{{workflow.namespace}}/{{workflow.parameters.a}}", code: codes.InvalidArgument},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := substituteEnumParameters(tc.in, tc.params)
			if tc.code == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, tc.out, out)
				return
			}
			assert.Equal(t, tc.code, status.Code(err), "%v", err)
			if tc.message != "" {
				assert.Equal(t, tc.message, status.Convert(err).Message())
			}
		})
	}
}
