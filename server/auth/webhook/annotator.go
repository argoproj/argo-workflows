package webhook

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"
)

var (
	NotMatched         = errors.New("not matched")
	VerificationFailed = status.Error(codes.Unauthenticated, "signature verification failed")
)

type account struct {
	webhookType, secret string
}

// Annotator creates an annotator that verifies webhook signatures and adds the appropriate access token to the request.
func Annotator(client typedcorev1.SecretInterface) (func(ctx context.Context, r *http.Request) metadata.MD, error) {

	// matchers for each types, these should be fast, i.e. no database or API interactions
	webhookTypes := map[string]func(secret string, r *http.Request) error{
		"github": github,
	}

	// we must store our config in a specific secret as we do not have `list secrets` RBAC, only `get secrets`
	list, err := client.Get("webhook-accounts", metav1.GetOptions{})
	if err != nil {
		// if the secret is not found, then this is disabled.
		if apierr.IsNotFound(err) {
			log.WithError(err).Info("webhook annotation disabled (i.e. github etc webhooks are not supported)")
			return func(ctx context.Context, r *http.Request) metadata.MD { return nil }, nil
		}
		return nil, err
	}
	accounts := make(map[string]account)
	for serviceAccountName, data := range list.Data {
		datum := map[string]string{}
		err := yaml.Unmarshal(data, &datum)
		if err != nil {
			return nil, err
		}
		accounts[serviceAccountName] = account{datum["type"], datum["secret"]}
	}

	log.WithField("accountCount", len(accounts)).Info("Webhook accounts loaded")

	return func(ctx context.Context, r *http.Request) metadata.MD {
		for serviceAccountName, account := range accounts {
			err := webhookTypes[account.webhookType](account.secret, r)
			switch err {
			case NotMatched:
				// no nothing
			case nil:
				secret, err := client.Get(serviceAccountName, metav1.GetOptions{})
				if err != nil {
					// it is not possible te error-out here
					log.WithError(err).WithField("account", serviceAccountName).Error("Failed to get access token, ignoring")
					return nil
				}
				return metadata.Pairs("Authorization", "Bearer "+string(secret.Data["token"]))
			default:
				// matched, but error
				return nil
			}
		}
		return nil
	}, nil
}
