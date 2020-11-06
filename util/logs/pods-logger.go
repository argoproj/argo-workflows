package logs

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	"github.com/argoproj/argo/server/auth"
)

type Callback func(pod *corev1.Pod, data []byte) error

func LogPods(ctx context.Context, namespace string, listOptions metav1.ListOptions, podLogOptions *corev1.PodLogOptions, callback Callback) error {
	coreV1 := auth.GetKubeClient(ctx).CoreV1()
	if podLogOptions == nil {
		podLogOptions = &corev1.PodLogOptions{}
	}
	podInterface := coreV1.Pods(namespace)
	list, err := podInterface.List(listOptions)
	if err != nil {
		return err
	}
	streaming := &sync.Map{}
	streamPod := func(pod *corev1.Pod) {
		logCtx := log.WithFields(log.Fields{"namespace": pod.Namespace, "podName": pod.Name})
		go func(pod *corev1.Pod) {
			err := func() error {
				_, loaded := streaming.LoadOrStore(pod.Name, true)
				if loaded {
					return nil
				}
				defer streaming.Delete(pod.Name)
				stream, err := coreV1.Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions).Stream()
				if err != nil {
					return err
				}
				defer func() { _ = stream.Close() }()
				logCtx.Debug("streaming pod logs")
				s := bufio.NewScanner(stream)
				for {
					select {
					case <-ctx.Done():
						return nil
					default:
						if !s.Scan() {
							return s.Err()
						}
						data := s.Bytes()
						logCtx.Debugln(string(data))
						err = callback(pod, data)
						if err != nil {
							return err
						}
					}
				}
			}()
			if err != nil {
				logCtx.Error(err)
			}
		}(pod)
	}
	for _, p := range list.Items {
		streamPod(p.DeepCopy())
	}
	watcher, err := watch.NewRetryWatcher(list.ResourceVersion, podInterface)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("failed to read event")
			}
			p, ok := event.Object.(*corev1.Pod)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			streamPod(p.DeepCopy()) // deep-copy needed as we use the same pointer in each loop
		}
	}
}
