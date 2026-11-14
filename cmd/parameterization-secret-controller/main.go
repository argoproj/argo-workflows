// Optional administrator-configured Vault materialization. Native Secret
// injection and native Workflow scheduling remain unchanged.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	external "github.com/argoproj/argo-workflows/v4/util/parameterization/externalsecrets"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	var namespace, configFile, index string
	var development bool
	flag.StringVar(&namespace, "namespace", "", "Single destination namespace")
	flag.StringVar(&configFile, "config", "", "Administrator-owned Vault source JSON")
	flag.StringVar(&index, "index", "", "Owned latest-version ConfigMap name")
	flag.BoolVar(&development, "allow-development-http", false, "Permit HTTP only for synthetic local tests")
	flag.Parse()
	if namespace == "" || configFile == "" || index == "" {
		return fmt.Errorf("namespace, index and config are required")
	}
	raw, err := os.ReadFile(configFile)
	if err != nil || len(raw) > 65536 {
		return fmt.Errorf("backend registration unavailable")
	}
	var config external.VaultConfig
	if json.Unmarshal(raw, &config) != nil {
		return fmt.Errorf("backend registration invalid")
	}
	config.AllowHTTPDevelopment = development
	provider, err := external.NewVault(config)
	if err != nil {
		return err
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster configuration required")
	}
	cfg.Timeout = 10 * time.Second
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	// Credentials rotate independently; backend origin, entry and destination form
	// the stable identity. Neither token bytes nor their digest enter Kubernetes.
	sourceID := p.Digest([]string{namespace, index, config.Address, config.Mount, config.Entry, config.Field})[:24]
	materializer := external.Materializer{Client: client, Provider: provider, Namespace: namespace, Name: index, SourceID: sourceID, Validity: 60 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if err = materializer.Reconcile(ctx); err != nil {
			log.Print(err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
