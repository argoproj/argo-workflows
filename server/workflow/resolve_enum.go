package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilcache "k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/jsonpath"

	"github.com/argoproj/argo-workflows/v4/errors"
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
	"github.com/argoproj/argo-workflows/v4/util/env"
)

const (
	defaultEnumMaxItems = 500
	defaultEnumCacheTTL = 10 * time.Second
	// Which resource types the cluster serves changes rarely and is the same for every user.
	enumDiscoveryCacheTTL = 5 * time.Minute
	enumCacheSize         = 1000
	enumListTimeout       = 15 * time.Second
)

var secretsResource = schema.GroupResource{Group: "", Resource: "secrets"}

// enumResolver limits and caches the Kubernetes calls behind ResolveParameterEnum, so opening a
// submit form repeatedly, or on a large cluster, doesn't hammer the API server. It caches at three levels:
//
//   - whole answers per caller and request, so a repeated request makes no Kubernetes calls at all
//   - list results per caller and query, shared between parameters and templates that run the same query
//   - API discovery per kind, shared between callers, because which resource types exist is not user-specific
//
// Only successes are cached. A cache TTL of 0 disables all three.
type enumResolver struct {
	maxItems int
	cacheTTL time.Duration
	cache    *utilcache.LRUExpireCache
	inflight singleflight.Group
}

func newEnumResolver(ctx context.Context) *enumResolver {
	maxItems := env.LookupEnvIntOr(ctx, "ARGO_SERVER_RESOLVE_ENUM_MAX_ITEMS", defaultEnumMaxItems)
	if maxItems <= 0 {
		maxItems = defaultEnumMaxItems
	}
	return &enumResolver{
		maxItems: maxItems,
		cacheTTL: env.LookupEnvDurationOr(ctx, "ARGO_SERVER_RESOLVE_ENUM_CACHE_TTL", defaultEnumCacheTTL),
		cache:    utilcache.NewLRUExpireCache(enumCacheSize),
	}
}

// enumQuery is one list call plus the JSONPath applied to its items.
type enumQuery struct {
	gvr           schema.GroupVersionResource
	namespace     string // empty for cluster-scoped resources
	labelSelector string
	jsonPath      string
}

// cached returns the value stored under key, or runs fn and stores its result for ttl if it succeeds.
// Identical calls that arrive while fn is running share its result, e.g. from several open forms.
func (r *enumResolver) cached(key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	if r.cacheTTL <= 0 {
		return fn()
	}
	if v, ok := r.cache.Get(key); ok {
		return v, nil
	}
	v, err, _ := r.inflight.Do(key, func() (any, error) {
		v, err := fn()
		if err == nil {
			r.cache.Add(key, v, ttl)
		}
		return v, err
	})
	return v, err
}

// listCached lists the resources of q, reusing a recent result for the same caller and query.
// Results depend on the caller's RBAC, so the key includes who is asking: callerKey identifies
// the credentials, and requestNamespace selects the service account in SSO RBAC.
func (r *enumResolver) listCached(ctx context.Context, client dynamic.Interface, callerKey, requestNamespace string, q enumQuery, jp *jsonpath.JSONPath) (*workflowpkg.ResolveParameterEnumResponse, error) {
	key := hashKey("list", callerKey, requestNamespace, q.gvr.String(), q.namespace, q.labelSelector, q.jsonPath)
	v, err := r.cached(key, r.cacheTTL, func() (any, error) { return r.list(ctx, client, q, jp) })
	if err != nil {
		return nil, err
	}
	return v.(*workflowpkg.ResolveParameterEnumResponse), nil
}

type enumResource struct {
	gvr        schema.GroupVersionResource
	namespaced bool
}

// discoverCached finds the resource for a kind, reusing a recent discovery result.
// Failures are not cached, so a newly installed CRD is found straight away.
func (r *enumResolver) discoverCached(disc discovery.DiscoveryInterface, apiVersion, kind string) (enumResource, error) {
	v, err := r.cached(hashKey("discovery", apiVersion, kind), enumDiscoveryCacheTTL, func() (any, error) {
		gvr, namespaced, err := resolveEnumResource(disc, apiVersion, kind)
		return enumResource{gvr: gvr, namespaced: namespaced}, err
	})
	if err != nil {
		return enumResource{}, err
	}
	return v.(enumResource), nil
}

// requestCacheKey identifies a request and its caller. Parameter values are part of it because
// cascading lookups depend on them.
func requestCacheKey(callerKey string, req *workflowpkg.ResolveParameterEnumRequest) string {
	parts := []string{"request", callerKey, req.Namespace, req.ResourceKind, req.ResourceName, req.TemplateName, req.ParameterName}
	names := make([]string, 0, len(req.Parameters))
	for name := range req.Parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		parts = append(parts, name, req.Parameters[name])
	}
	return hashKey(parts...)
}

func (r *enumResolver) list(ctx context.Context, client dynamic.Interface, q enumQuery, jp *jsonpath.JSONPath) (*workflowpkg.ResolveParameterEnumResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, enumListTimeout)
	defer cancel()

	var resourceIntf dynamic.ResourceInterface = client.Resource(q.gvr)
	if q.namespace != "" {
		resourceIntf = client.Resource(q.gvr).Namespace(q.namespace)
	}
	// Ask the API server for one page only; a continue token means there were more items.
	list, err := resourceIntf.List(ctx, metav1.ListOptions{LabelSelector: q.labelSelector, Limit: int64(r.maxItems)})
	if err != nil {
		return nil, err
	}
	items := list.Items
	truncated := list.GetContinue() != ""
	if len(items) > r.maxItems { // not every API server honours limit
		items = items[:r.maxItems]
		truncated = true
	}

	seen := map[string]bool{}
	values := []string{}
	for _, item := range items {
		results, err := jp.FindResults(item.Object)
		if err != nil {
			continue // the path does not apply to this item
		}
		for _, res := range results {
			for _, val := range res {
				if !val.IsValid() || !val.CanInterface() || val.Interface() == nil {
					continue
				}
				v := fmt.Sprintf("%v", val.Interface())
				if v == "" || seen[v] {
					continue
				}
				if len(values) == r.maxItems { // a JSONPath can return many values per item
					truncated = true
					continue
				}
				seen[v] = true
				values = append(values, v)
			}
		}
	}
	sort.Strings(values)
	return &workflowpkg.ResolveParameterEnumResponse{Values: values, Truncated: truncated}, nil
}

func hashKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ResolveParameterEnum resolves the enumFrom of a parameter defined on a stored WorkflowTemplate,
// ClusterWorkflowTemplate, CronWorkflow or Workflow, and returns the values extracted with its JSONPath.
//
// The enumFrom is read from the stored object, never from the request, so callers can only run lookups a
// template author defined. Both the object read and the resource list use the caller's own credentials,
// so Kubernetes RBAC applies to each. Secrets can never be listed. Results are capped and briefly cached.
func (s *workflowServer) ResolveParameterEnum(ctx context.Context, req *workflowpkg.ResolveParameterEnumRequest) (*workflowpkg.ResolveParameterEnumResponse, error) {
	if req.ParameterName == "" {
		return nil, sutils.ToStatusError(errors.Errorf(errors.CodeBadRequest, "parameterName is required"), codes.InvalidArgument)
	}
	callerKey := auth.CallerKey(ctx)
	// A template edit becomes visible once the cached answer expires.
	v, err := s.enumResolver.cached(requestCacheKey(callerKey, req), s.enumResolver.cacheTTL, func() (any, error) {
		return s.resolveParameterEnum(ctx, callerKey, req)
	})
	if err != nil {
		return nil, err
	}
	return v.(*workflowpkg.ResolveParameterEnumResponse), nil
}

func (s *workflowServer) resolveParameterEnum(ctx context.Context, callerKey string, req *workflowpkg.ResolveParameterEnumRequest) (*workflowpkg.ResolveParameterEnumResponse, error) {
	spec, err := s.getSpecForEnumResolution(ctx, req.Namespace, req.ResourceKind, req.ResourceName)
	if err != nil {
		return nil, err
	}
	enumFrom, err := findParameterEnumFrom(spec, req.TemplateName, req.ParameterName)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	jp := jsonpath.New("enumFrom").AllowMissingKeys(true)
	if err = jp.Parse(enumFrom.JSONPath); err != nil {
		return nil, sutils.ToStatusError(errors.Errorf(errors.CodeBadRequest, "enumFrom.jsonPath invalid: %v", err), codes.InvalidArgument)
	}

	res, err := s.enumResolver.discoverCached(auth.GetKubeClient(ctx).Discovery(), enumFrom.Resource.APIVersion, enumFrom.Resource.Kind)
	if err != nil {
		// An unknown kind is a bad-request error and stays InvalidArgument; a failing discovery call is not the caller's fault.
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	gvr, namespaced := res.gvr, res.namespaced
	if gvr.GroupResource() == secretsResource {
		return nil, status.Error(codes.PermissionDenied, "enumFrom cannot list Secrets")
	}

	q := enumQuery{gvr: gvr, jsonPath: enumFrom.JSONPath}
	if q.labelSelector, err = substituteEnumParameters(enumFrom.LabelSelector, req.Parameters); err != nil {
		return nil, err
	}
	if namespaced {
		q.namespace = req.Namespace
		if enumFrom.Resource.Namespace != "" {
			if q.namespace, err = substituteEnumParameters(enumFrom.Resource.Namespace, req.Parameters); err != nil {
				return nil, err
			}
		}
		if q.namespace == "" {
			return nil, sutils.ToStatusError(errors.Errorf(errors.CodeBadRequest, "namespace is required to list %s", gvr.Resource), codes.InvalidArgument)
		}
	}

	resp, err := s.enumResolver.listCached(ctx, auth.GetDynamicClient(ctx), callerKey, req.Namespace, q, jp)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return resp, nil
}

// getSpecForEnumResolution reads the object holding the parameter with the caller's own client.
func (s *workflowServer) getSpecForEnumResolution(ctx context.Context, namespace, kind, name string) (*wfv1.WorkflowSpec, error) {
	if name == "" {
		return nil, sutils.ToStatusError(errors.Errorf(errors.CodeBadRequest, "resourceName is required"), codes.InvalidArgument)
	}
	wfClient := auth.GetWfClient(ctx).ArgoprojV1alpha1()
	var obj metav1.Object
	var spec *wfv1.WorkflowSpec
	switch kind {
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		wftmpl, err := wfClient.WorkflowTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		obj, spec = wftmpl, &wftmpl.Spec
	case workflow.ClusterWorkflowTemplateKind, workflow.ClusterWorkflowTemplateSingular, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateShortName:
		cwftmpl, err := wfClient.ClusterWorkflowTemplates().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		obj, spec = cwftmpl, &cwftmpl.Spec
	case workflow.CronWorkflowKind, workflow.CronWorkflowSingular, workflow.CronWorkflowPlural, workflow.CronWorkflowShortName:
		cronWf, err := wfClient.CronWorkflows(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		obj, spec = cronWf, &cronWf.Spec.WorkflowSpec
	case workflow.WorkflowKind, workflow.WorkflowSingular, workflow.WorkflowPlural, workflow.WorkflowShortName:
		wf, err := wfClient.Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		// A workflow submitted from a template keeps the template's parameters in its stored spec.
		obj, spec = wf, wf.GetExecSpec()
	default:
		return nil, sutils.ToStatusError(errors.Errorf(errors.CodeBadRequest, "resource kind '%s' is not supported for resolving parameter enums", kind), codes.InvalidArgument)
	}
	if err := s.instanceIDService.Validate(obj); err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return spec, nil
}

// findParameterEnumFrom finds a parameter's enumFrom in spec.arguments, or in a template's inputs if templateName is set.
func findParameterEnumFrom(spec *wfv1.WorkflowSpec, templateName, parameterName string) (*wfv1.EnumFrom, error) {
	params := spec.Arguments.Parameters
	if templateName != "" {
		params = nil
		for _, t := range spec.Templates {
			if t.Name == templateName {
				params = t.Inputs.Parameters
				break
			}
		}
		if params == nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "template %q not found or has no input parameters", templateName)
		}
	}
	for _, p := range params {
		if p.Name == parameterName {
			if p.EnumFrom == nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "parameter %q has no enumFrom", parameterName)
			}
			return p.EnumFrom, nil
		}
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "parameter %q not found", parameterName)
}

// resolveEnumResource uses API discovery to find the resource name for a kind and whether it is namespaced.
// Guessing the plural from the kind breaks for kinds like Ingress or Endpoints.
func resolveEnumResource(disc discovery.DiscoveryInterface, apiVersion, kind string) (schema.GroupVersionResource, bool, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, false, errors.Errorf(errors.CodeBadRequest, "enumFrom.resource.apiVersion invalid: %v", err)
	}
	resources, err := disc.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		if apierr.IsNotFound(err) {
			return schema.GroupVersionResource{}, false, errors.Errorf(errors.CodeBadRequest, "enumFrom.resource.apiVersion %q is not served by the cluster", apiVersion)
		}
		return schema.GroupVersionResource{}, false, err
	}
	for _, r := range resources.APIResources {
		if r.Kind == kind && !strings.Contains(r.Name, "/") {
			return gv.WithResource(r.Name), r.Namespaced, nil
		}
	}
	return schema.GroupVersionResource{}, false, errors.Errorf(errors.CodeBadRequest, "enumFrom.resource kind %q not found in %s", kind, apiVersion)
}

// parameterReference matches {{workflow.parameters.<name>}}, allowing spaces inside the braces like the UI does.
var parameterReference = regexp.MustCompile(`{{\s*workflow\.parameters\.([^}\s]+)\s*}}`)

// substituteEnumParameters replaces {{workflow.parameters.<name>}} with the given values. A parameter that is
// missing or empty has no value yet, so it fails with FailedPrecondition naming the parameters it waits for;
// the UI shows that as a note rather than an error. Any other {{...}} reference can never resolve here, so it
// fails with InvalidArgument.
func substituteEnumParameters(s string, params map[string]string) (string, error) {
	if strings.Contains(parameterReference.ReplaceAllString(s, ""), "{{") {
		return "", status.Errorf(codes.InvalidArgument, "only {{workflow.parameters.<name>}} references are supported, got %q", s)
	}
	var waiting []string
	out := parameterReference.ReplaceAllStringFunc(s, func(ref string) string {
		name := parameterReference.FindStringSubmatch(ref)[1]
		if v := params[name]; v != "" {
			return v
		}
		if !slices.Contains(waiting, name) {
			waiting = append(waiting, name)
		}
		return ref
	})
	if len(waiting) > 0 {
		return "", status.Errorf(codes.FailedPrecondition, "waiting for a value for %s", strings.Join(waiting, ", "))
	}
	return out, nil
}
