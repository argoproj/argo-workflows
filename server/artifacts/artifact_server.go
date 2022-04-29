package artifacts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

const (
	// EnvArgoArtifactContentSecurityPolicy is the env variable to override the default security policy -
	//   Content-Security-Policy HTTP header
	EnvArgoArtifactContentSecurityPolicy = "ARGO_ARTIFACT_CONTENT_SECURITY_POLICY"
	// 	EnvArgoArtifactXFrameOptions is the env variable to set the server X-Frame-Options header
	EnvArgoArtifactXFrameOptions = "ARGO_ARTIFACT_X_FRAME_OPTIONS"
	// DefaultContentSecurityPolicy is the default policy added to the Content-Security-Policy HTTP header
	//   if no environment override has been added
	DefaultContentSecurityPolicy = "sandbox; base-uri 'none'; default-src 'none'; image-src: 'self'; style-src: 'self'"
	// DefaultXFrameOptions is the default value for the X-Frame-Options header
	DefaultXFrameOptions = "SAMESITE"
)

type ArtifactServer struct {
	gatekeeper           auth.Gatekeeper
	hydrator             hydrator.Interface
	wfArchive            sqldb.WorkflowArchive
	instanceIDService    instanceid.Service
	artDriverFactory     artifact.NewDriverFunc
	artifactRepositories artifactrepositories.Interface
	httpHeaderConfig     map[string]string
}

func NewArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artifactRepositories artifactrepositories.Interface) *ArtifactServer {
	return newArtifactServer(authN, hydrator, wfArchive, instanceIDService, artifact.NewDriver, artifactRepositories)
}

func newArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artDriverFactory artifact.NewDriverFunc, artifactRepositories artifactrepositories.Interface) *ArtifactServer {
	httpHeaderConfig := map[string]string{}

	env, defined := os.LookupEnv(EnvArgoArtifactContentSecurityPolicy)
	if defined {
		httpHeaderConfig["Content-Security-Policy"] = env
	} else {
		httpHeaderConfig["Content-Security-Policy"] = DefaultContentSecurityPolicy
	}

	env, defined = os.LookupEnv(EnvArgoArtifactXFrameOptions)
	if defined {
		httpHeaderConfig["X-Frame-Options"] = env
	} else {
		httpHeaderConfig["X-Frame-Options"] = DefaultXFrameOptions
	}

	return &ArtifactServer{authN, hydrator, wfArchive, instanceIDService, artDriverFactory, artifactRepositories, httpHeaderConfig}
}

func (a *ArtifactServer) GetOutputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, false)
}

func (a *ArtifactServer) GetInputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, true)
}

// single endpoint to be able to handle serving directories as well as files, both those that have been archived adn those that haven't
// Valid requests:
//  /artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeId}/outputs/{artifactName}
//  /artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeId}/outputs/{artifactName}/{fileName}
//  /artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeId}/outputs/{artifactName}/{fileDir}/.../{fileName}
// 'id' field represents 'uid' for archived workflows and 'name' for non-archived
func (a *ArtifactServer) GetArtifactFile(w http.ResponseWriter, r *http.Request) {

	const (
		NAMESPACE_INDEX       = 2
		ARCHIVE_DISCRIM_INDEX = 3
		ID_INDEX              = 4
		NODE_ID_INDEX         = 5
		DIRECTION_INDEX       = 6
		ARTIFACT_NAME_INDEX   = 7
		FILE_NAME_FIRST_INDEX = 8
	)

	var fileName *string
	requestPath := strings.Split(r.URL.Path, "/")
	if len(requestPath) >= FILE_NAME_FIRST_INDEX+1 { // they included a file path in the URL (not just artifact name)
		joined := strings.Join(requestPath[FILE_NAME_FIRST_INDEX:], "/")
		// sanitize file name
		cleanedPath := filepath.Clean(joined)
		fileName = &cleanedPath
	} else if len(requestPath) < ARTIFACT_NAME_INDEX+1 {
		a.serverInternalError(fmt.Errorf("request path is not valid, expected at least %d fields, got %d", ARTIFACT_NAME_INDEX+1, len(requestPath)), w)
		return
	}

	namespace := requestPath[NAMESPACE_INDEX]
	archiveDiscriminator := requestPath[ARCHIVE_DISCRIM_INDEX]
	id := requestPath[ID_INDEX] // if archiveDiscriminator == "archived-workflows", this represents workflow UID; if archiveDiscriminator == "workflows", this represents workflow name
	nodeId := requestPath[NODE_ID_INDEX]
	direction := requestPath[DIRECTION_INDEX]
	artifactName := requestPath[ARTIFACT_NAME_INDEX]

	if direction != "outputs" { // for now we just handle output artifacts
		a.serverInternalError(fmt.Errorf("request path is not valid, expected field at index %d to be 'outputs', got %s", DIRECTION_INDEX, direction), w)
		return
	}

	// verify user is authorized
	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	var wf *wfv1.Workflow

	// retrieve the Workflow
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

	artifact, driver, err := a.getArtifactAndDriver(ctx, nodeId, artifactName, false, wf, fileName)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	isDir, err := driver.IsDirectory(artifact)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	if isDir {
		// return an html page to the user

		files, err := driver.ListObjects(artifact)
		if err != nil {
			a.serverInternalError(err, w)
			return
		}
		log.Debugf("this is a directory, artifact: %+v; files: %v", artifact, files)

		// set headers
		for name, value := range a.httpHeaderConfig {
			w.Header().Add(name, value)
		}

		w.Write([]byte("<html><body><ul>\n"))

		for _, file := range files {

			pathSlice := strings.Split(file, "/")

			// verify the files are formatted as expected
			if len(pathSlice) < 2 {
				a.serverInternalError(fmt.Errorf("something went wrong: the files returned should each start with directory followed by file name; files:%+v", files), w)
			}

			// the artifactname should be the first level directory of our file - verify
			if pathSlice[0] != artifactName {
				a.serverInternalError(fmt.Errorf("something went wrong: the files returned should start with artifact name %s but don't; files:%+v", artifactName, files), w)
			}

			fullyQualifiedPath := fmt.Sprintf("%s/%s", strings.Join(requestPath[:ARTIFACT_NAME_INDEX], "/"), file)

			// add a link to the html page, which will be a relative filepath
			removeDirLen := len(requestPath) - ARTIFACT_NAME_INDEX - 1
			link := strings.Join(pathSlice[removeDirLen:], "/")

			w.Write([]byte(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", link, fullyQualifiedPath)))
		}

		w.Write([]byte("</ul></body></html>"))

	} else { // stream the file itself
		log.Debugf("not a directory, artifact: %+v", artifact)
		a.returnArtifact(ctx, w, r, artifact, driver, fileName)
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
	art, driver, err := a.getArtifactAndDriver(ctx, nodeId, artifactName, isInput, wf, nil)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	err = a.returnArtifact(ctx, w, r, art, driver, nil)

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
	art, driver, err := a.getArtifactAndDriver(ctx, nodeId, artifactName, isInput, wf, nil)
	if err != nil {
		a.serverInternalError(err, w)
		return
	}

	log.WithFields(log.Fields{"uid": uid, "nodeId": nodeId, "artifactName": artifactName, "isInput": isInput}).Info("Download artifact")
	err = a.returnArtifact(ctx, w, r, art, driver, nil)

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
	log.WithError(err).Error("Artifact Server returned internal error")
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
	err = art.Relocate(l)
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

func (a *ArtifactServer) returnArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request, art *wfv1.Artifact, driver common.ArtifactDriver, fileName *string) error {
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

	// Iterate and set the rest of the headers
	for name, value := range a.httpHeaderConfig {
		w.Header().Add(name, value)
	}

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
