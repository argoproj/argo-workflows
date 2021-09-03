package pipeline

import (
	"context"
	"io"

	dfv1 "github.com/argoproj-labs/argo-dataflow/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	pipelinepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/pipeline"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/logs"
)

type server struct{}

func (s *server) ListPipelines(ctx context.Context, req *pipelinepkg.ListPipelinesRequest) (*dfv1.PipelineList, error) {
	client := auth.GetDynamicClient(ctx)
	opts := metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = *req.ListOptions
	}
	list, err := client.Resource(dfv1.PipelineGroupVersionResource).Namespace(req.Namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	items := make([]dfv1.Pipeline, len(list.Items))
	for i, un := range list.Items {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &items[i]); err != nil {
			return nil, err
		}
	}
	return &dfv1.PipelineList{Items: items}, nil
}

func (s *server) WatchPipelines(req *pipelinepkg.ListPipelinesRequest, svr pipelinepkg.PipelineService_WatchPipelinesServer) error {
	ctx := svr.Context()
	client := auth.GetDynamicClient(ctx)
	opts := metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = *req.ListOptions
	}
	watcher, err := client.Resource(dfv1.PipelineGroupVersionResource).Namespace(req.Namespace).Watch(ctx, opts)
	if err != nil {
		return err
	}
	defer watcher.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-watcher.ResultChan():
			if !open {
				return io.EOF
			}
			un, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			pl := &dfv1.Pipeline{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, pl); err != nil {
				return err
			}
			if err := svr.Send(&pipelinepkg.PipelineWatchEvent{Type: string(event.Type), Object: pl}); err != nil {
				return err
			}
		}
	}
}

func (s *server) GetPipeline(ctx context.Context, req *pipelinepkg.GetPipelineRequest) (*dfv1.Pipeline, error) {
	client := auth.GetDynamicClient(ctx)
	opts := metav1.GetOptions{}
	if req.GetOptions != nil {
		opts = *req.GetOptions
	}
	un, err := client.Resource(dfv1.PipelineGroupVersionResource).Namespace(req.Namespace).Get(ctx, req.Name, opts)
	if err != nil {
		return nil, err
	}
	item := &dfv1.Pipeline{}
	return item, runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, item)
}

func (s *server) RestartPipeline(ctx context.Context, req *pipelinepkg.RestartPipelineRequest) (*pipelinepkg.RestartPipelineResponse, error) {
	client := auth.GetKubeClient(ctx)
	err := client.CoreV1().Pods(req.Namespace).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: dfv1.KeyPipelineName + "=" + req.Name},
	)
	if err != nil {
		return nil, err
	}
	return &pipelinepkg.RestartPipelineResponse{}, nil
}

func (s *server) DeletePipeline(ctx context.Context, req *pipelinepkg.DeletePipelineRequest) (*pipelinepkg.DeletePipelineResponse, error) {
	client := auth.GetDynamicClient(ctx)
	opts := metav1.DeleteOptions{}
	if req.DeleteOptions != nil {
		opts = *req.DeleteOptions
	}
	err := client.Resource(dfv1.PipelineGroupVersionResource).Namespace(req.Namespace).Delete(ctx, req.Name, opts)
	if err != nil {
		return nil, err
	}
	return &pipelinepkg.DeletePipelineResponse{}, nil
}

func (s *server) PipelineLogs(in *pipelinepkg.PipelineLogsRequest, svr pipelinepkg.PipelineService_PipelineLogsServer) error {
	labelSelector := dfv1.KeyPipelineName
	if in.Name != "" {
		labelSelector += "=" + in.Name
	}
	if in.StepName != "" {
		labelSelector += "," + dfv1.KeyStepName + "=" + in.StepName
	}
	return logs.LogPods(
		svr.Context(),
		in.Namespace,
		labelSelector,
		in.Grep,
		in.PodLogOptions,
		func(pod *corev1.Pod, data []byte) error {
			now := metav1.Now()
			return svr.Send(&pipelinepkg.LogEntry{
				Namespace:    pod.Namespace,
				PipelineName: pod.Labels[dfv1.KeyPipelineName],
				StepName:     pod.Labels[dfv1.KeyStepName],
				Time:         &now,
				Msg:          string(data),
			})
		},
	)
}

func (s *server) WatchSteps(req *pipelinepkg.WatchStepRequest, svr pipelinepkg.PipelineService_WatchStepsServer) error {
	ctx := svr.Context()
	client := auth.GetDynamicClient(ctx)
	opts := metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = *req.ListOptions
	}
	watcher, err := client.Resource(dfv1.StepGroupVersionResource).Namespace(req.Namespace).Watch(ctx, opts)
	if err != nil {
		return err
	}
	defer watcher.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-watcher.ResultChan():
			if !open {
				return io.EOF
			}
			un, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			step := &dfv1.Step{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, step); err != nil {
				return err
			}
			if err := svr.Send(&pipelinepkg.StepWatchEvent{Type: string(event.Type), Object: step}); err != nil {
				return err
			}
		}
	}
}

func NewPipelineServer() pipelinepkg.PipelineServiceServer {
	return &server{}
}
