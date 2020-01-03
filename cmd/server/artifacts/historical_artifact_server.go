package artifacts

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/persist/sqldb"
	artifact "github.com/argoproj/argo/workflow/artifacts"
)

type HistoricalArtifactServer struct {
	authN               auth.Gatekeeper
	wfHistoryRepository sqldb.WorkflowHistoryRepository
}

func NewHistoricalArtifactServer(authN auth.Gatekeeper, wfHistoryRepository sqldb.WorkflowHistoryRepository) *HistoricalArtifactServer {
	return &HistoricalArtifactServer{authN, wfHistoryRepository}
}

func (a *HistoricalArtifactServer) ServeArtifacts(w http.ResponseWriter, r *http.Request) {

	// TODO - we should not put the token in the URL - OSWAP obvs
	authHeader := r.URL.Query().Get("Authorization")
	ctx := metadata.NewIncomingContext(r.Context(), metadata.MD{"grpcgateway-authorization": []string{authHeader}})
	ctx, err := a.authN.Context(ctx)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	path := strings.SplitN(r.URL.Path, "/", 6)

	namespace := path[2]
	uid := path[3]
	nodeId := path[4]
	artifactName := path[5]

	data, err := a.getArtifact(ctx, namespace, uid, nodeId, artifactName)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(200)
	_, err = w.Write(data)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}

func (a *HistoricalArtifactServer) getArtifact(ctx context.Context, namespace, uid, nodeId, artifactName string) ([]byte, error) {
	kubeClient := auth.GetKubeClient(ctx)

	log.WithFields(log.Fields{"namespace": namespace, "uid": uid, "nodeId": nodeId, "artifactName": artifactName}).Info("Download historical artifact")

	wf, err := a.wfHistoryRepository.GetWorkflowHistory(namespace, uid)
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "get", "workflows", namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	art := wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	if art == nil {
		return nil, err
	}

	driver, err := artifact.NewDriver(art, resources{kubeClient, namespace})
	if err != nil {
		return nil, err
	}

	tmp, err := ioutil.TempFile(".", "artifact")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	defer func() { _ = os.Remove(path) }()

	err = driver.Load(art, path)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{"size": len(file)}).Debug("Artifact file size")

	return file, nil
}
