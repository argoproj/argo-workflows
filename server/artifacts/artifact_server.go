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

var errPermissionDenied = fmt.Errorf("permission denied")

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

func (a *ArtifactServer) serverInternalError(err error, w http.ResponseWriter) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (a *ArtifactServer) downloadArtifact(ctx context.Context, wf *wfv1.Workflow, nodeId string, isInput bool, artifactName string) (*os.File, string, error) {
	kubeClient := auth.GetKubeClient(ctx)

	var art *wfv1.Artifact
	if isInput {
		art = wf.Status.Nodes[nodeId].Inputs.GetArtifactByName(artifactName)
	} else {
		art = wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	}
	if art == nil {
		return nil, "", fmt.Errorf("artifact not found")
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

func (a *ArtifactServer) DownloadArtifact(w http.ResponseWriter, r *http.Request) {
	requestPath := strings.Split(r.URL.Path, "/")
	basePath := requestPath[1]

	minPathParts := map[string]int{
		"artifact-downloads":     7,
		"artifacts":              6,
		"input-artifacts":        6,
		"artifacts-by-uid":       5,
		"input-artifacts-by-uid": 5,
	}[basePath]

	if len(requestPath) < minPathParts {
		http.Error(w, "request path is not valid", http.StatusBadRequest)
		return
	}

	var namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName, item string
	switch basePath {
	case "artifact-downloads":
		// "/artifacts-downloads/{namespace}/{idDiscriminator}/{id}/{nodeId}/{artifactDiscriminator}/{artifactName}"
		namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName = requestPath[2], requestPath[3], requestPath[4], requestPath[5], requestPath[6], requestPath[7]
		if len(requestPath) == 9 {
			item = requestPath[8]
		}
	case "artifacts":
		// "/artifacts/{namespace}/{name}/{nodeId}/{artifactName}"
		namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName = requestPath[2], "name", requestPath[3], requestPath[4], "output", requestPath[5]
	case "input-artifacts":
		// "/input-artifacts/{namespace}/{name}/{nodeId}/{artifactName}"
		namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName = requestPath[2], "name", requestPath[3], requestPath[4], "input", requestPath[5]
	case "artifacts-by-uid":
		// "/artifacts-by-uid/{uid}/{nodeId}/{artifactName}"
		namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName = "???", "uid", requestPath[2], requestPath[3], "output", requestPath[4]
	case "input-artifacts-by-uid":
		// "/input-artifacts-by-uid/{uid}/{nodeId}/{artifactName}"
		namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName = "???", "uid", requestPath[2], requestPath[3], "input", requestPath[4]
	}

	if namespace == "???" && idDiscriminator == "uid" {
		wf, err := a.wfArchive.GetWorkflow(id)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
		namespace = wf.GetNamespace()
	}

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	wf, err := a.getWorkflowAndValidate(ctx, namespace, idDiscriminator, id)
	if errors.Is(err, errPermissionDenied) {
		a.unauthorizedError(w)
		return
	}
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	file, key, err := a.downloadArtifact(ctx, wf, nodeId, artifactDiscriminator == "input", artifactName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	defer os.Remove(file.Name())

	if item == "" {
		w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))
		http.ServeContent(w, r, "", time.Time{}, file)
	} else {
		gr, err := gzip.NewReader(file)
		if err != nil {
			a.serverInternalError(err, w)
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

func (a *ArtifactServer) getWorkflowAndValidate(ctx context.Context, namespace, idDiscriminator, id string) (*wfv1.Workflow, error) {
	var wf *wfv1.Workflow
	var err error
	if idDiscriminator == "uid" {
		wf, err = a.wfArchive.GetWorkflow(id)
		if err != nil {
			return nil, err
		}
		if err := a.validateAccess(ctx, wf); err != nil {
			return nil, err
		}
	} else {
		wfClient := auth.GetWfClient(ctx)
		wf, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}
	if err := a.instanceIDService.Validate(wf); err != nil {
		return nil, err
	}
	if err := a.hydrator.Hydrate(wf); err != nil {
		return nil, err
	}
	return wf, err
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
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type artifactDescription struct {
	Key         string `json:"key,omitempty"`
	Items       []Item `json:"items,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

func (a *ArtifactServer) GetArtifactDescription(w http.ResponseWriter, r *http.Request) {
	requestPath := strings.Split(r.URL.Path, "/")
	if len(requestPath) != 8 {
		a.serverInternalError(errors.New("request path is not valid"), w)
		return
	}
	namespace, idDiscriminator, id, nodeId, artifactDiscriminator, artifactName := requestPath[2], requestPath[3], requestPath[4], requestPath[5], requestPath[6], requestPath[7]

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	wf, err := a.getWorkflowAndValidate(ctx, namespace, idDiscriminator, id)
	if errors.Is(err, errPermissionDenied) {
		a.unauthorizedError(w)
		return
	}
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	file, key, err := a.downloadArtifact(ctx, wf, nodeId, artifactDiscriminator == "input", artifactName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}
	defer os.Remove(file.Name())

	d := &artifactDescription{
		Key:         key,
		ContentType: mime.TypeByExtension(filepath.Ext(key)),
	}

	if strings.HasSuffix(key, ".tgz") {
		gr, err := gzip.NewReader(file)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
		tr := tar.NewReader(gr)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				a.serverInternalError(err, w)
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
