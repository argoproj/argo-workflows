package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/util"
	"github.com/prometheus/common/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	//generate()
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	client := workflow.NewWorkflowServiceClient(conn)
	//wf := unmarshalWF(wfStr)
	config := util.InitKubeClient()

	clientConfig := workflow.ClientConfig{
		Host:            config.Host,
		APIPath:         config.APIPath,
		TLSClientConfig: config.TLSClientConfig,
		Username:        config.Username,
		Password:        config.Password,
		AuthProvider:    config.AuthProvider,
	}

	marshalledClientConfig, err := json.Marshal(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	md := metadata.Pairs(workflow.CLIENT_REST_CONFIG, string(marshalledClientConfig))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	fmt.Println(string(marshalledClientConfig))

	wq := workflow.WorkflowListRequest{Namespace: "default"}

	queried, err := client.List(ctx, &wq)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(queried)
}
