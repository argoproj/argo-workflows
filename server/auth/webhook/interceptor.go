package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
)

var (
	NotMatched         = errors.New("not matched")
	VerificationFailed = status.Error(codes.Unauthenticated, "signature verification failed")
)

type account struct {
	webhookType, secret string
}

func UnaryServerInterceptor(client typedcorev1.SecretInterface) (grpc.UnaryServerInterceptor, error) {

	// matchers for each types
	webhookTypes := map[string]func(md metadata.MD, secret string, payload []byte) error{
		"github": func(md metadata.MD, secret string, payload []byte) error {
			if len(md["x-github-event"]) != 1 && len(md["x-hub-signature"]) != 1 {
				return NotMatched
			}
			mac := hmac.New(sha1.New, []byte(secret))
			_, _ = mac.Write(payload)
			expectedMAC := hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(md["x-hub-signature"][0][5:]), []byte(expectedMAC)) {
				return VerificationFailed
			}
			return nil
		},
	}

	list, err := client.Get("webhooks", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	accounts := make(map[string]account)
	for serviceAccountName, data := range list.Data {
		a := map[string]string{}
		err := yaml.Unmarshal(data, &a)
		if err != nil {
			return nil, err
		}
		accounts[serviceAccountName] = account{a["type"], a["secret"]}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, _ := metadata.FromIncomingContext(ctx)
		eventRequest, ok := req.(*eventpkg.EventRequest)
		if ok && len(md["authorization"]) == 0 && info.FullMethod == "/event.EventService/ReceiveEvent" {
			payload := eventRequest.Event.Value
			for serviceAccountName, account := range accounts {
				err := webhookTypes[account.webhookType](md, account.secret, payload)
				switch err {
				case NotMatched:
				case nil:
					secret, err := client.Get(serviceAccountName, metav1.GetOptions{})
					if err != nil {
						return nil, err
					}
					ctx = metadata.NewIncomingContext(ctx, metadata.Join(md, metadata.Pairs("authorization", "Bearer "+string(secret.Data["token"]))))
				default:
					return nil, err
				}
			}
		}
		return handler(ctx, req)
	}, nil
}
