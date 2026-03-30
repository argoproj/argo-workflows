package apiserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	storageutil "github.com/argoproj/argo-workflows/v4/pkg/storage"
	sqlrest "github.com/argoproj/argo-workflows/v4/pkg/storage/rest"
	"github.com/argoproj/argo-workflows/v4/pkg/storage/watch"
)

// SimpleAggregatedServer is a simplified HTTP server that serves the aggregated API
// without the full k8s.io/apiserver framework complexity.
type SimpleAggregatedServer struct {
	db           *gorm.DB
	watchManager *watch.Manager
	scheme       *runtime.Scheme
	port         int
	server       *http.Server
	// stores maps resource name (e.g. "workflows") to its REST storage.
	stores map[string]rest.Storage
}

// apiResource describes a resource for discovery responses.
type apiResource struct {
	name         string
	singularName string
	namespaced   bool
	kind         string
}

var allResources = []apiResource{
	{"workflows", "workflow", true, "Workflow"},
	{"workflows/status", "workflow", true, "Workflow"},
	{"workflowtemplates", "workflowtemplate", true, "WorkflowTemplate"},
	{"clusterworkflowtemplates", "clusterworkflowtemplate", false, "ClusterWorkflowTemplate"},
	{"cronworkflows", "cronworkflow", true, "CronWorkflow"},
	{"cronworkflows/status", "cronworkflow", true, "CronWorkflow"},
	{"workflowtasksets", "workflowtaskset", true, "WorkflowTaskSet"},
	{"workflowtasksets/status", "workflowtaskset", true, "WorkflowTaskSet"},
	{"workflowtaskresults", "workflowtaskresult", true, "WorkflowTaskResult"},
	{"workflowartifactgctasks", "workflowartifactgctask", true, "WorkflowArtifactGCTask"},
	{"workflowartifactgctasks/status", "workflowartifactgctask", true, "WorkflowArtifactGCTask"},
	{"workfloweventbindings", "workfloweventbinding", true, "WorkflowEventBinding"},
}

func NewSimpleAggregatedServer(db *gorm.DB, port int) (*SimpleAggregatedServer, error) {
	scheme := runtime.NewScheme()
	if err := wfv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add to scheme: %w", err)
	}

	wm := watch.NewManager(db)

	stores := make(map[string]rest.Storage)
	for k, v := range sqlrest.NewWorkflowStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewWorkflowTemplateStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewClusterWorkflowTemplateStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewCronWorkflowStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewWorkflowTaskSetStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewWorkflowTaskResultStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewWorkflowArtifactGCTaskStorage(db, wm, scheme) {
		stores[k] = v
	}
	for k, v := range sqlrest.NewWorkflowEventBindingStorage(db, wm, scheme) {
		stores[k] = v
	}

	return &SimpleAggregatedServer{
		db:           db,
		watchManager: wm,
		scheme:       scheme,
		port:         port,
		stores:       stores,
	}, nil
}

func (s *SimpleAggregatedServer) Run(stopCh <-chan struct{}) error {
	fmt.Printf("Starting Simple Aggregated API Server on port %d...\n", s.port)

	mux := http.NewServeMux()

	// Namespaced resource paths
	mux.HandleFunc("/apis/argoproj.io/v1alpha1/namespaces/", s.handleNamespacedRequest)

	// Cluster-scoped resource paths (e.g. clusterworkflowtemplates)
	mux.HandleFunc("/apis/argoproj.io/v1alpha1/", s.handleClusterScopedRequest)

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// API group discovery
	mux.HandleFunc("/apis/argoproj.io", func(w http.ResponseWriter, r *http.Request) {
		apiGroup := &metav1.APIGroup{
			TypeMeta:   metav1.TypeMeta{Kind: "APIGroup", APIVersion: "v1"},
			Name:       "argoproj.io",
			Versions:   []metav1.GroupVersionForDiscovery{{GroupVersion: "argoproj.io/v1alpha1", Version: "v1alpha1"}},
			PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "argoproj.io/v1alpha1", Version: "v1alpha1"},
		}
		writeJSON(w, apiGroup)
	})

	// Top-level API group list
	mux.HandleFunc("/apis", func(w http.ResponseWriter, r *http.Request) {
		apiList := &metav1.APIGroupList{
			TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
			Groups: []metav1.APIGroup{
				{
					Name:     "argoproj.io",
					Versions: []metav1.GroupVersionForDiscovery{{GroupVersion: "argoproj.io/v1alpha1", Version: "v1alpha1"}},
					PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "argoproj.io/v1alpha1", Version: "v1alpha1"},
				},
			},
		}
		writeJSON(w, apiList)
	})

	// Resource list discovery — exact match required before the prefix match above
	mux.HandleFunc("/apis/argoproj.io/v1alpha1", func(w http.ResponseWriter, r *http.Request) {
		verbs := metav1.Verbs{"create", "delete", "get", "list", "patch", "update", "watch"}
		resources := make([]metav1.APIResource, 0, len(allResources))
		for _, res := range allResources {
			// Skip sub-resources in the top-level list
			if strings.Contains(res.name, "/") {
				continue
			}
			resources = append(resources, metav1.APIResource{
				Name:         res.name,
				SingularName: res.singularName,
				Namespaced:   res.namespaced,
				Kind:         res.kind,
				Verbs:        verbs,
			})
		}
		writeJSON(w, &metav1.APIResourceList{
			TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
			GroupVersion: "argoproj.io/v1alpha1",
			APIResources: resources,
		})
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", s.port),
		Handler: mux,
	}

	go func() {
		cert, key, err := generateSelfSignedCert(
			[]string{"argo-server", "argo-server.argo", "argo-server.argo.svc", "localhost"},
			[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")},
		)
		if err != nil {
			fmt.Printf("ERROR: failed to generate cert: %v\n", err)
			return
		}
		tlsCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			fmt.Printf("ERROR: failed to load cert: %v\n", err)
			return
		}
		s.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		}
		listener, err := tls.Listen("tcp", s.server.Addr, s.server.TLSConfig)
		if err != nil {
			fmt.Printf("ERROR: failed to create TLS listener: %v\n", err)
			return
		}
		defer listener.Close()
		fmt.Printf("Simple Aggregated API Server READY and serving on https://0.0.0.0:%d\n", s.port)
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("ERROR: server error: %v\n", err)
		}
	}()

	<-stopCh
	fmt.Println("Shutting down Simple Aggregated API Server...")
	return s.server.Shutdown(context.Background())
}

// handleNamespacedRequest handles /apis/argoproj.io/v1alpha1/namespaces/{ns}/{resource}[/{name}][/{subresource}]
func (s *SimpleAggregatedServer) handleNamespacedRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/apis/argoproj.io/v1alpha1/namespaces/")
	parts := strings.SplitN(path, "/", 4)
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]
	resource := parts[1]
	var name, subresource string
	if len(parts) > 2 {
		name = parts[2]
	}
	if len(parts) > 3 {
		subresource = parts[3]
		resource = resource + "/" + subresource
	}

	ctx := genericapirequest.WithNamespace(context.Background(), namespace)
	s.dispatch(w, r, ctx, namespace, resource, name)
}

// handleClusterScopedRequest handles /apis/argoproj.io/v1alpha1/{resource}[/{name}]
// (only reached for paths that don't match the more-specific namespaced prefix).
func (s *SimpleAggregatedServer) handleClusterScopedRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/apis/argoproj.io/v1alpha1/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(path, "/", 3)
	resource := parts[0]
	var name string
	if len(parts) > 1 {
		name = parts[1]
	}

	ctx := context.Background()
	s.dispatch(w, r, ctx, "", resource, name)
}

// dispatch routes a parsed request to the correct store and HTTP verb handler.
func (s *SimpleAggregatedServer) dispatch(w http.ResponseWriter, r *http.Request, ctx context.Context, namespace, resource, name string) {
	fmt.Printf("DEBUG: %s /apis/argoproj.io/v1alpha1 ns=%q resource=%q name=%q query=%q\n", r.Method, namespace, resource, name, r.URL.RawQuery)

	store, ok := s.stores[resource]
	if !ok {
		http.Error(w, fmt.Sprintf("resource %q not found", resource), http.StatusNotFound)
		return
	}

	q := r.URL.Query()

	// Watch support: GET with ?watch=true
	if r.Method == http.MethodGet && q.Get("watch") == "true" {
		watcher, ok := store.(rest.Watcher)
		if !ok {
			http.Error(w, "watch not supported", http.StatusMethodNotAllowed)
			return
		}
		lister, _ := store.(rest.Lister)
		opts := &metainternalversion.ListOptions{}
		if rv := q.Get("resourceVersion"); rv != "" {
			opts.ResourceVersion = rv
		}
		sendInitialEvents := q.Get("sendInitialEvents") == "true"
		// Derive the correct kind for BOOKMARK events from the allResources table.
		kind := "Workflow"
		for _, res := range allResources {
			if res.name == resource {
				kind = res.kind
				break
			}
		}
		s.serveWatch(w, r, ctx, watcher, lister, opts, sendInitialEvents, kind)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if name != "" {
			getter, ok := store.(rest.Getter)
			if !ok {
				http.Error(w, "get not supported", http.StatusMethodNotAllowed)
				return
			}
			obj, err := getter.Get(ctx, name, &metav1.GetOptions{})
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, obj)
		} else {
			lister, ok := store.(rest.Lister)
			if !ok {
				http.Error(w, "list not supported", http.StatusMethodNotAllowed)
				return
			}
			obj, err := lister.List(ctx, &metainternalversion.ListOptions{})
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, obj)
		}

	case http.MethodPost:
		creater, ok := store.(rest.Creater)
		if !ok {
			http.Error(w, "create not supported", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
			return
		}
		obj := store.New()
		if err := json.Unmarshal(body, obj); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode body: %v", err), http.StatusBadRequest)
			return
		}
		// Ensure namespace and generated name are set.
		if accessor, err2 := meta.Accessor(obj); err2 == nil {
			if namespace != "" && accessor.GetNamespace() == "" {
				accessor.SetNamespace(namespace)
			}
			if accessor.GetName() == "" && accessor.GetGenerateName() != "" {
				accessor.SetName(accessor.GetGenerateName() + randomSuffix(5))
			}
		}
		created, err := creater.Create(ctx, obj, nil, &metav1.CreateOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		updater, ok := store.(rest.Updater)
		if !ok {
			http.Error(w, "update not supported", http.StatusMethodNotAllowed)
			return
		}
		if name == "" {
			http.Error(w, "name required for update", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
			return
		}
		obj := store.New()
		if err := json.Unmarshal(body, obj); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode body: %v", err), http.StatusBadRequest)
			return
		}
		updated, _, err := updater.Update(ctx, name, rest.DefaultUpdatedObjectInfo(obj), nil, nil, false, &metav1.UpdateOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, updated)

	case http.MethodPatch:
		updater, ok := store.(rest.Updater)
		if !ok {
			http.Error(w, "patch not supported", http.StatusMethodNotAllowed)
			return
		}
		if name == "" {
			http.Error(w, "name required for patch", http.StatusBadRequest)
			return
		}
		patchBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
			return
		}
		// Fetch current object to apply patch onto.
		getter, ok2 := store.(rest.Getter)
		if !ok2 {
			http.Error(w, "get not supported (required for patch)", http.StatusMethodNotAllowed)
			return
		}
		existing, err := getter.Get(ctx, name, &metav1.GetOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		existingBytes, err := json.Marshal(existing)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal existing object: %v", err), http.StatusInternalServerError)
			return
		}
		patchedBytes, err := strategicpatch.StrategicMergePatch(existingBytes, patchBytes, existing)
		if err != nil {
			// Fall back to plain merge patch
			patchedBytes, err = applyMergePatch(existingBytes, patchBytes)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to apply patch: %v", err), http.StatusBadRequest)
				return
			}
		}
		obj := store.New()
		if err := json.Unmarshal(patchedBytes, obj); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode patched object: %v", err), http.StatusBadRequest)
			return
		}
		updated, _, err := updater.Update(ctx, name, rest.DefaultUpdatedObjectInfo(obj), nil, nil, false, &metav1.UpdateOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, updated)

	case http.MethodDelete:
		deleter, ok := store.(rest.GracefulDeleter)
		if !ok {
			http.Error(w, "delete not supported", http.StatusMethodNotAllowed)
			return
		}
		if name == "" {
			http.Error(w, "name required for delete", http.StatusBadRequest)
			return
		}
		obj, _, err := deleter.Delete(ctx, name, nil, &metav1.DeleteOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, obj)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// serveWatch streams watch events as newline-delimited JSON using the k8s wire format.
// The wire format is: {"type":"ADDED","object":{...}}.
// When sendInitialEvents is true (WatchList protocol, k8s 1.27+), existing objects are
// sent as ADDED events first, followed by a BOOKMARK with "k8s.io/initial-events-end".
func (s *SimpleAggregatedServer) serveWatch(w http.ResponseWriter, r *http.Request, ctx context.Context, watcher rest.Watcher, lister rest.Lister, opts *metainternalversion.ListOptions, sendInitialEvents bool, kind string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	wi, err := watcher.Watch(ctx, opts)
	if err != nil {
		writeError(w, err)
		return
	}
	defer wi.Stop()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	rv, _ := storageutil.CurrentResourceVersion(s.db)
	rvStr := fmt.Sprintf("%d", rv)

	bookmarkObj := func(rvs string, endOfInitial bool) json.RawMessage {
		if endOfInitial {
			return json.RawMessage(fmt.Sprintf(
				`{"apiVersion":"argoproj.io/v1alpha1","kind":%q,"metadata":{"resourceVersion":%q,"annotations":{"k8s.io/initial-events-end":"true"}}}`,
				kind, rvs,
			))
		}
		return json.RawMessage(fmt.Sprintf(
			`{"apiVersion":"argoproj.io/v1alpha1","kind":%q,"metadata":{"resourceVersion":%q}}`,
			kind, rvs,
		))
	}

	if sendInitialEvents && lister != nil {
		// WatchList protocol: send all existing objects as ADDED, then end BOOKMARK.
		list, err := lister.List(ctx, &metainternalversion.ListOptions{})
		if err == nil {
			items, _ := meta.ExtractList(list)
			for _, item := range items {
				objBytes, err := json.Marshal(item)
				if err != nil {
					continue
				}
				_ = enc.Encode(watchEvent{Type: "ADDED", Object: json.RawMessage(objBytes)})
			}
		}
		_ = enc.Encode(watchEvent{Type: "BOOKMARK", Object: bookmarkObj(rvStr, true)})
	} else {
		_ = enc.Encode(watchEvent{Type: "BOOKMARK", Object: bookmarkObj(rvStr, false)})
	}
	flusher.Flush()

	// Periodically send BOOKMARKs to keep the watch alive and allow reconnection.
	bookmarkTicker := time.NewTicker(30 * time.Second)
	defer bookmarkTicker.Stop()

	for {
		select {
		case event, open := <-wi.ResultChan():
			if !open {
				return
			}
			objBytes, err := json.Marshal(event.Object)
			if err != nil {
				continue
			}
			_ = enc.Encode(watchEvent{
				Type:   string(event.Type),
				Object: json.RawMessage(objBytes),
			})
			flusher.Flush()
		case <-bookmarkTicker.C:
			if rv2, err2 := storageutil.CurrentResourceVersion(s.db); err2 == nil {
				_ = enc.Encode(watchEvent{Type: "BOOKMARK", Object: bookmarkObj(fmt.Sprintf("%d", rv2), false)})
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// watchEvent is the k8s watch wire format with proper JSON field names.
type watchEvent struct {
	Type   string          `json:"type"`
	Object json.RawMessage `json:"object"`
}

// writeJSON writes obj as JSON with application/json content type.
func writeJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(obj)
}

// writeError maps k8s API errors to appropriate HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	type statusError interface{ Status() metav1.Status }
	if se, ok := err.(statusError); ok {
		s := se.Status()
		if s.Code != 0 {
			code = int(s.Code)
		}
	}
	http.Error(w, err.Error(), code)
}

// applyMergePatch performs a simple JSON merge patch (RFC 7396).
func applyMergePatch(base, patch []byte) ([]byte, error) {
	var baseMap, patchMap map[string]interface{}
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, err
	}
	mergeMaps(baseMap, patchMap)
	return json.Marshal(baseMap)
}

func mergeMaps(dst, src map[string]interface{}) {
	for k, v := range src {
		if v == nil {
			delete(dst, k)
			continue
		}
		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := dst[k].(map[string]interface{}); ok {
				mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

func randomSuffix(n int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + rnd.Intn(26))
	}
	return string(b)
}
