package kubelet

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

const (
	readWSResponseTimeout = time.Minute * 1
	containerShimPrefix   = "://"
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
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, errors.InternalWrapError(fmt.Errorf("unexpected non 200 status code: %d", resp.StatusCode))
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

func (k *kubeletClient) getLogs(namespace, podName, containerName string) (string, error) {
	u, err := url.ParseRequestURI(fmt.Sprintf("https://%s/containerLogs/%s/%s/%s", k.kubeletEndpoint, namespace, podName, containerName))
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	resp, err := k.httpClient.Do(&http.Request{
		Method: http.MethodGet,
		Header: k.httpHeader,
		URL:    u,
	})
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.InternalWrapError(fmt.Errorf("unexpected non 200 status code: %d", resp.StatusCode))
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	return string(b), resp.Body.Close()
}

func getContainerID(container *v1.ContainerStatus) string {
	i := strings.Index(container.ContainerID, containerShimPrefix)
	if i == -1 {
		return ""
	}
	return container.ContainerID[i+len(containerShimPrefix):]
}

func (k *kubeletClient) getContainerStatus(containerID string) (*v1.ContainerStatus, error) {
	podList, err := k.getPodList()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if getContainerID(&container) != containerID {
				continue
			}
			return &container, nil
		}
	}
	return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
}

func (k *kubeletClient) GetContainerLogs(containerID string) (string, error) {
	podList, err := k.getPodList()
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if getContainerID(&container) != containerID {
				continue
			}
			return k.getLogs(pod.Namespace, pod.Name, container.Name)
		}
	}
	return "", errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
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

// TerminatePodWithContainerID invoke the given SIG against the PID1 of the container.
// No-op if the container is on the hostPID
func (k *kubeletClient) TerminatePodWithContainerID(containerID string, sig syscall.Signal) error {
	podList, err := k.getPodList()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	for _, pod := range podList.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if getContainerID(&container) != containerID {
				continue
			}
			if container.State.Terminated != nil {
				log.Infof("Container %s is already terminated: %v", container.ContainerID, container.State.Terminated.String())
				return nil
			}
			if pod.Spec.HostPID {
				return fmt.Errorf("cannot terminate a hostPID Pod %s", pod.Name)
			}
			if pod.Spec.RestartPolicy != "Never" {
				return fmt.Errorf("cannot terminate pod with a %q restart policy", pod.Spec.RestartPolicy)
			}
			u, err := url.ParseRequestURI(fmt.Sprintf("wss://%s/exec/%s/%s/%s?command=/bin/sh&&command=-c&command=kill+-%d+1&output=1&error=1", k.kubeletEndpoint, pod.Namespace, pod.Name, container.Name, sig))
			if err != nil {
				return errors.InternalWrapError(err)
			}
			_, err = k.exec(u)
			return err
		}
	}
	return errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod list", containerID))
}

// CreateArchive exec in the given containerID and create a tarball of the given sourcePath. Works with directory
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
			if getContainerID(&container) != containerID {
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
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	if timeout == 0 {
		timer.Stop()
	} else {
		defer timer.Stop()
	}

	log.Infof("Starting to wait completion of containerID %s ...", containerID)
	for {
		select {
		case <-ticker.C:
			containerStatus, err := k.getContainerStatus(containerID)
			if err != nil {
				return err
			}
			if containerStatus.State.Terminated == nil {
				continue
			}
			log.Infof("ContainerID %q is terminated: %v", containerID, containerStatus.String())
			return nil
		case <-timer.C:
			return fmt.Errorf("timeout after %s", timeout.String())
		}
	}
}
