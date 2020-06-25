package events

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type Controller interface {
	Queue(namespace, name, nodeID string, art *wfv1.HTTPArtifact)
	Run(ctx context.Context)
}

type controller struct {
	wfIf  versioned.Interface
	queue workqueue.RateLimitingInterface
}

func NewController(wfIf versioned.Interface) Controller {
	return &controller{wfIf, workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())}
}

type item struct {
	namespace string
	name      string
	nodeID    string
	artifact  *wfv1.HTTPArtifact
}

func (c *controller) Queue(namespace, name, nodeID string, art *wfv1.HTTPArtifact) {
	c.queue.Add(item{namespace, name, nodeID, art})
}

func (c *controller) Run(ctx context.Context) {
	defer c.queue.ShutDown()
	go wait.Until(c.runWorker, time.Second, ctx.Done())
	<-ctx.Done()
}

func (c *controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	next, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(next)
	item := next.(item)
	logCtx := log.WithFields(log.Fields{"namespace": item.namespace, "workflow": item.name, "nodeID": item.nodeID})
	httpError := makeHTTPRequest(item.artifact)

	// TODO retry

	wfIf := c.wfIf.ArgoprojV1alpha1().Workflows(item.namespace)
	wf, err := wfIf.Get(item.name, metav1.GetOptions{})
	if err != nil {
		logCtx.WithError(err).Error("failed to get workflow")
		return true
	}
	node := wf.Status.Nodes.FindByID(item.nodeID)
	if node == nil {
		logCtx.Error("failed to find node")
		return true
	}
	if httpError != nil {
		node.Phase = wfv1.NodeFailed
		node.Message = fmt.Sprintf("failed to make HTTP request: %v", httpError.Error())
	} else {
		node.Phase = wfv1.NodeSucceeded
	}
	wf.Status.Nodes[node.ID] = *node
	_, err = wfIf.Update(wf)
	if err != nil {
		logCtx.WithError(err).Error("failed to update workflow")
	}
	return true
}
func makeHTTPRequest(in *wfv1.HTTPArtifact) error {
	data, err := json.Marshal(in.Body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(in.Method, in.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	for _, h := range in.Headers {
		req.Header.Add(h.Name, h.Value)
	}
	resp, err := (&http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: in.InsecureSkipVerify}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP request failed: %v", resp.Status)
	}
	return nil
}
