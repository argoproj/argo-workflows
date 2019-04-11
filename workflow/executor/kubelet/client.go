package kubelet

import (
	"bytes"
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

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
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
}

var _ execcommon.KubernetesClientInterface = &kubeletClient{}

func newKubeletClient() (*kubeletClient, error) {
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
		tlsConfig.BuildNameToCertificate()
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

func (k *kubeletClient) getPodList() (*v1.PodList, error) {
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
	err = checkHTTPErr(resp)
	if err != nil {
		return nil, err
	}
	podListDecoder := json.NewDecoder(resp.Body)
	podList := &v1.PodList{}
	err = podListDecoder.Decode(podList)
	if err != nil {
		_ = resp.Body.Close()
		return nil, errors.InternalWrapError(err)
	}
	return podList, resp.Body.Close()
}

func (k *kubeletClient) GetLogStream(containerID string) (io.ReadCloser, error) {
	podList, err := k.getPodList()
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if execcommon.GetContainerID(&container) != containerID {
				continue
			}
			resp, err := k.doRequestLogs(pod.Namespace, pod.Name, container.Name)
			if err != nil {
				return nil, err
			}
			return resp.Body, nil
		}
	}
	return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
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

func (k *kubeletClient) GetContainerStatus(containerID string) (*v1.Pod, *v1.ContainerStatus, error) {
	podList, err := k.getPodList()
	if err != nil {
		return nil, nil, errors.InternalWrapError(err)
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if execcommon.GetContainerID(&container) != containerID {
				continue
			}
			return &pod, &container, nil
		}
	}
	return nil, nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
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

// createArchive exec in the given containerID and create a tarball of the given sourcePath. Works with directory
func (k *kubeletClient) CreateArchive(containerID, sourcePath string) (*bytes.Buffer, error) {
	return k.getCommandOutput(containerID, fmt.Sprintf("command=tar&command=-cf&command=-&command=%s&output=1", sourcePath))
}

// GetFileContents exec in the given containerID and cat the given sourcePath.
func (k *kubeletClient) GetFileContents(containerID, sourcePath string) (*bytes.Buffer, error) {
	return k.getCommandOutput(containerID, fmt.Sprintf("command=cat&command=%s&output=1", sourcePath))
}

func (k *kubeletClient) getCommandOutput(containerID, command string) (*bytes.Buffer, error) {
	podList, err := k.getPodList()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if execcommon.GetContainerID(&container) != containerID {
				continue
			}
			if container.State.Terminated != nil {
				err = fmt.Errorf("container %s is terminated: %v", container.ContainerID, container.State.Terminated.String())
				return nil, err
			}
			u, err := url.ParseRequestURI(fmt.Sprintf("wss://%s/exec/%s/%s/%s?%s", k.kubeletEndpoint, pod.Namespace, pod.Name, container.Name, command))
			if err != nil {
				return nil, err
			}
			u, err = k.exec(u)
			if err != nil {
				return nil, err
			}
			return k.readFileContents(u)
		}
	}
	return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
}

// WaitForTermination of the given containerID, set the timeout to 0 to discard it
func (k *kubeletClient) WaitForTermination(containerID string, timeout time.Duration) error {
	return execcommon.WaitForTermination(k, containerID, timeout)
}

func (k *kubeletClient) KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error {
	u, err := url.ParseRequestURI(fmt.Sprintf("wss://%s/exec/%s/%s/%s?command=/bin/sh&&command=-c&command=kill+-%d+1&output=1&error=1", k.kubeletEndpoint, pod.Namespace, pod.Name, container.Name, sig))
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_, err = k.exec(u)
	return err
}

func (k *kubeletClient) KillGracefully(containerID string) error {
	return execcommon.KillGracefully(k, containerID)
}

func (k *kubeletClient) CopyArchive(containerID, sourcePath, destPath string) error {
	return execcommon.CopyArchive(k, containerID, sourcePath, destPath)
}
