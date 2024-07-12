package metrics

type Callbacks struct {
	PodPhase          PodPhaseCallback
	WorkflowPhase     WorkflowPhaseCallback
	WorkflowCondition WorkflowConditionCallback
	IsLeader          IsLeaderCallback
}
