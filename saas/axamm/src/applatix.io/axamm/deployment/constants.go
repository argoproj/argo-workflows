package deployment

import "applatix.io/axerror"

const (
	DeployStateInit        = "Init"
	DeployStateWaiting     = "Waiting"
	DeployStateError       = "Error"
	DeployStateActive      = "Active"
	DeployStateStopping    = "Stopping"
	DeployStateStopped     = "Stopped"
	DeployStateTerminating = "Terminating"
	DeployStateTerminated  = "Terminated"
	DeployStateUpgrading   = "Upgrading"
)

const (
	ErrMissingHeartBeat    = "ERR_MISSING_HEART_BEAT"
	ErrCreatingDeployment  = "ERR_CREATING_DEPLOYMENT"
	ErrDeletingDeployment  = "ERR_DELETING_DEPLOYMENT"
	ErrReservingFixture    = "ERR_RESERVING_FIXTURE"
	ErrReleasingFixture    = "ERR_RELEASING_FIXTURE"
	ErrDeploymentDegraded  = "DEGRADED"
	ErrInstanceComingUp    = "COMING_UP"
	ErrInstanceScalingDown = "SCALING_DOWN"
	ErrDeploymentTimeout   = "TIMEOUT"
	ErrScalingDeployment   = "ERR_SCALING_DEPLOYMENT"
	ErrReserveResource     = "ERR_RESERVE_RESOURCE"
	ErrReleaseResource     = "ERR_RELEASE_RESOURCE"
	ErrNotEnoughResource   = "NOT_ENOUGH_RESOURCE"
	ErrUpgradingDeployment = "ERR_UPGRADING_DEPLOYMENT"
)

const (
	InfoDeploymentActive = "ACTIVE"
	InfoDeployed         = "DEPLOYED"
	InfoDeploying        = "DEPLOYING"
	InfoUpgrading        = "UPGRADING"
)

const (
	RedisDeployUpKeyTemplate     = "deployment-up-key-%s"
	RedisDeployUpListKeyTemplate = "deployment-up-list-key-%s"
	RedisDeployUpdate            = "deployments-status"
)

var deployStateMap map[string]map[string]int = map[string]map[string]int{
	DeployStateInit:        map[string]int{DeployStateInit: 1, DeployStateWaiting: 1, DeployStateActive: 0, DeployStateError: 0, DeployStateStopping: 0, DeployStateStopped: 0, DeployStateTerminating: 0, DeployStateTerminated: 1, DeployStateUpgrading: 0},
	DeployStateWaiting:     map[string]int{DeployStateInit: 0, DeployStateWaiting: 1, DeployStateActive: 1, DeployStateError: 1, DeployStateStopping: 1, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 0},
	DeployStateActive:      map[string]int{DeployStateInit: 0, DeployStateWaiting: 1, DeployStateActive: 1, DeployStateError: 1, DeployStateStopping: 1, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 1},
	DeployStateError:       map[string]int{DeployStateInit: 0, DeployStateWaiting: 1, DeployStateActive: 1, DeployStateError: 1, DeployStateStopping: 1, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 1},
	DeployStateStopping:    map[string]int{DeployStateInit: 0, DeployStateWaiting: 0, DeployStateActive: 0, DeployStateError: 0, DeployStateStopping: 1, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 0},
	DeployStateStopped:     map[string]int{DeployStateInit: 0, DeployStateWaiting: 1, DeployStateActive: 0, DeployStateError: 0, DeployStateStopping: 0, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 1},
	DeployStateTerminating: map[string]int{DeployStateInit: 0, DeployStateWaiting: 0, DeployStateActive: 0, DeployStateError: 0, DeployStateStopping: 0, DeployStateStopped: 0, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 0},
	DeployStateTerminated:  map[string]int{DeployStateInit: 0, DeployStateWaiting: 0, DeployStateActive: 0, DeployStateError: 0, DeployStateStopping: 0, DeployStateStopped: 0, DeployStateTerminating: 0, DeployStateTerminated: 1, DeployStateUpgrading: 0},
	DeployStateUpgrading:   map[string]int{DeployStateInit: 0, DeployStateWaiting: 0, DeployStateActive: 1, DeployStateError: 1, DeployStateStopping: 1, DeployStateStopped: 1, DeployStateTerminating: 1, DeployStateTerminated: 1, DeployStateUpgrading: 1},
}

const (
	// Timeout which we allow a deployment remain in init state before timing out and terminating the deployment
	DEPLOYMENT_INIT_TIMEOUT_SEC = 1800
)

func (d *Deployment) checkStateChange(new string) (*axerror.AXError, int) {
	if deployStateMap[d.Status][new] == 1 {
		return nil, 200
	}
	return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Deployment state change from %v to %v is not valid option.", d.Status, new), axerror.REST_BAD_REQ
}
