package artifacts

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/packer"
)

type ArtifactServer struct {
	kubeClientset kubernetes.Interface
	wfClientSet   versioned.Interface
}

func NewArtifactServer(kubeClientset kubernetes.Interface, wfClientSet versioned.Interface) *ArtifactServer {
	return &ArtifactServer{kubeClientset, wfClientSet}
}

func (a *ArtifactServer) DownloadArtifact(w http.ResponseWriter, r *http.Request) {
	path := strings.SplitN(r.URL.Path, "/", 6)

	namespace := path[2]
	workflowName := path[3]
	nodeId := path[4]
	artifactName := path[5]

	data, err := a.getArtifact(namespace, workflowName, nodeId, artifactName)
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

func (a *ArtifactServer) getArtifact(namespace, workflowName, nodeId, artifactName string) ([]byte, error) {

	log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName}).Info("Download artifact")

	wf, err := a.wfClientSet.ArgoprojV1alpha1().Workflows(namespace).Get(workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	// TODO - offload node status
	art := wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	if art == nil {
		return nil, err
	}

	driver, err := artifact.NewDriver(art, resources{a.kubeClientset, namespace})
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
