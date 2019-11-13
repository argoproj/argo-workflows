package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/cmd/server/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"sigs.k8s.io/yaml"

)

var wfStr = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`


func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}



//func generate(){
//
//	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
//	//kubeConfigFlags.AddFlags(flags)
//	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
//	//matchVersionKubeConfigFlags.AddFlags(cmds.PersistentFlags())
//	f := cmdutil.NewFactory(nil)
//	f.RESTClient()
//
//}
func main(){
	//generate()
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	client := workflow.NewWorkflowServiceClient(conn)
	//wf := unmarshalWF(wfStr)
	config := util.InitKubeClient()
	//
	////tc, err :=config.TransportConfig()
	//
	var clientConfig workflow.ClientConfig
	//
	clientConfig.Host = config.Host
	clientConfig.APIPath = config.APIPath
	clientConfig.TLSClientConfig = config.TLSClientConfig
	clientConfig.Username = config.Username
	clientConfig.Password = config.Password
	clientConfig.AuthProvider = config.AuthProvider
	//
	//
	//
	by,err := json.Marshal(clientConfig)
	fmt.Println(err)
	//
	md := metadata.Pairs(workflow.CLIENT_REST_CONFIG, string(by))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//wq := workflow.WorkflowQuery{}
	//created, err :=client.Get(ctx,&wq)
	//
	//fmt.Println("errr",err)
	//
	fmt.Println(string(by))
	wq := workflow.WorkflowListRequest { Namespace:"default"}
	queried, err := client.List(ctx, &wq)
	if err !=nil {
		fmt.Println("errr",err)
	}
	fmt.Println(queried)
	//var wuq workflow.WorkflowUpdateQuery
	////wuq.Workflow = queried
	////wur, err := client.Retry(context.TODO(), &wuq)
	////
	////if err !=nil {
	////	fmt.Println("errr",err)
	////}
	////fmt.Println(wur)
	////
	////name := "scripts-bash-5ksp4"
	////query := workflow.WorkflowQuery{Name: name,}
	////
	////
	//created, err :=client.Create(ctx,wf)
	//if err !=nil {
	//	fmt.Println("errr",err)
	//}
	//fmt.Println(created)
	//
	////byte1, err := wflist.Workflows.Marshal()
	////for inx,_ := range wflist.Workflows {
	////	fmt.Println("Response:", wflist.Workflows[inx].Name)
	////
	////}


}
