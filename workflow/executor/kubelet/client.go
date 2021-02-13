package kubelet

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	execcommon "github.com/argoproj/argo-workflows/v3/workflow/executor/common"
)

const (
	readWSResponseTimeout = time.Minute * 1
)

type kubeletClient struct {
	httpClient      *http.Client
	httpHeader      http.Header
	websocketDialer *websocket.Dialer

	// kubeletEndpoint is host:port without any scheme like:
	// - 127.0.0.1:10250
	// - my-host.com:10250
	kubeletEndpoint string
	namespace       string
	podName         string
}

var _ execcommon.KubernetesClientInterface = &kubeletClient{}

func newKubeletClient(namespace, podName string) (*kubeletClient, error) {
	kubeletHost := os.Getenv(common.EnvVarDownwardAPINodeIP)
	if kubeletHost == "" {
		return nil, fmt.Errorf("empty envvar %s", common.EnvVarDownwardAPINodeIP)
	}
	kubeletPort, _ := strconv.Atoi(os.Getenv(common.EnvVarKubeletPort))
	if kubeletPort == 0 {
		kubeletPort = 10250
		log.Infof("Non configured envvar %s, defaulting the kubelet port to %d", common.EnvVarKubeletPort, kubeletPort)
	}
	b, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	bearerToken := string(b)

	tlsConfig := &tls.Config{}
	if os.Getenv(common.EnvVarKubeletInsecure) == "true" {
		log.Warningf("Using a kubelet client with insecure options")
		tlsConfig.InsecureSkipVerify = true
	} else {
		log.Warningf("Loading service account ca.crt as certificate authority to reach the kubelet api")
		caCert, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.InternalWrapError(fmt.Errorf("fail to load certificate authority: %s", string(caCert)))
		}
		tlsConfig.RootCAs = caCertPool
	}
	return &kubeletClient{
		httpClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   time.Second * 60,
		},
		httpHeader: http.Header{
			"Authorization": {"bearer " + bearerToken},
		},
		websocketDialer: &websocket.Dialer{
			TLSClientConfig:  tlsConfig,
			HandshakeTimeout: time.Second * 5,
		},
		kubeletEndpoint: fmt.Sprintf("%s:%d", kubeletHost, kubeletPort),
		namespace:       namespace,
		podName:         podName,
	}, nil
}

func checkHTTPErr(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return errors.InternalWrapError(fmt.Errorf("unexpected non 200 status code: %d, body: %s", resp.StatusCode, string(b)))
	}
	return nil
}

func (k *kubeletClient) getPod() (*corev1.Pod, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("https://%s/pods", k.kubeletEndpoint))
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	resp, err := k.httpClient.Do(&http.Request{
		Method: http.MethodGet,
		Header: k.httpHeader,
		URL:    u,
	})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	log.Infof("List pod %d (kubelet)", resp.StatusCode) // log that we are listing pods from Kubelet
	err = checkHTTPErr(resp)
	if err != nil {
		return nil, err
	}
	podListDecoder := json.NewDecoder(resp.Body)
	podList := &corev1.PodList{}
	err = podListDecoder.Decode(podList)
	if err != nil {
		_ = resp.Body.Close()
		return nil, errors.InternalWrapError(err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}
	for _, item := range podList.Items {
		if item.Namespace == k.namespace && item.Name == k.podName {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("pod %q is not found in the pod list", k.podName)
}

func (k *kubeletClient) GetLogStream(containerName string) (io.ReadCloser, error) {
	resp, err := k.doRequestLogs(k.namespace, k.podName, containerName)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (k *kubeletClient) doRequestLogs(namespace, podName, containerName string) (*http.Response, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("https://%s/containerLogs/%s/%s/%s", k.kubeletEndpoint, namespace, podName, containerName))
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	resp, err := k.httpClient.Do(&http.Request{
		Method: http.MethodGet,
		Header: k.httpHeader,
		URL:    u,
	})
	if err != nil {
		return nil, err
	}
	err = checkHTTPErr(resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (k *kubeletClient) GetContainerStatus(ctx context.Context, containerName string) (*corev1.Pod, *corev1.ContainerStatus, error) {
	pod, containerStatus, err := k.GetContainerStatuses(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range containerStatus {
		if containerName == s.Name {
			return pod, &s, nil
		}
	}
	return nil, nil, fmt.Errorf("container %q is not found in the pod", containerName)
}

func (k *kubeletClient) GetContainerStatuses(ctx context.Context) (*corev1.Pod, []corev1.ContainerStatus, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, nil, errors.InternalWrapError(err)
	}
	return pod, pod.Status.ContainerStatuses, nil
}

func (k *kubeletClient) exec(u *url.URL) (*url.URL, error) {
	_, resp, err := k.websocketDialer.Dial(u.String(), k.httpHeader)
	if resp == nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusFound {
		return nil, err
	}
	log.Infof("Exec for %s with status code: %d", u.String(), resp.StatusCode)
	redirect, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	redirect.Scheme = "ws"
	log.Infof("Exec for %s returns URL: %s", u.String(), redirect.String())
	return redirect, nil
}

func (k *kubeletClient) readFileContents(u *url.URL) (*bytes.Buffer, error) {
	conn, _, err := k.websocketDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	timeout := time.NewTimer(readWSResponseTimeout)
	defer timeout.Stop()

	buf := &bytes.Buffer{}
	for {
		select {
		case <-timeout.C:
			return nil, fmt.Errorf("timeout of %s reached while reading file contents", readWSResponseTimeout)

		default:
			_, b, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return buf, conn.Close()
				}
				log.Errorf("Unexpected error while reading messages on WS: %v", err)
				_ = conn.Close()
				return buf, err
			}
			if len(b) < 1 {
				continue
			}
			i := 0
			// skip SOH (start of heading)
			if int(b[0]) == 1 {
				i = 1
			}
			_, err = buf.Write(b[i:])
			if err != nil {
				log.Errorf("Unexpected error while reading messages on WS: %v", err)
				_ = conn.Close()
				return nil, err
			}
		}
	}
}

// createArchive exec in the given containerName and create a tarball of the given sourcePath. Works with directory
func (k *kubeletClient) CreateArchive(ctx context.Context, containerName, sourcePath string) (*bytes.Buffer, error) {
	return k.getCommandOutput(containerName, fmt.Sprintf("command=tar&command=-cf&command=-&command=%s&output=1", sourcePath))
}

// GetFileContents exec in the given containerName and cat the given sourcePath.
func (k *kubeletClient) GetFileContents(containerName, sourcePath string) (*bytes.Buffer, error) {
	return k.getCommandOutput(containerName, fmt.Sprintf("command=cat&command=%s&output=1", sourcePath))
}

func (k *kubeletClient) getCommandOutput(containerName, command string) (*bytes.Buffer, error) {
	pod, container, err := k.GetContainerStatus(context.Background(), containerName)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	if container.State.Terminated != nil {
		return nil, fmt.Errorf("container %q is terminated: %v", containerName, container.State.Terminated.String())
	}
	u, err := url.ParseRequestURI(fmt.Sprintf("wss://%s/exec/%s/%s/%s?%s", k.kubeletEndpoint, pod.Namespace, pod.Name, containerName, command))
	if err != nil {
		return nil, err
	}
	u, err = k.exec(u)
	if err != nil {
		return nil, err
	}
	return k.readFileContents(u)
}

// WaitForTermination of the given container, set the timeout to 0 to discard it
func (k *kubeletClient) WaitForTermination(ctx context.Context, containerNames []string, timeout time.Duration) error {
	return execcommon.WaitForTermination(ctx, k, containerNames, timeout)
}

func (k *kubeletClient) KillContainer(pod *corev1.Pod, container *corev1.ContainerStatus, sig syscall.Signal) error {
	u, err := url.ParseRequestURI(fmt.Sprintf("wss://%s/exec/%s/%s/%s?command=/bin/sh&&command=-c&command=kill+-%d+1&output=1&error=1", k.kubeletEndpoint, pod.Namespace, pod.Name, container.Name, sig))
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_, err = k.exec(u)
	return err
}

func (k *kubeletClient) KillGracefully(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	return execcommon.KillGracefully(ctx, k, containerNames, terminationGracePeriodDuration)
}

func (k *kubeletClient) CopyArchive(ctx context.Context, containerName, sourcePath, destPath string) error {
	return execcommon.CopyArchive(ctx, k, containerName, sourcePath, destPath)
}
