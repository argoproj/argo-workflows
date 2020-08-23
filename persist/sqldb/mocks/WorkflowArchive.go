// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import labels "k8s.io/apimachinery/pkg/labels"
import mock "github.com/stretchr/testify/mock"

import time "time"
import v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

// WorkflowArchive is an autogenerated mock type for the WorkflowArchive type
type WorkflowArchive struct {
	mock.Mock
}

// ArchiveWorkflow provides a mock function with given fields: wf
func (_m *WorkflowArchive) ArchiveWorkflow(wf *v1alpha1.Workflow) error {
	ret := _m.Called(wf)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1alpha1.Workflow) error); ok {
		r0 = rf(wf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteExpiredWorkflows provides a mock function with given fields: ttl
func (_m *WorkflowArchive) DeleteExpiredWorkflows(ttl time.Duration) error {
	ret := _m.Called(ttl)

	var r0 error
	if rf, ok := ret.Get(0).(func(time.Duration) error); ok {
		r0 = rf(ttl)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteWorkflow provides a mock function with given fields: uid
func (_m *WorkflowArchive) DeleteWorkflow(uid string) error {
	ret := _m.Called(uid)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(uid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetWorkflow provides a mock function with given fields: uid
func (_m *WorkflowArchive) GetWorkflow(uid string) (*v1alpha1.Workflow, error) {
	ret := _m.Called(uid)

	var r0 *v1alpha1.Workflow
	if rf, ok := ret.Get(0).(func(string) *v1alpha1.Workflow); ok {
		r0 = rf(uid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.Workflow)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(uid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListWorkflows provides a mock function with given fields: namespace, minStartAt, maxStartAt, labelRequirements, limit, offset
func (_m *WorkflowArchive) ListWorkflows(namespace string, minStartAt time.Time, maxStartAt time.Time, labelRequirements labels.Requirements, limit int, offset int) (v1alpha1.Workflows, error) {
	ret := _m.Called(namespace, minStartAt, maxStartAt, labelRequirements, limit, offset)

	var r0 v1alpha1.Workflows
	if rf, ok := ret.Get(0).(func(string, time.Time, time.Time, labels.Requirements, int, int) v1alpha1.Workflows); ok {
		r0 = rf(namespace, minStartAt, maxStartAt, labelRequirements, limit, offset)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1alpha1.Workflows)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, time.Time, time.Time, labels.Requirements, int, int) error); ok {
		r1 = rf(namespace, minStartAt, maxStartAt, labelRequirements, limit, offset)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
