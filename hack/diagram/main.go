package main

import (
	"github.com/blushft/go-diagrams/diagram"
	"github.com/blushft/go-diagrams/nodes/apps"
	"github.com/blushft/go-diagrams/nodes/gcp"
	"github.com/blushft/go-diagrams/nodes/generic"
	"github.com/blushft/go-diagrams/nodes/k8s"
	"github.com/blushft/go-diagrams/nodes/programming"
	"github.com/blushft/go-diagrams/nodes/saas"
)

func main() {
	d, err := diagram.New(diagram.Filename("diagram"), diagram.Label("Argo Workflows"), diagram.Direction("LR"))
	if err != nil {
		panic(err)
	}

	kubeCluster := diagram.NewGroup("kubernetes-cluster").Label("Kubernetes Cluster")

	user := apps.Client.User(diagram.NodeLabel("User"))
	browser := generic.Device.Tablet(diagram.NodeLabel("Web Browser"))
	argoCLI := programming.Language.Bash(diagram.NodeLabel("Argo CLI"))
	kubectl := programming.Language.Bash(diagram.NodeLabel("Kubectl CLI"))
	webhook := apps.Vcs.Git(diagram.NodeLabel("WebHook"))
	apiClient := programming.Language.Python(diagram.NodeLabel("API Client"))
	prom := apps.Monitoring.Prometheus(diagram.NodeLabel("Prometheus Collector"))
	lb := gcp.Network.LoadBalancing(diagram.NodeLabel("Network Load Balancer"))
	svc := k8s.Network.Svc(diagram.NodeLabel("Argo Server Service"))
	argoServer := k8s.Compute.Pod(diagram.NodeLabel("3 x Argo Server"))
	workflowController := k8s.Compute.Pod(diagram.NodeLabel("1 x Workflow Controller"))
	k8sapi := k8s.Controlplane.Api(diagram.NodeLabel("Kubernetes API"))
	workflowArchive := gcp.Database.Sql(diagram.NodeLabel("Workflow Archive (e.g. MySQL)"))
	workflowPod := k8s.Compute.Pod(diagram.NodeLabel("1000s x Workflow Pod"))
	storage := gcp.Database.Datastore(diagram.NodeLabel("Artifact Store (e.g. S3)"))
	authProvider := saas.Identity.Auth0(diagram.NodeLabel("OAuth Provider"))

	kubeCluster.NewGroup("argo").
		Label("argo system namespace").
		Add(svc, argoServer, workflowController).
		Connect(svc, argoServer, diagram.Forward())

	kubeCluster.NewGroup("user namespace").
		Label("user namespace ").
		Add(workflowPod)

	kubeCluster.NewGroup("kube-system").
		Label("kube-system namespace").
		Add(k8sapi)

	d.Connect(user, browser, diagram.Forward()).Group(kubeCluster)
	d.Connect(user, argoCLI, diagram.Forward()).Group(kubeCluster)
	d.Connect(user, kubectl, diagram.Forward()).Group(kubeCluster)
	d.Connect(browser, lb, diagram.Forward()).Group(kubeCluster)
	d.Connect(apiClient, lb, diagram.Forward()).Group(kubeCluster)
	d.Connect(webhook, lb, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoCLI, lb, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoCLI, k8sapi, diagram.Forward()).Group(kubeCluster)
	d.Connect(lb, svc, diagram.Forward()).Group(kubeCluster)
	d.Connect(kubectl, k8sapi, diagram.Forward()).Group(kubeCluster)
	d.Connect(prom, workflowController, diagram.Forward()).Group(kubeCluster)
	d.Connect(workflowController, storage, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoServer, authProvider, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoServer, storage, diagram.Forward()).Group(kubeCluster)
	d.Connect(workflowPod, storage, diagram.Forward()).Group(kubeCluster)
	d.Connect(workflowPod, k8sapi, diagram.Forward()).Group(kubeCluster)
	d.Connect(workflowController, k8sapi, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoServer, k8sapi, diagram.Forward()).Group(kubeCluster)
	d.Connect(argoServer, workflowArchive, diagram.Forward()).Group(kubeCluster)
	d.Connect(workflowController, workflowArchive, diagram.Forward()).Group(kubeCluster)

	if err := d.Render(); err != nil {
		panic(err)
	}
}
