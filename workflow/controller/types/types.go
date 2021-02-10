package types

import wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

type InitNodeFunc = func(nodeType wfv1.NodeType, phase wfv1.NodePhase) *wfv1.NodeStatus
