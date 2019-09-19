package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/cmd/server/workflow"
	"google.golang.org/grpc"
)

func main(){

	conn, err := grpc.Dial("localhost:8082", grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	name := "scripts-bash-5ksp4"
	query := workflow.WorkflowQuery{Name: name,}
	client := workflow.NewWorkflowServiceClient(conn)
	wflist, err :=client.List(context.TODO(),&query)
	if err !=nil {
		fmt.Println("errr",err)
	}

	//byte1, err := wflist.Workflows.Marshal()
	for inx,_ := range wflist.Workflows {
		fmt.Println("Response:", wflist.Workflows[inx].Name)
		fmt.Println("/n /n")
	}


}
