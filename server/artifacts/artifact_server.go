package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	artifact "github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

var (
	errPermissionDenied = fmt.Errorf("permission denied")
	errNotFound         = fmt.Errorf("artifact not found")
)

type ArtifactServer struct {
	gatekeeper           auth.Gatekeeper
	hydrator             hydrator.Interface
	wfArchive            sqldb.WorkflowArchive
	instanceIDService    instanceid.Service
	artDriverFactory     artifact.NewDriverFunc
	artifactRepositories artifactrepositories.Interface
}

func NewArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artifactRepositories artifactrepositories.Interface) *ArtifactServer {
	return newArtifactServer(authN, hydrator, wfArchive, instanceIDService, artifact.NewDriver, artifactRepositories)
}

func newArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artDriverFactory artifact.NewDriverFunc, artifactRepositories artifactrepositories.Interface) *ArtifactServer {
	return &ArtifactServer{authN, hydrator, wfArchive, instanceIDService, artDriverFactory, artifactRepositories}
}

func (a *ArtifactServer) HandlerFunc() http.HandlerFunc {
	h := mux.NewRouter()
	h.HandleFunc("/workflow-artifacts/v2/artifact-descriptions/{namespace}/{idDiscrim}/{id}/{nodeId}/{artifactDiscrim}/{artifactName}", a.getArtifactDescription)
	h.HandleFunc("/workflow-artifacts/v2/artifacts/{namespace}/{idDiscrim}/{id}/{nodeId}/{artifactDiscrim}/{artifactName}", a.getArtifact)
	h.HandleFunc("/workflow-artifacts/v2/artifact-items/{namespace}/{idDiscrim}/{id}/{nodeId}/{artifactDiscrim}/{artifactName}/{item}", a.getArtifactItem)
	return func(w http.ResponseWriter, r *http.Request) {
		log.WithField("url", r.URL).Info("artifact server request")
		h.ServeHTTP(w, r)
	}
}

func (a *ArtifactServer) gateKeeping(r *http.Request, ns types.NamespacedRequest) (context.Context, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		cookie, err := r.Cookie("authorization")
		if err != nil {
			if err != http.ErrNoCookie {
				return nil, err
			}
		} else {
			token = cookie.Value
		}
	}
	ctx := metadata.NewIncomingContext(r.Context(), metadata.MD{"authorization": []string{token}})
	return a.gatekeeper.ContextWithRequest(ctx, ns)
}

func (a *ArtifactServer) unauthorizedError(w http.ResponseWriter) {
	w.WriteHeader(401)
}

func (a *ArtifactServer) error(err error, w http.ResponseWriter) {
	log.WithError(err).Error("failed artifact server request")
	if errors.Is(err, errPermissionDenied) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	} else if errors.Is(err, errNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (a *ArtifactServer) downloadArtifact(ctx context.Context, wf *wfv1.Workflow, nodeId string, artifactDiscrim string, artifactName string) (*os.File, string, error) {
	kubeClient := auth.GetKubeClient(ctx)

	var art *wfv1.Artifact
	switch artifactDiscrim {
	case "input":
		art = wf.Status.Nodes[nodeId].Inputs.GetArtifactByName(artifactName)
	case "output":
		art = wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	default:
		return nil, "", fmt.Errorf("invalid artifact discriminator %q", artifactDiscrim)
	}
	if art == nil {
		return nil, "", errNotFound
	}

	ar, err := a.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return nil, "", err
	}
	l := ar.ToArtifactLocation()
	err = art.Relocate(l)
	if err != nil {
		return nil, "", err
	}

	key, _ := art.GetKey()

	driver, err := a.artDriverFactory(ctx, art, resources{kubeClient, wf.Namespace})
	if err != nil {
		return nil, "", err
	}

	tmpPath := filepath.Join("/tmp", "artifact-"+rand.String(32))
	err = driver.Load(art, tmpPath)
	if err != nil {
		return nil, "", err
	}

	file, err := os.Open(tmpPath)
	if err != nil {
		return nil, "", err
	}

	return file, key, nil
}

func (a *ArtifactServer) Redirect(w http.ResponseWriter, r *http.Request) {
	requestPath := strings.Split(r.URL.Path, "/")
	pathDiscrim := requestPath[1]

	pathParts := map[string]int{
		"artifacts":              6,
		"input-artifacts":        6,
		"artifacts-by-uid":       5,
		"input-artifacts-by-uid": 5,
	}[pathDiscrim]

	if len(requestPath) != pathParts {
		http.Error(w, "request path is not valid", http.StatusBadRequest)
		return
	}

	var namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName string
	switch pathDiscrim {
	case "artifacts":
		// "/artifacts/{namespace}/{name}/{nodeId}/{artifactName}"
		namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName = requestPath[2], "name", requestPath[3], requestPath[4], "output", requestPath[5]
	case "input-artifacts":
		// "/input-artifacts/{namespace}/{name}/{nodeId}/{artifactName}"
		namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName = requestPath[2], "name", requestPath[3], requestPath[4], "input", requestPath[5]
	case "artifacts-by-uid":
		// "/artifacts-by-uid/{uid}/{nodeId}/{artifactName}"
		namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName = "", "uid", requestPath[2], requestPath[3], "output", requestPath[4]
	case "input-artifacts-by-uid":
		// "/input-artifacts-by-uid/{uid}/{nodeId}/{artifactName}"
		namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName = "", "uid", requestPath[2], requestPath[3], "input", requestPath[4]
	}

	if idDiscrim == "uid" {
		wf, err := a.wfArchive.GetWorkflow(id)
		if err != nil {
			a.error(err, w)
			return
		}
		namespace = wf.GetNamespace()
	}
	http.Redirect(w, r, fmt.Sprintf("/workflow-artifacts/v2/artifacts/%s/%s/%s/%s/%s/%s", namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName), http.StatusMovedPermanently)
}

func (a *ArtifactServer) getArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifactItem(w, r)
}

func (a *ArtifactServer) getArtifactItem(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName := vars["namespace"], vars["idDiscrim"], vars["id"], vars["nodeId"], vars["artifactDiscrim"], vars["artifactName"]
	item := vars["item"]

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	wf, err := a.getWorkflowAndValidate(ctx, namespace, idDiscrim, id)
	if err != nil {
		a.error(err, w)
		return
	}

	file, key, err := a.downloadArtifact(ctx, wf, nodeId, artifactDiscrim, artifactName)
	if err != nil {
		a.error(err, w)
		return
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	if item == "" {
		w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))
		http.ServeContent(w, r, "", time.Time{}, file)
	} else {
		gr, err := gzip.NewReader(file)
		if err != nil {
			a.error(err, w)
			return
		}
		tr := tar.NewReader(gr)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if header.Name != item {
				continue
			}
			w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(item)))
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, tr)
			return
		}
	}
}

func (a *ArtifactServer) getWorkflowAndValidate(ctx context.Context, namespace, idDiscrim, id string) (*wfv1.Workflow, error) {
	var wf *wfv1.Workflow
	var err error
	switch idDiscrim {
	case "uid":
		wf, err = a.wfArchive.GetWorkflow(id)
		if err != nil {
			return nil, err
		}
		if err := a.validateAccess(ctx, wf); err != nil {
			return nil, err
		}
	case "name":
		wfClient := auth.GetWfClient(ctx)
		wf, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid ID discriminator %q", idDiscrim)
	}
	if err := a.instanceIDService.Validate(wf); err != nil {
		return nil, err
	}
	if err := a.hydrator.Hydrate(wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (a *ArtifactServer) validateAccess(ctx context.Context, wf *wfv1.Workflow) error {
	allowed, err := auth.CanI(ctx, "get", "workflows", wf.Namespace, wf.Name)
	if err != nil {
		return err
	}
	if !allowed {
		return errPermissionDenied
	}
	return nil
}

type Item struct {
	// Name is the file name within the archive
	Name        string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType,omitempty"`
}

type artifactDescription struct {
	Filename    string `json:"filename,omitempty"`
	Items       []Item `json:"items,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

func (a *ArtifactServer) getArtifactDescription(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	namespace, idDiscrim, id, nodeId, artifactDiscrim, artifactName := vars["namespace"], vars["idDiscrim"], vars["id"], vars["nodeId"], vars["artifactDiscrim"], vars["artifactName"]

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	wf, err := a.getWorkflowAndValidate(ctx, namespace, idDiscrim, id)
	if err != nil {
		a.error(err, w)
		return
	}

	file, key, err := a.downloadArtifact(ctx, wf, nodeId, artifactDiscrim, artifactName)
	if err != nil {
		a.error(err, w)
		return
	}
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	d := &artifactDescription{
		Filename:    filepath.Base(key),
		ContentType: mime.TypeByExtension(filepath.Ext(key)),
	}

	if strings.HasSuffix(key, ".tgz") {
		gr, err := gzip.NewReader(file)
		if err != nil {
			a.error(err, w)
			return
		}
		tr := tar.NewReader(gr)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				a.error(err, w)
				return
			}
			d.Items = append(d.Items, Item{
				Name:        header.Name,
				Size:        header.FileInfo().Size(),
				ContentType: mime.TypeByExtension(filepath.Ext(header.Name)),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(d)
}
