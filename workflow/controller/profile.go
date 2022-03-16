package controller

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
)

type profile struct {
	policyDef
	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig         *rest.Config
	kubernetesClient   kubernetes.Interface
	workflowClient     wfclientset.Interface
	metadataClient     metadata.Interface
	podInformer        cache.SharedIndexInformer
	podGCInformer      cache.SharedIndexInformer
	taskResultInformer cache.SharedIndexInformer
	done               func()
}

func (p *profile) run(done <-chan struct{}) {
	go p.podInformer.Run(done)
	go p.podGCInformer.Run(done)
	go p.taskResultInformer.Run(done)
	<-done
}

func (p *profile) hasSynced() bool {
	return p.taskResultInformer.HasSynced() && p.podInformer.HasSynced() && p.podGCInformer.HasSynced()
}
