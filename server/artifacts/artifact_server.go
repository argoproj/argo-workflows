package artifacts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/env"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type ArtifactServer struct {
	gatekeeper           auth.Gatekeeper
	hydrator             hydrator.Interface
	wfArchive            sqldb.WorkflowArchive
	instanceIDService    instanceid.Service
	artDriverFactory     artifacts.NewDriverFunc
	artifactRepositories artifactrepositories.Interface
	logger               logging.Logger
}

type Direction string

const (
	Outputs Direction = "outputs"
	Inputs  Direction = "inputs"
)

func NewArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artifactRepositories artifactrepositories.Interface, logger logging.Logger) *ArtifactServer {
	return newArtifactServer(authN, hydrator, wfArchive, instanceIDService, artifacts.NewDriver, artifactRepositories, logger)
}

func newArtifactServer(authN auth.Gatekeeper, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, instanceIDService instanceid.Service, artDriverFactory artifacts.NewDriverFunc, artifactRepositories artifactrepositories.Interface, logger logging.Logger) *ArtifactServer {
	return &ArtifactServer{authN, hydrator, wfArchive, instanceIDService, artDriverFactory, artifactRepositories, logger}
}

// nolint: contextcheck
func (a *ArtifactServer) GetOutputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, false)
}

// nolint: contextcheck
func (a *ArtifactServer) GetInputArtifact(w http.ResponseWriter, r *http.Request) {
	a.getArtifact(w, r, true)
}

// single endpoint to be able to handle serving directories as well as files, both those that have been archived and those that haven't
// Valid requests:
//
//	/artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeID}/[inputs|outputs]/{artifactName}
//	/artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeID}/[inputs|outputs]/{artifactName}/{fileName}
//	/artifact-files/{namespace}/[archived-workflows|workflows]/{id}/{nodeID}/[inputs|outputs]/{artifactName}/{fileDir}/.../{fileName}
//
// 'id' field represents 'uid' for archived workflows and 'name' for non-archived
// nolint: contextcheck
func (a *ArtifactServer) GetArtifactFile(w http.ResponseWriter, r *http.Request) {
	const (
		namespaceIndex      = 2
		archiveDiscrimIndex = 3
		idIndex             = 4
		nodeIDIndex         = 5
		directionIndex      = 6
		artifactNameIndex   = 7
		fileNameFirstIndex  = 8
	)

	var fileName *string
	requestPath := strings.Split(r.URL.Path, "/")
	if len(requestPath) >= fileNameFirstIndex+1 { // they included a file path in the URL (not just artifact name)
		joined := strings.Join(requestPath[fileNameFirstIndex:], "/")
		// sanitize file name
		cleanedPath := path.Clean(joined)
		fileName = &cleanedPath
	} else if len(requestPath) < artifactNameIndex+1 {
		a.httpBadRequestError(w)
		return
	}

	namespace := requestPath[namespaceIndex]
	archiveDiscriminator := requestPath[archiveDiscrimIndex]
	id := requestPath[idIndex] // if archiveDiscriminator == "archived-workflows", this represents workflow UID; if archiveDiscriminator == "workflows", this represents workflow name
	nodeID := requestPath[nodeIDIndex]
	direction := Direction(requestPath[directionIndex])
	artifactName := requestPath[artifactNameIndex]

	if direction != Outputs && direction != Inputs { // for now we handle output and input artifacts
		a.httpBadRequestError(w)
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
		a.logger.WithFields(logging.Fields{
			"namespace":    namespace,
			"workflowName": workflowName,
			"nodeID":       nodeID,
			"artifactName": artifactName,
		}).Info(ctx, "Get artifact file")

		wf, err = a.getWorkflowAndValidate(ctx, namespace, workflowName)
		if err != nil {
			a.serverInternalError(ctx, err, w)
			return
		}
	case "archived-workflows":
		uid := id
		a.logger.WithFields(logging.Fields{
			"namespace":    namespace,
			"uid":          uid,
			"nodeID":       nodeID,
			"artifactName": artifactName,
		}).Info(ctx, "Get artifact file")

		wf, err = a.wfArchive.GetWorkflow(ctx, uid, "", "")
		if err != nil {
			a.serverInternalError(ctx, err, w)
			return
		}

		// check that the namespace passed in matches this workflow's namespace
		if wf.GetNamespace() != namespace {
			a.httpBadRequestError(w)
			return
		}

		// return 401 if the client does not have permission to get wf
		err = a.validateAccess(ctx, wf)
		if err != nil {
			a.unauthorizedError(w)
			return
		}
	default:
		a.httpBadRequestError(w)
		return
	}

	isInput := direction == Inputs

	artifact, driver, err := a.getArtifactAndDriver(ctx, nodeID, artifactName, isInput, wf, fileName)
	if err != nil {
		a.serverInternalError(ctx, err, w)
		return
	}

	isDir := strings.HasSuffix(r.URL.Path, "/")

	if !isDir {
		isDir, err := driver.IsDirectory(ctx, artifact)
		if err != nil {
			if !argoerrors.IsCode(argoerrors.CodeNotImplemented, err) {
				a.serverInternalError(ctx, err, w)
				return
			}
		}
		if isDir {
			http.Redirect(w, r, r.URL.String()+"/", http.StatusTemporaryRedirect)
			return
		}
	}

	if isDir {
		// return an html page to the user

		objects, err := driver.ListObjects(ctx, artifact)
		if err != nil {
			a.httpFromError(ctx, err, w)
			return
		}
		a.logger.WithFields(logging.Fields{
			"artifact": artifact,
			"files":    objects,
		}).Debug(ctx, "this is a directory")

		key, _ := artifact.GetKey()
		for _, object := range objects {

			// object is prefixed by the key, we must trim it
			dir, file := path.Split(strings.TrimPrefix(object, key+"/"))

			// if dir is empty string, we are in the root dir
			// we found in index.html, abort and redirect there
			if dir == "" && file == "index.html" {
				w.Header().Set("Location", r.URL.String()+"index.html")
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
		}
		a.setSecurityHeaders(w)
		w.WriteHeader(http.StatusOK)
		output, err := a.renderDirectoryListing(objects, key)
		if err != nil {
			a.serverInternalError(ctx, err, w)
			return
		}
		_, _ = w.Write(output)
	} else { // stream the file itself
		a.logger.WithFields(logging.Fields{
			"artifact": artifact,
		}).Debug(ctx, "not a directory")

		err = a.returnArtifact(ctx, w, artifact, driver)

		if err != nil {
			a.httpFromError(ctx, err, w)
		}
	}

}

func (a *ArtifactServer) renderDirectoryListing(objects []string, key string) ([]byte, error) {
	output := bytes.NewBufferString("<html><body><ul>\n<li><a href=\"..\">..</a></li>\n")

	dirs := map[string]bool{} // to de-dupe sub-dirs

	// Use html/template to prevent XSS attacks.
	// The "./" prefix is necessary so the template engine recognizes it as a relative URL.
	// Without that, a file called "javascript:alert(1)" would be escaped to "#ZgotmplZ" by the urlFilter.
	tmpl, err := template.New("list").Parse("<li><a href=\"./{{.}}\">{{.}}</a></li>\n")
	if err != nil {
		return nil, err
	}

	for _, object := range objects {

		// object is prefixed the key, we must trim it
		dir, file := path.Split(strings.TrimPrefix(object, key+"/"))

		// if dir is empty string, we are in the root dir
		if dir == "" {
			if err = tmpl.Execute(output, file); err != nil {
				return nil, err
			}
		} else if dirs[dir] {
			continue
		} else {
			if err = tmpl.Execute(output, dir); err != nil {
				return nil, err
			}
			dirs[dir] = true
		}
	}
	_, _ = output.WriteString("</ul></body></html>")
	return output.Bytes(), nil
}

func (a *ArtifactServer) getArtifact(w http.ResponseWriter, r *http.Request, isInput bool) {
	requestPath := strings.SplitN(r.URL.Path, "/", 6)
	if len(requestPath) != 6 {
		a.httpBadRequestError(w)
		return
	}
	namespace := requestPath[2]
	workflowName := requestPath[3]
	nodeID := requestPath[4]
	artifactName := requestPath[5]

	ctx, err := a.gateKeeping(r, types.NamespaceHolder(namespace))
	if err != nil {
		a.unauthorizedError(w)
		return
	}

	a.logger.WithFields(logging.Fields{
		"namespace":    namespace,
		"workflowName": workflowName,
		"nodeID":       nodeID,
		"artifactName": artifactName,
		"isInput":      isInput,
	}).Info(ctx, "Download artifact")

	wf, err := a.getWorkflowAndValidate(ctx, namespace, workflowName)
	if err != nil {
		a.httpFromError(ctx, err, w)
		return
	}
	art, driver, err := a.getArtifactAndDriver(ctx, nodeID, artifactName, isInput, wf, nil)
	if err != nil {
		a.serverInternalError(ctx, err, w)
		return
	}

	err = a.returnArtifact(ctx, w, art, driver)

	if err != nil {
		a.httpFromError(ctx, err, w)
		return
	}
}

func (a *ArtifactServer) GetOutputArtifactByUID(w http.ResponseWriter, r *http.Request) {
	a.getArtifactByUID(w, r, false)
}

func (a *ArtifactServer) GetInputArtifactByUID(w http.ResponseWriter, r *http.Request) {
	a.getArtifactByUID(w, r, true)
}

// nolint: contextcheck
func (a *ArtifactServer) getArtifactByUID(w http.ResponseWriter, r *http.Request, isInput bool) {
	requestPath := strings.SplitN(r.URL.Path, "/", 5)
	if len(requestPath) != 5 {
		a.httpBadRequestError(w)
		return
	}
	uid := requestPath[2]
	nodeID := requestPath[3]
	artifactName := requestPath[4]

	// We need to know the namespace before we can do gate keeping
	ctx := r.Context()
	ctx = logging.WithLogger(ctx, a.logger)
	wf, err := a.wfArchive.GetWorkflow(ctx, uid, "", "")
	if err != nil {
		a.httpFromError(ctx, err, w)
		return
	}

	ctx, err = a.gateKeeping(r, types.NamespaceHolder(wf.GetNamespace()))
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
	art, driver, err := a.getArtifactAndDriver(ctx, nodeID, artifactName, isInput, wf, nil)
	if err != nil {
		a.serverInternalError(ctx, err, w)
		return
	}

	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{
		"uid":          uid,
		"nodeId":       nodeID,
		"artifactName": artifactName,
		"isInput":      isInput,
	}).Info(ctx, "Download artifact")

	err = a.returnArtifact(ctx, w, art, driver)

	if err != nil {
		a.httpFromError(ctx, err, w)
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

	// Ensure context has a logger for artifact operations
	if logging.GetLoggerFromContextOrNil(ctx) == nil {
		ctx = logging.WithLogger(ctx, a.logger)
	}

	ctx, err := a.gatekeeper.ContextWithRequest(ctx, ns)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

func (a *ArtifactServer) unauthorizedError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (a *ArtifactServer) serverInternalError(ctx context.Context, err error, w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Artifact Server returned internal error")
}

func (a *ArtifactServer) httpBadRequestError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func (a *ArtifactServer) httpFromError(ctx context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		return
	}
	statusCode := http.StatusInternalServerError
	e := &apierr.StatusError{}
	if errors.As(err, &e) { // check if it's a Kubernetes API error
		// There is a http error code somewhere in the error stack
		statusCode = int(e.Status().Code)
	} else {
		// check if it's an internal ArgoError
		argoerr, typeOkay := err.(argoerrors.ArgoError)
		if typeOkay {
			statusCode = argoerr.HTTPCode()
		}
	}

	http.Error(w, http.StatusText(statusCode), statusCode)
	if statusCode == http.StatusInternalServerError {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Artifact Server returned internal error")
	}
}

func (a *ArtifactServer) getArtifactAndDriver(ctx context.Context, nodeID, artifactName string, isInput bool, wf *wfv1.Workflow, fileName *string) (*wfv1.Artifact, common.ArtifactDriver, error) {
	logger := logging.RequireLoggerFromContext(ctx)

	kubeClient := auth.GetKubeClient(ctx)

	var art *wfv1.Artifact

	nodeStatus, err := wf.Status.Nodes.Get(nodeID)
	if err != nil {
		logger.WithError(err).WithField("nodeID", nodeID).Error(ctx, "Was unable to retrieve node")
		return nil, nil, fmt.Errorf("was not able to retrieve node")
	}
	if isInput {
		art = nodeStatus.Inputs.GetArtifactByName(artifactName)
	} else {
		art = nodeStatus.Outputs.GetArtifactByName(artifactName)
	}
	if art == nil {
		return nil, nil, fmt.Errorf("artifact not found: %s, isInput=%t, Workflow Status=%+v", artifactName, isInput, wf.Status)
	}

	// Artifact Location can be defined in various places:
	// 1. In the Artifact itself
	// 2. Defined by Controller configmap
	// 3. Workflow spec defines artifactRepositoryRef which is a ConfigMap which defines the location
	// 4. Template defines ArchiveLocation
	// 5. Inline Template

	var archiveLocation *wfv1.ArtifactLocation
	templateNode, err := wf.Status.Nodes.Get(nodeID)
	if err != nil {
		logger.WithError(err).WithField("nodeID", nodeID).Error(ctx, "was unable to retrieve node")
		return nil, nil, fmt.Errorf("unable to get artifact and driver; could not get node for %s: %v", nodeID, err)
	}
	templateName := util.GetTemplateFromNode(*templateNode)
	if templateName != "" {
		template := wf.GetTemplateByName(templateName)
		if template == nil {
			return nil, nil, fmt.Errorf("no template found for name %q associated with nodeID %q", templateName, nodeID)
		}
		archiveLocation = template.ArchiveLocation // this is case 4
	}

	if templateName == "" || !archiveLocation.HasLocation() {
		ar, err := a.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef) // this should handle cases 2, 3 and 5
		if err != nil {
			return art, nil, err
		}
		archiveLocation = ar.ToArtifactLocation()
	}

	err = art.Relocate(archiveLocation) // if the Artifact defines the location (case 1), it will be used; otherwise whatever archiveLocation is set to
	if err != nil {
		return art, nil, err
	}
	if fileName != nil {
		err = art.AppendToKey(*fileName)
		if err != nil {
			return art, nil, fmt.Errorf("error appending filename %s to key of artifact %+v: err: %v", *fileName, art, err)
		}
		logger.WithFields(logging.Fields{
			"fileName": *fileName,
			"artifact": art,
		}).Debug(ctx, "appended key to artifact")
	}

	driver, err := a.artDriverFactory(ctx, art, resources{kubeClient, wf.Namespace})
	if err != nil {
		return art, nil, err
	}
	logger.WithFields(logging.Fields{
		"artifact": art,
	}).Debug(ctx, "successfully located driver associated with artifact")

	return art, driver, nil
}

func (a *ArtifactServer) setSecurityHeaders(w http.ResponseWriter) {
	// Set strict CSP headers for defense-in-depth against XSS: https://web.dev/articles/strict-csp
	w.Header().Add("Content-Security-Policy", env.GetString("ARGO_ARTIFACT_CONTENT_SECURITY_POLICY", "sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self' 'unsafe-inline'"))
	// Mitigate clickjacking attacks
	w.Header().Add("X-Frame-Options", env.GetString("ARGO_ARTIFACT_X_FRAME_OPTIONS", "SAMEORIGIN"))
}

func (a *ArtifactServer) returnArtifact(ctx context.Context, w http.ResponseWriter, art *wfv1.Artifact, driver common.ArtifactDriver) error {
	logger := logging.RequireLoggerFromContext(ctx)
	stream, err := driver.OpenStream(ctx, art)
	if err != nil {
		return err
	}

	defer func() {
		if err := stream.Close(); err != nil {
			logger.WithError(err).WithField("stream", stream).Warn(ctx, "Error closing stream")
		}
	}()

	key, _ := art.GetKey()
	w.Header().Add("Content-Disposition", fmt.Sprintf(`filename="%s"`, path.Base(key)))
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(key)))
	a.setSecurityHeaders(w)

	_, err = io.Copy(w, stream)
	if err != nil {
		errStr := fmt.Sprintf("failed to stream artifact: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return errors.New(errStr)
	} else {
		w.WriteHeader(http.StatusOK)
	}

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
	err = a.hydrator.Hydrate(ctx, wf)
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
