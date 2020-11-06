package logs

import (
	"bufio"
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	"github.com/argoproj/argo/server/auth"
)

type Callback func(pod *corev1.Pod, data []byte) error

func LogLabelledPods(ctx context.Context, namespace string, listOptions metav1.ListOptions, podLogOptions *corev1.PodLogOptions, callback Callback) error {
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
		go func() {
			err := func() error {
				_, loaded := streaming.LoadOrStore(pod.Name, true)
				if loaded {
					return nil
				}
				logCtx.Debug("streaming pod logs")
				defer streaming.Delete(pod.Name)
				stream, err := coreV1.Pods(namespace).GetLogs(pod.Name, podLogOptions).Stream()
				if err != nil {
					return err
				}
				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					data := scanner.Bytes()
					logCtx.Debugln(string(data))
					err = callback(pod, data)
					if err != nil {
						return err
					}
				}
				return nil
			}()
			if err != nil {
				logCtx.Error(err)
			}
		}()
	}
	for _, p := range list.Items {
		streamPod(&p)
	}
	watcher, err := watch.NewRetryWatcher(list.ResourceVersion, podInterface)
	if err != nil {
		return err
	}
	for event := range watcher.ResultChan() {
		p, ok := event.Object.(*corev1.Pod)
		if !ok {
			return apierr.FromObject(event.Object)
		}
		streamPod(p.DeepCopy()) // deep-copy needed as we use the same pointer in each loop
	}
	return nil
}
