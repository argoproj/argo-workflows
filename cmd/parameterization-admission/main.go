// Optional validating webhook for the experimental parameterization extension.
// It is independent of Workflow scheduling and has no Kubernetes write client.
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/server/parameterization"
)

func main() {
	var namespace, address, cert, key string
	flag.StringVar(&namespace, "namespace", "", "Only namespace admitted by this deployment")
	flag.StringVar(&address, "listen", ":8443", "HTTPS listening address")
	flag.StringVar(&cert, "tls-cert", "/tls/tls.crt", "Serving certificate")
	flag.StringVar(&key, "tls-key", "/tls/tls.key", "Serving private key")
	flag.Parse()
	if namespace == "" {
		log.Fatal("namespace is required")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("in-cluster configuration required")
	}
	cfg.Timeout = 8 * time.Second
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/validate", parameterization.AdmissionHandler{Namespace: namespace, Config: cfg, Authorization: kc.AuthorizationV1()})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	server := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	log.Fatal(server.ListenAndServeTLS(cert, key))
}
