package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/plugin/pkg/client/auth/exec"
	"k8s.io/client-go/transport"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

func getClientConfig() *workflow.ClientConfig {
	var err error
	restConfig, err := Config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	if restConfig.ExecProvider != nil {
		tc, _ := restConfig.TransportConfig()
		auth, _ := exec.GetAuthenticator(restConfig.ExecProvider)
		_ = auth.UpdateTransportConfig(tc)
		rt, _ := transport.New(tc)
		req := http.Request{Header: map[string][]string{}}
		_, _ = rt.RoundTrip(&req)
		token := req.Header.Get("Authorization")
		restConfig.BearerToken = strings.TrimPrefix(token, "Bearer ")
	}
	var clientConfig workflow.ClientConfig
	_ = copier.Copy(&clientConfig, restConfig)
	return &clientConfig
}

func ContextWithAuthorization() context.Context {
	localConfig := getClientConfig()
	configByte, err := json.Marshal(localConfig)
	if err != nil {
		log.Fatal(err)
	}
	configEncoded := base64.StdEncoding.EncodeToString(configByte)
	// TODO - do we need "token"?
	md := metadata.Pairs("grpcgateway-authorization", configEncoded, "token", localConfig.BearerToken)
	return metadata.NewOutgoingContext(context.Background(), md)
}
