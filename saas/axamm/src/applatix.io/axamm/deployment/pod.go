package deployment

const (
	DeploymentIdLabel = "deployment_id"
	PodPhaseRunning   = "running"
)

type DeploymentResult struct {
	Result *DeploymentStatus `json:"result"`
}

type DeploymentStatus struct {
	Name              string `json:"name"`
	AvailableReplicas int    `json:"available_replicas"`
	DesiredReplicas   int    `json:"desired_replicas"`
	Pods              []*Pod `json:"pods"`
}

func (status *DeploymentStatus) GetFailures(deployment_id string) []*Failure {
	failures := []*Failure{}

	pods := status.GetPods(deployment_id)

	if pods != nil {
		for _, pod := range pods {
			if pod.Failure != nil {
				failures = append(failures, pod.Failure)
			}
		}
	}
	return failures
}

func (status *DeploymentStatus) ReadyPods(deployment_id string) int {
	count := 0
	pods := status.GetPods(deployment_id)

	if pods != nil {
		for _, pod := range pods {
			if pod.Ready() {
				count++
			}
		}
	}
	return count
}

func (status *DeploymentStatus) GetPods(deployment_id string) []*Pod {
	pods := []*Pod{}

	if status.Pods != nil {
		for _, pod := range status.Pods {
			if pod.getDeploymentId() == deployment_id {
				pods = append(pods, pod)
			}
		}
	}
	return pods
}

type Pod struct {
	Name       string            `json:"name"`
	Phase      string            `json:"phase"`
	StartTime  string            `json:"start_time"`
	Containers []*Container      `json:"containers"`
	Mtime      int64             `json:"mtime"`
	Failure    *Failure          `json:"failure"`
	Labels     map[string]string `json:"labels"`
}

func (pod *Pod) Ready() bool {
	if len(pod.Containers) != 0 {
		for _, ctn := range pod.Containers {
			if ctn.Ready {
				return true
			}
		}
	}
	return false
}

func (pod *Pod) getDeploymentId() string {
	return pod.Labels[DeploymentIdLabel]
}

type Container struct {
	Name         string               `json:"name"`
	Image        string               `json:"image"`
	ImageId      string               `json:"image_id"`
	ContainerId  string               `json:"container_id"`
	Ready        bool                 `json:"ready"`
	RestartCount int64                `json:"restart_count"`
	State        *ContainerStateGroup `json:"state"`
	LastState    *ContainerStateGroup `json:"last_state"`
}

type ContainerState struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

type ContainerStateGroup struct {
	Running    *ContainerState `json:"running"`
	Terminated *ContainerState `json:"terminated"`
	Waiting    *ContainerState `json:"waiting"`
}

type Failure struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

const (
	TypeHeartBeat          = "HEART_BEAT"
	TypeBirthCry           = "BIRTH_CRY"
	TypeTombStone          = "TOMB_STONE"
	TypeArtifactLoadStart  = "LOADING_ARTIFACTS"
	TypeArtifactLoadFailed = "ARTIFACT_LOAD_FAILED"
)

type PodHeartBeat struct {
	Date int64      `json:"date"`
	Key  string     `json:"key"`
	Data *PodStatus `json:"data"`
}

type PodStatus struct {
	Type      string `json:"type"`
	Version   string `json:"version"`
	PodStatus *Pod   `json:"podStatus"`
}
