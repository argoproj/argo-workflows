package common

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

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
	if !s.enableClientAuth {
		return s.wfClientset, s.kubeClientset, nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("unable to get metadata from incoming context")
	}
	authorization := md.Get("grpcgateway-authorization")
	if len(authorization) == 0 {
		return nil, nil, status.Error(codes.Unauthenticated, "Authorization header not found")
	}
	// Format is `Bearer base64(~/.kube/config)'
	restConfigBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorization[0], "Bearer "))
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "Invalid token found in Authorization header: %v", err)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(restConfigBytes)
	if err != nil {
		return nil, nil, err
	}
	wfClientset, err := versioned.NewForConfig(restConfig)
	if err != nil {
		log.Errorf("Failure to create wfClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Errorf("Failure to create kubeClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}
	return wfClientset, clientset, nil
}
