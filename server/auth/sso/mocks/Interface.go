// Code generated by mockery v2.51.0. DO NOT EDIT.

package mocks

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"

	types "github.com/argoproj/argo-workflows/v3/server/auth/types"
)

// Interface is an autogenerated mock type for the Interface type
type Interface struct {
	mock.Mock
}

// Authorize provides a mock function with given fields: authorization
func (_m *Interface) Authorize(authorization string) (*types.Claims, error) {
	ret := _m.Called(authorization)

	if len(ret) == 0 {
		panic("no return value specified for Authorize")
	}

	var r0 *types.Claims
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*types.Claims, error)); ok {
		return rf(authorization)
	}
	if rf, ok := ret.Get(0).(func(string) *types.Claims); ok {
		r0 = rf(authorization)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Claims)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(authorization)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HandleCallback provides a mock function with given fields: writer, request
func (_m *Interface) HandleCallback(writer http.ResponseWriter, request *http.Request) {
	_m.Called(writer, request)
}

// HandleRedirect provides a mock function with given fields: writer, request
func (_m *Interface) HandleRedirect(writer http.ResponseWriter, request *http.Request) {
	_m.Called(writer, request)
}

// IsRBACEnabled provides a mock function with no fields
func (_m *Interface) IsRBACEnabled() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsRBACEnabled")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// NewInterface creates a new instance of Interface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *Interface {
	mock := &Interface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
