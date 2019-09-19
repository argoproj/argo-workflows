package workflow

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	//log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Server struct{
	Namespace 	string
	Clientset versioned.Clientset
}


func NewServer(Namespace string, clientset versioned.Clientset) WorkflowServiceServer{
	return &Server{Namespace:Namespace, Clientset:clientset}
}

func (s *Server) Create(context.Context, *v1alpha1.Workflow) (*WorkflowCreateResponse, error){

	return nil, nil
}


func (s *Server)Get(ctx context.Context, query *WorkflowQuery) (*WorkflowResponse, error){
	wf, err := s.Clientset.ArgoprojV1alpha1().Workflows(s.Namespace).Get(query.Name, v1.GetOptions{})
	fmt.Println("Welcome")
	if err != nil {
		fmt.Println(err)
	}
	var wfRsp WorkflowResponse
	//bytes, err :=  json.Marshal(wf)
	wfRsp.Workflows = wf

	//err = wfRsp.Workflows.Unmarshal(byte)
	fmt.Println("Error : ",err)
	fmt.Println(wfRsp.GetWorkflows())
	//bytes, err := wfRsp.Marshal()
	if err != nil {
		fmt.Println(err)
	}
	//wfRsp.Unmarshal(bytes)
	fmt.Println(wfRsp)

	return &wfRsp, err
}

func (s *Server) List(ctx context.Context, query *WorkflowQuery) (*WorkflowListResponse, error) {
	wfList, err := s.Clientset.ArgoprojV1alpha1().Workflows(s.Namespace).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println(wfList)
	var wfListItem []*v1alpha1.Workflow
	for idx,_ := range wfList.Items{
		wfListItem = append(wfListItem, &wfList.Items[idx])
	}
	var wfListRsp = WorkflowListResponse{}
	wfListRsp.Workflows = wfListItem
	fmt.Println(wfListRsp)
	return &wfListRsp,nil

}