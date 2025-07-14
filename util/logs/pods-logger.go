package logs

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type Callback func(pod *corev1.Pod, data []byte) error

func LogPods(ctx context.Context, kubernetesClient kubernetes.Interface, namespace, labelSelector, grep string, podLogOptions *corev1.PodLogOptions, callback Callback) error {
	coreV1 := kubernetesClient.CoreV1()
	if podLogOptions == nil {
		podLogOptions = &corev1.PodLogOptions{}
	}
	rx, err := regexp.Compile(grep)
	if err != nil {
		return err
	}
	podInterface := coreV1.Pods(namespace)
	list, err := podInterface.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	streaming := &sync.Map{}
	streamPod := func(pod *corev1.Pod) {
		logger := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": pod.Namespace, "podName": pod.Name})
		go func(pod *corev1.Pod) {
			err := func() error {
				_, loaded := streaming.LoadOrStore(pod.Name, true)
				if loaded {
					return nil
				}
				defer streaming.Delete(pod.Name)
				stream, err := coreV1.Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions).Stream(ctx)
				if err != nil {
					return err
				}
				defer func() { _ = stream.Close() }()
				logger.Debug(ctx, "streaming pod logs")
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
						if rx.Match(data) {
							logger.Debug(ctx, string(data))
							if err := callback(pod, data); err != nil {
								return err
							}
						}
					}
				}
			}()
			if err != nil {
				logger.WithError(err).Error(ctx, "streaming pod logs")
			}
		}(pod)
	}
	for _, p := range list.Items {
		streamPod(p.DeepCopy())
	}
	for {
		done, err := func() (bool, error) {
			watcher, err := podInterface.Watch(ctx, metav1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				return true, err
			}
			defer watcher.Stop()
			for {
				select {
				case <-ctx.Done():
					return true, nil
				case event, open := <-watcher.ResultChan():
					if !open {
						return false, io.EOF
					}
					p, ok := event.Object.(*corev1.Pod)
					if !ok {
						return true, apierr.FromObject(event.Object)
					}
					streamPod(p.DeepCopy()) // deep-copy needed as we use the same pointer in each loop
				}
			}
		}()
		if done {
			return err
		}
	}
}
