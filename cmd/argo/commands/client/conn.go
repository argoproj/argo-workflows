package client

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var ArgoServer string

func AddArgoServerFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&ArgoServer, "argo-server", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
}

func GetClientConn() *grpc.ClientConn {
	conn, err := grpc.Dial(ArgoServer, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return conn
}
