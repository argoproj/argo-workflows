package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"

	"github.com/spf13/cobra"
)

const (
	DefaultRegistry  = "docker.io"
	DefaultNamespace = "argoproj"
)

var (
	dockerPath string
	registry   = DefaultRegistry
	namespace  = DefaultNamespace
	homePath   string
)

var (
	clusterArgs clusterFlags
)

type clusterFlags struct {
	registry        string // --registry
	registrySecrets string // --registry-secrets
	imageNamespace  string // --image-namespace
	imageVersion    string // --image-version
}

func init() {
	RootCmd.AddCommand(clusterCmd)

	clusterCmd.Flags().StringVar(&clusterArgs.registry, "registry", "", fmt.Sprintf("argocluster registry (e.g. %s)", DefaultRegistry))
	clusterCmd.Flags().StringVar(&clusterArgs.registrySecrets, "registry-secrets", "", fmt.Sprint("base64 encoded docker config file container secrets for docker registry"))
	clusterCmd.Flags().StringVar(&clusterArgs.imageNamespace, "image-namespace", "", fmt.Sprintf("argocluster image namespace (e.g. %s)", DefaultNamespace))
	clusterCmd.Flags().StringVar(&clusterArgs.imageVersion, "image-version", "", fmt.Sprintf("argocluster image version (e.g. %s)", Version))
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: fmt.Sprintf("Type 'argo cluster' to start cluster management shell"),
	Run:   clusterShell,
}

func clusterShell(cmd *cobra.Command, args []string) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		log.Fatalf("docker could not be found on your path. Make sure that docker is installed and on your path")
	}

	registry := getSetting(clusterArgs.registry, "ARGO_CLUSTER_DIST_REGISTRY", DefaultRegistry)
	registrySecret := getSetting(clusterArgs.registrySecrets, "ARGO_CLUSTER_DIST_REGISTRY_SECRETS", "")
	namespace := getSetting(clusterArgs.imageNamespace, "ARGO_CLUSTER_IMAGE_NAMESPACE", DefaultNamespace)
	version := getSetting(clusterArgs.imageVersion, "ARGO_CLUSTER_IMAGE_VERSION", Version)

	usr, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}
	homePath := usr.HomeDir

	clusterManagerImage := fmt.Sprintf("%s/%s/axclustermanager:%s", registry, namespace, version)
	fmt.Printf("Getting the most up to date docker image (%s) for cluster management...\n", clusterManagerImage)
	runCmdTTY(dockerPath, "pull", clusterManagerImage)

	fmt.Println("Entering cluster management shell...")
	volAWS := fmt.Sprintf("%s/.aws:/root/.aws", homePath)
	volKube := fmt.Sprintf("%s/.kube:/tmp/ax_kube", homePath)
	volSSH := fmt.Sprintf("%s/.ssh:/root/.ssh", homePath)
	volArgo := fmt.Sprintf("%s/.argo:/root/.argo", homePath)

	envRegistry := fmt.Sprintf("ARGO_DIST_REGISTRY=%s", registry)
	envNamespace := fmt.Sprintf("AX_NAMESPACE=%s", namespace)
	envVersion := fmt.Sprintf("AX_VERSION=%s", version)
	envRegistrySecrets := fmt.Sprintf("ARGO_DIST_REGISTRY_SECRETS=%s", registrySecret)

	runCmdTTY(dockerPath, "run",
		"-it", "--net", "host",
		"-v", volAWS, "-v", volKube, "-v", volSSH, "-v", volArgo, // map required volumes from home directory
		"-e", envRegistry, "-e", envNamespace, "-e", envVersion, "-e", envRegistrySecrets, // Create env vars required for cluster manager
		clusterManagerImage)
}
