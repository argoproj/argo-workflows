package webhook

import (
	"bytes"
	"errors"
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
	// sentinal error indicating that the request did not match, and can be safely ignored
	NotMatched = errors.New("request not matched")
	// sentinal error indicating that the request, while not necessarily bogus, is not allow to use this service webhookAccount
	VerificationFailed = status.Error(codes.Unauthenticated, "signature verification failed")
)

type webhookAccount struct {
	Type   string `json:"type"`
	Secret string `json:"secret"`
}

type parser = func(secret string, r *http.Request) error

// parser for each types, these should be fast, i.e. no database or API interactions
var webhookParsers = map[string]parser{
	"github": githubParse,
}

const pathPrefix = "/api/v1/events/"

// MiddlewareInterceptor creates an annotator that verifies webhook signatures and adds the appropriate access token to the request.
func MiddlewareInterceptor(client kubernetes.Interface, namespace string) func(w http.ResponseWriter, r *http.Request, next http.Handler) {
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
	if r.Method != "POST" || len(r.Header["Authorization"]) > 0 || !strings.HasPrefix(r.URL.Path, pathPrefix) {
		return nil
	}
	if r.URL.Path != pathPrefix {
		namespace = strings.TrimPrefix(r.URL.Path, pathPrefix)
	}
	secrets := client.CoreV1().Secrets(namespace)
	webhookAccounts, err := secrets.Get("argo-workflows-webhook-accounts", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secrets: %w", err)
	}
	buf, _ := ioutil.ReadAll(r.Body)
	log.Debugln(string(buf))
	defer func() { r.Body = ioutil.NopCloser(bytes.NewBuffer(buf)) }()
	for serviceAccountName, data := range webhookAccounts.Data {
		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		a := &webhookAccount{}
		err := yaml.Unmarshal(data, a)
		if err != nil {
			return fmt.Errorf("failed to unmarshal webhook account \"%s\": %w", serviceAccountName, err)
		}
		log.WithFields(log.Fields{"serviceAccountName": serviceAccountName, "webhookType": a.Type}).Debug("Parsing webhook request")
		switch webhookParsers[a.Type](a.Secret, r) {
		case NotMatched:
			// no nothing
		case nil:
			tokenSecret, err := secrets.Get(serviceAccountName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get token secret for \"%s\": %w", serviceAccountName, err)
			}
			r.Header["Authorization"] = []string{"Bearer " + string(tokenSecret.Data["token"])}
			return nil
		default:
			// matched, but error
			return fmt.Errorf("error parsing payload \"%s\": %w", serviceAccountName, err)
		}
	}
	return nil
}
