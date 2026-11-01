// The optional experimental controller writes definition validation status only.
// It does not watch Workflows or own their scheduling, status or execution.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/server/parameterization"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var namespace string
	var clusterTemplates bool
	flag.StringVar(&namespace, "namespace", "", "Single namespace for definition status reconciliation")
	flag.BoolVar(&clusterTemplates, "cluster-templates", false, "Observe native ClusterWorkflowTemplate references with separately granted read permissions")
	flag.Parse()
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster configuration required: %w", err)
	}
	cfg.Timeout = 15 * time.Second
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	controller, err := parameterization.NewDefinitionController(client, namespace, clusterTemplates)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !controller.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	health := &http.Server{Addr: ":6061", Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() {
		if serveErr := health.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Print(serveErr)
			cancel()
		}
	}()
	defer health.Close()
	if err = controller.Run(ctx, func(err error) { log.Print(err) }); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
