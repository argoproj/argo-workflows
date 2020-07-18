package webhook

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	// sentinel error indicating that the client is not allow to use this webhook
	ErrVerificationFailed = status.Error(codes.Unauthenticated, "signature verification failed")
)

type webhookClient struct {
	// e.g "github"
	Type string `json:"type"`
	// e.g. "shh!"
	Secret string `json:"secret"`
}

// true, nil => matches
// false, nil => no matches
// false, error => bogus - please abort
type parser = func(secret string, r *http.Request) bool

// parser for each types, these should be fast, i.e. no database or API interactions
var webhookParsers = map[string]parser{
	"github": githubParse,
}

const pathPrefix = "/api/v1/events/"

// Interceptor creates an annotator that verifies webhook signatures and adds the appropriate access token to the request.
func Interceptor(client kubernetes.Interface, namespace string) func(w http.ResponseWriter, r *http.Request, next http.Handler) {
	return func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		err := addWebhookAuthorization(r, namespace, client)
		if err != nil {
			log.WithError(err).Error("Failed to parse webhook request")
			w.WriteHeader(403)
			// hide the message from the user, because it could help them attack us
			_, _ = w.Write([]byte(`{"message": "failed to parse webhook request"}`))
		} else {
			next.ServeHTTP(w, r)
		}
	}
}

func addWebhookAuthorization(r *http.Request, namespace string, client kubernetes.Interface) error {
	// try and abort before we do any work
	if r.Method != "POST" || len(r.Header["Authorization"]) > 0 || !strings.HasPrefix(r.URL.Path, pathPrefix) {
		return nil
	}
	if r.URL.Path != pathPrefix {
		namespace = strings.TrimPrefix(r.URL.Path, pathPrefix)
	}
	secrets := client.CoreV1().Secrets(namespace)
	webhookClients, err := secrets.Get("argo-workflows-webhook-clients", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get webhook clients: %w", err)
	}
	// we need to read the request body to check the signature, but we still need it for the GRPC request,
	// so read it all now, and then reinstate when we are done
	buf, _ := ioutil.ReadAll(r.Body)
	defer func() { r.Body = ioutil.NopCloser(bytes.NewBuffer(buf)) }()
	for clientName, data := range webhookClients.Data {
		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		client := &webhookClient{}
		err := yaml.Unmarshal(data, client)
		if err != nil {
			return fmt.Errorf("failed to unmarshal webhook client \"%s\": %w", clientName, err)
		}
		log.WithFields(log.Fields{"clientName": clientName, "webhookType": client.Type}).Debug("Parsing webhook request")
		ok := webhookParsers[client.Type](client.Secret, r)
		if ok {
			tokenSecret, err := secrets.Get(clientName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get token secret for \"%s\": %w", clientName, err)
			}
			r.Header["Authorization"] = []string{"Bearer " + string(tokenSecret.Data["token"])}
			return nil
		}
	}
	return nil
}
