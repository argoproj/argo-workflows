package artifacts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	apierr "k8s.io/apimachinery/pkg/api/errors"

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

const (
	// EnvArgoArtifactContentSecurityPolicy is the env variable to override the default security policy -
	//   Content-Security-Policy HTTP header
	EnvArgoArtifactContentSecurityPolicy = "ARGO_ARTIFACT_CONTENT_SECURITY_POLICY"
	// 	EnvArgoArtifactXFrameOptions is the env variable to set the server X-Frame-Options header
	EnvArgoArtifactXFrameOptions = "ARGO_ARTIFACT_X_FRAME_OPTIONS"
	// DefaultContentSecurityPolicy is the default policy added to the Content-Security-Policy HTTP header
	//   if no environment override has been added
	DefaultContentSecurityPolicy = "sandbox; base-uri 'none'; default-src 'none'; img-src 'self'; style-src 'self'"
	// DefaultXFrameOptions is the default value for the X-Frame-Options header
	DefaultXFrameOptions = "SAMEORIGIN"
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

func (a *ArtifactServer) getArtifact(w http.ResponseWriter, r *http.Request, isInput bool) {
	requestPath := strings.SplitN(r.URL.Path, "/", 6)
	if len(requestPath) != 6 {
		a.httpBadRequestError("request path is not valid", w)
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
		a.httpFromError(err, "Artifact Server returned error", w)
		return
	}

	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput)

	if err != nil {
		a.httpFromError(err, "Artifact Server returned error", w)
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
		a.httpBadRequestError("request path is not valid", w)
		return
	}
	uid := requestPath[2]
	nodeId := requestPath[3]
	artifactName := requestPath[4]

	// We need to know the namespace before we can do gate keeping
	wf, err := a.wfArchive.GetWorkflow(uid)
	if err != nil {
		a.httpFromError(err, "Artifact Server returned error", w)
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
	err = a.returnArtifact(ctx, w, r, wf, nodeId, artifactName, isInput)

	if err != nil {
		a.httpFromError(err, "Artifact Server returned error", w)
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
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (a *ArtifactServer) serverInternalError(err error, w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	log.WithError(err).Error("Artifact Server returned internal error")
}

func (a *ArtifactServer) httpError(statusCode int, logText string, w http.ResponseWriter) {
	statusText := http.StatusText(statusCode)
	http.Error(w, statusText, statusCode)
	log.WithFields(log.Fields{
		"statusCode": statusCode,
		"statusText": statusText,
	}).Error(logText)
}

func (a *ArtifactServer) httpBadRequestError(logText string, w http.ResponseWriter) {
	a.httpError(http.StatusBadRequest, logText, w)
}

func (a *ArtifactServer) httpFromError(err error, logText string, w http.ResponseWriter) {
	e := &apierr.StatusError{}
	if errors.As(err, &e) {
		// There is a http error code somewhere in the error stack
		statusCode := int(e.Status().Code)
		statusText := http.StatusText(statusCode)
		http.Error(w, statusText, statusCode)

		log.WithError(err).
			WithFields(log.Fields{
				"statusCode": statusCode,
				"statusText": statusText,
			}).Error(logText)
	} else {
		// Unknown error - return internal error
		a.serverInternalError(err, w)
	}
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
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(key)))

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
