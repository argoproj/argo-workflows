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
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
	requestPath := strings.Split(r.URL.Path, "/")
	var cluster, namespace, workflowName, nodeId, artifactName string
	switch len(requestPath) {
	case 6:
		cluster, namespace, workflowName, nodeId, artifactName = common.PrimaryCluster(), requestPath[2], requestPath[3], requestPath[4], requestPath[5]
	case 7:
		cluster, namespace, workflowName, nodeId, artifactName = requestPath[2], requestPath[3], requestPath[4], requestPath[5], requestPath[6]
	default:
		a.serverInternalError(errors.New("request path is not valid"), w)
		return
	}

	ctx, err := a.gateKeeping(r, &types.Msg{
		Cluster:   cluster,
		Namespace: namespace,
		Resource:  "workflows",
		Act:       "get",
	})
	if err != nil {
		a.unauthorizedError(err, w)
		return
	}

	log.WithFields(log.Fields{"cluster": cluster, "namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")

	wf, err := a.getWorkflowAndValidate(ctx, cluster, namespace, workflowName)
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
	requestPath := strings.Split(r.URL.Path, "/")

	var cluster, uid, nodeId, artifactName string
	switch len(requestPath) {
	case 5:
		cluster, uid, nodeId, artifactName = common.PrimaryCluster(), requestPath[2], requestPath[3], requestPath[4]
	case 6:
		cluster, uid, nodeId, artifactName = requestPath[2], requestPath[3], requestPath[4], requestPath[5]
	default:
		a.serverInternalError(errors.New("request path is not valid"), w)
		return
	}

	// We need to know the namespace before we can do gate keeping
	wf, err := a.wfArchive.GetWorkflow(cluster, uid)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	ctx, err := a.gateKeeping(r, &types.Msg{
		Cluster:   cluster,
		Namespace: wf.GetNamespace(),
		Act:       "get",
		Resource:  "workflows",
	})
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

	log.WithFields(log.Fields{"cluster": cluster, "uid": uid, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")
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

func (a *ArtifactServer) getWorkflowAndValidate(ctx context.Context, cluster, namespace, workflowName string) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = a.instanceIDService.Validate(wf)
	if err != nil {
		return nil, err
	}
	err = a.hydrator.Hydrate(cluster, wf)
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
