package artifacts

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	artifact "github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
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

func (a *ArtifactServer) GetOutputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, false)
}

func (a *ArtifactServer) GetInputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, true)
}

func (a *ArtifactServer) getArtifact(w http.ResponseWriter, r *http.Request, isInput bool) {
	requestPath := strings.SplitN(r.URL.Path, "/", 6)
	if len(requestPath) != 6 {
		a.serverInternalError(errors.New("request path is not valid"), w)
		return
	}
	namespace := requestPath[2]
	workflowName := requestPath[3]
	nodeId := requestPath[4]
	artifactName := requestPath[5]

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(err, w)
		return
	}

	log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")

	wf, err := a.getWorkflowAndValidate(ctx, namespace, workflowName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput)

	if err != nil {
		a.serverInternalError(err, w)
		return
	}
}

func (a *ArtifactServer) GetOutputArtifactByUID(w http.ResponseWriter, r *http.Request) {
	a.getArtifactByUID(w, r, false)
}

func (a *ArtifactServer) GetInputArtifactByUID(w http.ResponseWriter, r *http.Request) {
	a.getArtifactByUID(w, r, true)
}

func (a *ArtifactServer) getArtifactByUID(w http.ResponseWriter, r *http.Request, isInput bool) {
	requestPath := strings.SplitN(r.URL.Path, "/", 5)
	if len(requestPath) != 5 {
		a.serverInternalError(errors.New("request path is not valid"), w)
		return
	}
	uid := requestPath[2]
	nodeId := requestPath[3]
	artifactName := requestPath[4]

	// We need to know the namespace before we can do gate keeping
	wf, err := a.wfArchive.GetWorkflow(uid)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(wf.GetNamespace()))
	if err != nil {
		a.unauthorizedError(err, w)
		return
	}

	// return 401 if the client does not have permission to get wf
	err = a.validateAccess(ctx, wf)
	if err != nil {
		a.unauthorizedError(err, w)
		return
	}

	log.WithFields(log.Fields{"uid": uid, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")
	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput)

	if err != nil {
		a.serverInternalError(err, w)
		return
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

func (a *ArtifactServer) unauthorizedError(err error, w http.ResponseWriter) {
	w.WriteHeader(401)
	_, _ = w.Write([]byte(err.Error()))
}

func (a *ArtifactServer) serverInternalError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(err.Error()))
}

func (a *ArtifactServer) returnArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request, wf *wfv1.Workflow, nodeId, artifactName string, isInput bool) error {
	kubeClient := auth.GetKubeClient(ctx)

	var art *wfv1.Artifact
	if isInput {
		art = wf.Status.Nodes[nodeId].Inputs.GetArtifactByName(artifactName)
	} else {
		art = wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	}
	if art == nil {
		return fmt.Errorf("artifact not found")
	}

	ar, err := a.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return err
	}
	l := ar.ToArtifactLocation()
	err = art.Relocate(l)
	if err != nil {
		return err
	}

	driver, err := a.artDriverFactory(ctx, art, resources{kubeClient, wf.Namespace})
	if err != nil {
		return err
	}
	tmp, err := ioutil.TempFile("/tmp", "artifact")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	err = driver.Load(art, tmpPath)
	if err != nil {
		return err
	}

	file, err := os.Open(filepath.Clean(tmpPath))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Error closing file[%s]: %v", tmpPath, err)
		}
	}()

	stats, err := file.Stat()
	if err != nil {
		return err
	}

	contentLength := strconv.FormatInt(stats.Size(), 10)
	log.WithFields(log.Fields{"size": contentLength}).Debug("Artifact file size")

	key, _ := art.GetKey()
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))
	w.WriteHeader(200)

	http.ServeContent(w, r, "", time.Time{}, file)

	return nil
}

func (a *ArtifactServer) getWorkflowAndValidate(ctx context.Context, namespace string, workflowName string) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = a.instanceIDService.Validate(wf)
	if err != nil {
		return nil, err
	}
	err = a.hydrator.Hydrate(wf)
	if err != nil {
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
		return status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil
}
