package artifacts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

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
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
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

// valid requests:
// 1. /artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeId}/outputs/{artifactName}
// 2. /artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeId}/outputs/{artifactName}/{fileName}
func (a *ArtifactServer) GetArtifactFile(w http.ResponseWriter, r *http.Request) {

	const (
		NAMESPACE_INDEX       = 2
		ARCHIVE_DISCRIM_INDEX = 3
		ID_INDEX              = 4
		NODE_ID_INDEX         = 5
		DIRECTION_INDEX       = 6
		ARTIFACT_NAME_INDEX   = 7
		FILE_NAME_INDEX       = 8
	)

	fileName := ""
	requestPath := strings.Split(r.URL.Path, "/")
	switch len(requestPath) {
	case ARTIFACT_NAME_INDEX + 1:

	case FILE_NAME_INDEX + 1:
		fileName = requestPath[FILE_NAME_INDEX]
	default:
		a.serverInternalError(fmt.Errorf("request path is not valid, expected %d or %d fields, got %d", ARTIFACT_NAME_INDEX+1, FILE_NAME_INDEX+1, len(requestPath)), w)
		return
	}

	namespace := requestPath[NAMESPACE_INDEX]
	archiveDiscriminator := requestPath[ARCHIVE_DISCRIM_INDEX]
	id := requestPath[ID_INDEX] // if archiveDiscriminator == "archived-workflows", this represents workflow UID; if archiveDiscriminator == "workflows", this represents workflow name
	nodeId := requestPath[NODE_ID_INDEX]
	direction := requestPath[DIRECTION_INDEX]
	artifactName := requestPath[ARTIFACT_NAME_INDEX]

	if direction != "outputs" {
		a.serverInternalError(fmt.Errorf("request path is not valid, expected field at index %d to be 'outputs', got %s", DIRECTION_INDEX, direction), w)
		return
	}

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	var wf *wfv1.Workflow

	// getArtifact for artifactName
	switch archiveDiscriminator {
	case "workflows":
		workflowName := id
		log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName}).Info("Get artifact file")

		wf, err = a.getWorkflowAndValidate(ctx, namespace, workflowName)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
	case "archived-workflows":
		uid := id
		log.WithFields(log.Fields{"namespace": namespace, "uid": uid, "nodeId": nodeId, "artifactName": artifactName}).Info("Get artifact file")

		wf, err = a.wfArchive.GetWorkflow(uid)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}

		// check that the namespace passed in matches this workflow's namespace
		if wf.GetNamespace() != namespace {
			a.serverInternalError(fmt.Errorf("request namespace '%s' doesn't match Workflow namespace: '%s'", namespace, wf.GetNamespace()), w)
			return
		}

		// return 401 if the client does not have permission to get wf
		err = a.validateAccess(ctx, wf)
		if err != nil {
			a.unauthorizedError(w)
			return
		}
	default:
		a.serverInternalError(fmt.Errorf("request path is not valid, expected field at index %d to be 'workflows' or 'archived-workflows', got %s",
			ARCHIVE_DISCRIM_INDEX, archiveDiscriminator), w)
		return
	}

	log.Debugf("successfully retrieved workflow %+v", wf) //todo: delete

	// todo: determine what happens when we call get kubeclient both here and in returnArtifact()
	artifact, driver, err := a.getArtifactAndDriver(ctx, nodeId, artifactName, false, wf, &fileName)
	if err != nil {
		// todo: which type of error here?
		a.serverInternalError(err, w)
		return
	}

	filesTmp, err := driver.ListObjects(artifact)
	log.Debugf("result of ListObjects for artifact %+v: filesTmp=%v, err=%v", artifact, filesTmp, err)

	isDir, err := driver.IsDirectory(artifact)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	if isDir {
		files, err := driver.ListObjects(artifact)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
		log.Debugf("this is a directory, artifact: %+v; files: %v", artifact, files)
	} else {
		log.Debugf("not a directory, artifact: %+v", artifact)
	}

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
		a.unauthorizedError(w)
		return
	}

	log.WithFields(log.Fields{"namespace": namespace, "workflowName": workflowName, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")

	wf, err := a.getWorkflowAndValidate(ctx, namespace, workflowName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput, nil)

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
		a.unauthorizedError(w)
		return
	}

	// return 401 if the client does not have permission to get wf
	err = a.validateAccess(ctx, wf)
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	log.WithFields(log.Fields{"uid": uid, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")
	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput, nil)

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

func (a *ArtifactServer) unauthorizedError(w http.ResponseWriter) {
	w.WriteHeader(401)
}

func (a *ArtifactServer) serverInternalError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(err.Error()))
	log.Errorf("Artifact Server returned internal error:%v", err)
}

func (a *ArtifactServer) getArtifactAndDriver(ctx context.Context, nodeId, artifactName string, isInput bool, wf *wfv1.Workflow, fileName *string) (*wfv1.Artifact, common.ArtifactDriver, error) {
	kubeClient := auth.GetKubeClient(ctx)

	var art *wfv1.Artifact
	if isInput {
		art = wf.Status.Nodes[nodeId].Inputs.GetArtifactByName(artifactName)
	} else {
		art = wf.Status.Nodes[nodeId].Outputs.GetArtifactByName(artifactName)
	}
	if art == nil {
		return nil, nil, fmt.Errorf("artifact not found: %s", artifactName)
	}

	ar, err := a.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return art, nil, err
	}
	l := ar.ToArtifactLocation()
	err = art.Relocate(l) // todo: want a better understanding of why we do this
	if err != nil {
		return art, nil, err
	}
	if fileName != nil {
		err = art.AppendToKey(*fileName)
		if err != nil {
			return art, nil, fmt.Errorf("error appending filename %s to key of artifact %+v: err: %v", *fileName, art, err)
		}
		log.Debugf("appended key %s to artifact %+v", *fileName, art)
	}

	driver, err := a.artDriverFactory(ctx, art, resources{kubeClient, wf.Namespace})
	if err != nil {
		return art, nil, err
	}
	log.Debugf("successfully located driver associated with artifact %+v", art)

	return art, driver, nil
}

func (a *ArtifactServer) returnArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request, wf *wfv1.Workflow, nodeId, artifactName string, isInput bool, fileName *string) error {
	/*kubeClient := auth.GetKubeClient(ctx)

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
	if fileName != nil {
		err = art.AppendToKey(*fileName)
		if err != nil {
			return fmt.Errorf("error appending filename %s to key of artifact %+v: err: %v", *fileName, art, err)
		}
	}

	driver, err := a.artDriverFactory(ctx, art, resources{kubeClient, wf.Namespace})
	if err != nil {
		return err
	}*/

	art, driver, err := a.getArtifactAndDriver(ctx, nodeId, artifactName, isInput, wf, fileName)
	if err != nil {
		return err
	}

	stream, err := driver.OpenStream(art)
	if err != nil {
		return err
	}

	defer func() {
		if err := stream.Close(); err != nil {
			log.Warningf("Error closing stream[%s]: %v", stream, err)
		}
	}()

	key, _ := art.GetKey()
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))

	_, err = io.Copy(w, stream)
	if err != nil {
		return fmt.Errorf("failed to copy stream for artifact, err:%v", err)
	}

	w.WriteHeader(http.StatusOK)

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
