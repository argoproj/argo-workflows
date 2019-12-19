package common

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type Server struct {
	enableClientAuth bool
	Namespace        string
	wfClientset      versioned.Interface
	kubeClientset    kubernetes.Interface
}

func NewServer(enableClientAuth bool, namespace string, wfClientset versioned.Interface, kubeClientset kubernetes.Interface) *Server {
	return &Server{enableClientAuth, namespace, wfClientset, kubeClientset}
}

func (s *Server) GetWFClient(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	if !s.enableClientAuth {
		return s.wfClientset, s.kubeClientset, nil
	}

	var restConfigStr, bearerToken string
	if len(md.Get(CLIENT_REST_CONFIG)) == 0 {
		return nil, nil, status.Error(codes.Unauthenticated, "Client kubeconfig is not found")
	}
	restConfigStr = md.Get(CLIENT_REST_CONFIG)[0]

	if len(md.Get(AUTH_TOKEN)) > 0 {
		bearerToken = md.Get(AUTH_TOKEN)[0]
	}

	restConfig := rest.Config{}

	err := json.Unmarshal([]byte(restConfigStr), &restConfig)
	if err != nil {
		return nil, nil, err
	}

	restConfig.BearerToken = bearerToken

	wfClientset, err := versioned.NewForConfig(&restConfig)
	if err != nil {
		log.Errorf("Failure to create wfClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(&restConfig)
	if err != nil {
		log.Errorf("Failure to create kubeClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}

	return wfClientset, clientset, nil
}
