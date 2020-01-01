package artifacts

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/cmd/server/workflow"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/packer"
)

type ArtifactServer struct {
	authN       auth.Gatekeeper
	wfDBService *workflow.DBService
}

func NewArtifactServer(authN auth.Gatekeeper, wfDBService *workflow.DBService) *ArtifactServer {
	return &ArtifactServer{authN, wfDBService}
}

func (a *ArtifactServer) ServeArtifacts(w http.ResponseWriter, r *http.Request) {

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
	workflowName := path[3]
	nodeId := path[4]
	artifactName := path[5]

	data, err := a.getArtifact(ctx, namespace, workflowName, nodeId, artifactName)
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

func (a *ArtifactServer) getArtifact(ctx context.Context, namespace, workflowName, nodeId, artifactName string) ([]byte, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName}).Info("Download artifact")

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	if wf.Status.OffloadNodeStatus {
		offloadedWf, err := a.wfDBService.Get(workflowName, namespace)
		if err != nil {
			return nil, err
		}
		wf.Status.Nodes = offloadedWf.Status.Nodes
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
