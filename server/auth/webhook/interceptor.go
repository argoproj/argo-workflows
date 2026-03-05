package webhook

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/secrets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type webhookClient struct {
	// e.g "github"
	Type string `json:"type"`
	// e.g. "shh!"
	Secret string `json:"secret"`
	// XHubHeaderName specifies the header name for x-hub type (default: "X-Hub-Signature-256")
	XHubHeaderName string `json:"x-hub-header-name,omitempty"`
	// XHubHashAlgorithm specifies the hash algorithm for x-hub type (default: "sha256")
	XHubHashAlgorithm string `json:"x-hub-hash,omitempty"`
	// XHubEncoding specifies the signature encoding for x-hub type (default: "hex")
	XHubEncoding string `json:"x-hub-encoding,omitempty"`
}

type matcher = func(secret string, r *http.Request) bool

// parser for each types, these should be fast, i.e. no database or API interactions
var webhookParsers = map[string]matcher{
	"bitbucket":       bitbucketMatch,
	"bitbucketserver": bitbucketserverMatch,
	"github":          githubMatch,
	"gitlab":          gitlabMatch,
}

const pathPrefix = "/api/v1/events/"

type Interceptor struct {
	logger logging.Logger
}

func NewInterceptor(logger logging.Logger) *Interceptor {
	return &Interceptor{logger: logger}
}

// Interceptor creates an annotator that verifies webhook signatures and adds the appropriate access token to the request.
func (i *Interceptor) Interceptor(client kubernetes.Interface) func(w http.ResponseWriter, r *http.Request, next http.Handler) {
	return func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		err := i.addWebhookAuthorization(r, client)
		if err != nil {
			i.logger.WithError(err).Error(r.Context(), "Failed to process webhook request")
			w.WriteHeader(http.StatusForbidden)
			// hide the message from the user, because it could help them attack us
			_, _ = w.Write([]byte(`{"message": "failed to process webhook request"}`))
		} else {
			next.ServeHTTP(w, r)
		}
	}
}

func (i *Interceptor) addWebhookAuthorization(r *http.Request, kube kubernetes.Interface) error {
	// try and exit quickly before we do anything API calls
	if r.Method != http.MethodPost || len(r.Header["Authorization"]) > 0 || !strings.HasPrefix(r.URL.Path, pathPrefix) {
		return nil
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, pathPrefix), "/", 2)
	if len(parts) != 2 {
		return nil
	}
	namespace := parts[0]
	secretsInterface := kube.CoreV1().Secrets(namespace)
	ctx := r.Context()

	webhookClients, err := secretsInterface.Get(ctx, "argo-workflows-webhook-clients", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get webhook clients: %w", err)
	}
	// we need to read the request body to check the signature, but we still need it for the GRPC request,
	// so read it all now, and then reinstate when we are done
	buf, _ := io.ReadAll(r.Body)
	defer func() { r.Body = io.NopCloser(bytes.NewBuffer(buf)) }()
	serviceAccountInterface := kube.CoreV1().ServiceAccounts(namespace)
	for serviceAccountName, data := range webhookClients.Data {
		r.Body = io.NopCloser(bytes.NewBuffer(buf))
		client := &webhookClient{}
		err := yaml.Unmarshal(data, client)
		if err != nil {
			return fmt.Errorf("failed to unmarshal webhook client \"%s\": %w", serviceAccountName, err)
		}
		i.logger.WithFields(logging.Fields{"serviceAccountName": serviceAccountName, "webhookType": client.Type}).Debug(r.Context(), "Attempting to match webhook request")
		var ok bool
		if client.Type == "x-hub" {
			config := &XHubConfig{
				HashAlgorithm: client.XHubHashAlgorithm,
				HeaderName:    client.XHubHeaderName,
				Encoding:      client.XHubEncoding,
			}
			ok = xHubMatch(client.Secret, r, config)
		} else {
			ok = webhookParsers[client.Type](client.Secret, r)
		}
		if ok {
			i.logger.WithField("serviceAccountName", serviceAccountName).Debug(r.Context(), "Matched webhook request")
			serviceAccount, err := serviceAccountInterface.Get(ctx, serviceAccountName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get service account \"%s\": %w", serviceAccountName, err)
			}
			tokenSecret, err := secretsInterface.Get(ctx, secrets.TokenNameForServiceAccount(serviceAccount), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get token secret \"%s\": %w", tokenSecret, err)
			}
			r.Header["Authorization"] = []string{"Bearer " + string(tokenSecret.Data["token"])}
			return nil
		}
	}
	return nil
}
