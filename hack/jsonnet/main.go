package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Copied from https://github.com/jsonnet-libs/k8s/blob/master/main.go
type Target struct {
	Output       string `json:"output"`
	Openapi      string `json:"openapi"`
	Prefix       string `json:"prefix"`
	PatchDir     string `json:"patchDir"`
	ExtensionDir string `json:"extensionDir"`
}

// Copied from https://github.com/jsonnet-libs/k8s/blob/master/main.go
type Config struct {
	Specs []Target `json:"specs"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopCh
		log.Print("SIGINT/SIGTERM received, shutting down")
		cancel()
	}()

	if err := runCommand(ctx); err != nil {
		log.Fatal(err)
	}
}

func runCommand(ctx context.Context) error {
	pwd, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("unable to get pwd: %w", err)
	}

	listener, err := net.Listen("tcp", ":0")

	if err != nil {
		return fmt.Errorf("failed to open tcp listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.FileServer(http.Dir(fmt.Sprintf("%s/dist", pwd))),
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("listen: %s\n", err)
		}
	}()

	defer listener.Close()
	defer srv.Shutdown(ctx)

	cfgFile, err := ioutil.TempFile("", "argo-jsonnet-generate-cfg")

	if err != nil {
		return fmt.Errorf("unable to create temporary config file: %w", err)
	}

	defer os.Remove(cfgFile.Name())

	err = json.NewEncoder(cfgFile).Encode(Config{
		Specs: []Target{
			{
				Prefix:       "io.argoproj.workflow.",
				PatchDir:     "hack/jsonnet/_custom/argo",
				ExtensionDir: "hack/jsonnet/_extensions/argo",
				Output:       "jsonnet",
				Openapi:      fmt.Sprintf("http://127.0.0.1:%d/swaggifed.swagger.json", port),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("unable to write config to file %s: %w", cfgFile.Name(), err)
	}

	cmd := exec.CommandContext(ctx, "k8s", "-c", cfgFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to run jsonnet generator: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to generate jsonnet lib: %w", err)
	}

	return nil
}
