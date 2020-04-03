package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/packer"
)

type ArtifactServer struct {
	authN                 auth.Gatekeeper
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	wfArchive             sqldb.WorkflowArchive
}

func NewArtifactServer(authN auth.Gatekeeper, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchive sqldb.WorkflowArchive) *ArtifactServer {
	return &ArtifactServer{authN, offloadNodeStatusRepo, wfArchive}
}

func (a *ArtifactServer) GetArtifact(w http.ResponseWriter, r *http.Request) {

	ctx, err := a.gateKeeping(r)
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

	log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName}).Info("Download artifact")

	wf, err := a.getWorkflow(ctx, namespace, workflowName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	data, err := a.getArtifact(ctx, wf, nodeId, artifactName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s.tgz"`, artifactName))
	a.ok(w, data)
}

// Information to enable downloading logs for a given workflow.
type LogDownloadInfo struct {
	WorkflowName      string `json:"name"`
	WorkflowNamespace string `json:"namespace"`
}

func (a *ArtifactServer) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx, err := a.gateKeeping(r)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	tmpDir, err := ioutil.TempDir(".", "main-logs")
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	var workflowNames []LogDownloadInfo
	err = json.Unmarshal([]byte(r.PostFormValue("workflows")), &workflowNames)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	for _, workflowInfo := range workflowNames {
		wf, err := a.getWorkflow(ctx, workflowInfo.WorkflowNamespace, workflowInfo.WorkflowName)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}

		err = os.Mkdir(filepath.Join(tmpDir, workflowInfo.WorkflowName), 0744)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}

		err = a.getLogArtifacts(ctx, wf, filepath.Join(tmpDir, workflowInfo.WorkflowName))
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
	}

	data, err := dirToTarGz(tmpDir, "")
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	w.Header().Add("Content-Disposition", `filename="workflow-main-logs.tgz"`)
	a.ok(w, data)
}

// Get node IDs for nodes with main-logs artifacts.
func GetLogNodeIds(w *wfv1.Workflow) []string {
	visited := make(map[string]struct{})
	var getLoggedNodeIdsRecursive func(nodeName string) []string
	getLoggedNodeIdsRecursive = func (nodeName string) []string {
		_, wasVisited := visited[nodeName]
		if wasVisited {
			return make([]string, 0)
		}
		visited[nodeName] = struct{}{}
		node := w.Status.Nodes[nodeName]
		var hasLogs bool
		nodeOutputs := node.Outputs
		if nodeOutputs != nil {
			items := nodeOutputs.Artifacts
			if items != nil {
				for _, item := range items {
					if item.Name == "main-logs" {
						hasLogs = true
						break
					}
				}
			}
		}
		var childItems []string
		for _, childNodeName := range node.Children {
			childItems = append(childItems, getLoggedNodeIdsRecursive(childNodeName)...)
		}
		if hasLogs {
			return append([]string{node.ID}, childItems...)
		}
		return childItems
	}
	var nodes []string
	for _, node := range w.Status.Nodes {
		nodes = append(nodes, getLoggedNodeIdsRecursive(node.Name)...)
	}
	return nodes
}

func (a *ArtifactServer) GetArtifactByUID(w http.ResponseWriter, r *http.Request) {
	ctx, err := a.gateKeeping(r)
	if err != nil {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	path := strings.SplitN(r.URL.Path, "/", 6)

	uid := path[2]
	nodeId := path[3]
	artifactName := path[4]

	log.WithFields(log.Fields{"uid": uid, "nodeId": nodeId, "artifactName": artifactName}).Info("Download artifact")

	wf, err := a.getWorkflowByUID(ctx, uid)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	data, err := a.getArtifact(ctx, wf, nodeId, artifactName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s.tgz"`, artifactName))
	a.ok(w, data)
}

func (a *ArtifactServer) gateKeeping(r *http.Request) (context.Context, error) {
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
	return a.authN.Context(ctx)
}

func (a *ArtifactServer) ok(w http.ResponseWriter, data []byte) {
	w.WriteHeader(200)
	_, err := w.Write(data)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}

func (a *ArtifactServer) serverInternalError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(err.Error()))
}

func (a *ArtifactServer) getArtifact(ctx context.Context, wf *wfv1.Workflow, nodeId, artifactName string) ([]byte, error) {
	kubeClient := auth.GetKubeClient(ctx)

	art := wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	if art == nil {
		return nil, fmt.Errorf("artifact not found")
	}

	driver, err := artifact.NewDriver(art, resources{kubeClient, wf.Namespace})
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

// Get log artifacts from all logged nodes in the given workflow. Write them to the specified
// directory.
func(a *ArtifactServer) getLogArtifacts(ctx context.Context, wf *wfv1.Workflow, destDir string) error {
	if destDir == "" {
		destDir = "."
	}

	nodeIds := GetLogNodeIds(wf)

	for _, nodeId := range nodeIds {
		art, err := a.getArtifact(ctx, wf, nodeId, "main-logs")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(destDir, nodeId), art, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// dirToTarGz converts a directory to a tar.gz file. Files from the source directory are placed in
// the tar.gz's root. Returns a byte array of the tar.gz file.
func dirToTarGz(sourceDir string, destDir string) ([]byte, error) {
	tmpFile, err := ioutil.TempFile(".", "dir-to-tar")
	if err != nil {
		return nil, err
	}

	gzw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gzw)

	err = filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		err2 := addFileToTgz(fi, file, sourceDir, destDir, tw)
		if err2 != nil {
			return err2
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = gzw.Close()
	if err != nil {
		return nil, err
	}

	err = tw.Close()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	err = os.Remove(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	return data, err
}

func addFileToTgz(fi os.FileInfo, file string, sourceDir string, destDir string, tw *tar.Writer) error {
	if !fi.Mode().IsRegular() {
		return nil
	}

	header, err := tar.FileInfoHeader(fi, fi.Name())
	if err != nil {
		return err
	}

	// Remove the source dir from the path. All files go in root.
	header.Name = strings.Trim(strings.Replace(file, sourceDir, destDir, -1), string(filepath.Separator))

	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *ArtifactServer) getWorkflow(ctx context.Context, namespace string, workflowName string) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	if wf.Status.IsOffloadNodeStatus() {
		if a.offloadNodeStatusRepo.IsEnabled() {
			offloadedNodes, err := a.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
			if err != nil {
				return nil, err
			}
			wf.Status.Nodes = offloadedNodes
		} else {
			log.WithFields(log.Fields{"namespace": namespace, "name": workflowName}).Warn(sqldb.OffloadNodeStatusDisabledWarning)
		}
	}
	return wf, nil
}

func (a *ArtifactServer) getWorkflowByUID(ctx context.Context, uid string) (*wfv1.Workflow, error) {
	wf, err := a.wfArchive.GetWorkflow(uid)
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "get", "workflows", wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, nil
}
