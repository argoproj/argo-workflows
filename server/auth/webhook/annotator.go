package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func Annotator(client typedcorev1.SecretInterface) (func(ctx context.Context, r *http.Request) metadata.MD, error) {

	peekBody := func(r *http.Request) []byte {
		buf, _ := ioutil.ReadAll(r.Body)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		return buf
	}

	// matchers for each types
	webhookTypes := map[string]func(secret string, r *http.Request) error{
		"github": func(secret string, r *http.Request) error {
			if len(r.Header["X-Github-Event"]) != 1 && len(r.Header["X-Hub-Signature"]) != 1 {
				return NotMatched
			}
			mac := hmac.New(sha1.New, []byte(secret))
			_, _ = mac.Write(peekBody(r))
			expectedMAC := hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(r.Header["X-Hub-Signature"][0][5:]), []byte(expectedMAC)) {
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

	return func(ctx context.Context, r *http.Request) metadata.MD {
		for serviceAccountName, account := range accounts {
			err := webhookTypes[account.webhookType](account.secret, r)
			switch err {
			case NotMatched:
			case nil:
				secret, err := client.Get(serviceAccountName, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return metadata.Pairs("authorization", "Bearer "+string(secret.Data["token"]))
			default:
				// matched, but error
				return nil
			}
		}
		return nil
	}, nil
}
