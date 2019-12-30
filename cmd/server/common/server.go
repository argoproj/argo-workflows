package common

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"

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
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if !s.enableClientAuth {
			return s.wfClientset, s.kubeClientset, nil
		}
		return nil, nil, fmt.Errorf("unable to get metadata from incoming context")
	}
	authorization := md.Get("grpcgateway-authorization")
	if len(authorization) == 0 {
		if !s.enableClientAuth {
			return s.wfClientset, s.kubeClientset, nil
		}
		return nil, nil, status.Error(codes.Unauthenticated, "Authorization header not found")
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	restConfigBytes, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "Invalid token found in Authorization header %s: %v", token, err)
	}

	var restConfig rest.Config
	err = json.Unmarshal(restConfigBytes, &restConfig)
	if err != nil {
		return nil, nil, err
	}

	if s.enableClientAuth {
		// we want to prevent people using in-cluster set-up
		if restConfig.BearerTokenFile != "" || restConfig.CAFile != "" || restConfig.CertFile != "" || restConfig.KeyFile != "" {
			return nil, nil, status.Errorf(codes.Unauthenticated, "illegal bearer token")
		}
		host := strings.SplitN(restConfig.Host, ":", 2)[0]
		if host == "localhost" || net.ParseIP(host).IsLoopback() {
			return nil, nil, status.Errorf(codes.Unauthenticated, "illegal bearer token")
		}
	}

	wfClientset, err := versioned.NewForConfig(&restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create wfClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	clientset, err := kubernetes.NewForConfig(&restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create kubeClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	return wfClientset, clientset, nil
}
