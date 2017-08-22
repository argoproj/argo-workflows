// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package native

import "applatix.io/axerror"
import "applatix.io/axops/session"
import "applatix.io/axops/user"
import (
	"applatix.io/axops/auth"
)

var (
	ErrMissingPassword       = axerror.ERR_API_INVALID_PARAM.NewWithMessage("You must provide password.")
	ErrMissingUsername       = axerror.ERR_API_INVALID_PARAM.NewWithMessage("You must provide username.")
	ErrMissingUser           = axerror.ERR_API_AUTH_FAILED.NewWithMessage("The user doesn't exist.")
	ErrPasswordMismatch      = axerror.ERR_API_AUTH_FAILED.NewWithMessage("The username/password is invalid.")
	ErrNotNativeScheme       = axerror.ERR_API_AUTH_FAILED.NewWithMessage("The user account is not managed in local, please login via other methods.")
	ErrNotNativeNotSupported = axerror.ERR_API_AUTH_FAILED.NewWithMessage("The user account is not managed in local, the operation is not supported.")
)

type NativeScheme struct {
	*auth.BaseScheme
}

const AUTH_NATIVE_SCHEME = "native"

func (s *NativeScheme) Name() string {
	return AUTH_NATIVE_SCHEME
}

func (s *NativeScheme) Scheme() map[string]interface{} {
	scheme := map[string]interface{}{}
	scheme["name"] = s.Name()
	scheme["enabled"] = true
	return scheme
}

func (s *NativeScheme) Verify(username, password string) (*user.User, *session.Session, *axerror.AXError) {
	u, axErr := user.GetUserByName(username)
	if axErr != nil {
		return nil, nil, axErr
	}

	if u == nil {
		return nil, nil, ErrMissingUser
	}

	if u.State == user.UserStateBanned {
		return nil, nil, auth.ErrUserBanned
	}

	if u.State == user.UserStateInit {
		return nil, nil, auth.ErrUserNotConfirmed
	}

	hasNative := false
	for _, scheme := range u.AuthSchemes {
		if scheme == "native" {
			hasNative = true
		}
	}

	if !hasNative {
		return nil, nil, ErrNotNativeScheme
	}

	if !u.CheckPassword(password) {
		return nil, nil, ErrPasswordMismatch
	}

	ssn := &session.Session{
		UserID:   u.ID,
		Username: u.Username,
		State:    u.State,
		Scheme:   AUTH_NATIVE_SCHEME,
	}

	return u, ssn, nil
}

func (s *NativeScheme) Login(params map[string]string) (*user.User, *session.Session, *axerror.AXError) {
	username, ok := params["username"]
	if !ok {
		return nil, nil, ErrMissingUsername
	}
	password, ok := params["password"]
	if !ok {
		return nil, nil, ErrMissingPassword
	}
	u, ssn, axErr := s.Verify(username, password)
	if axErr != nil {
		return nil, nil, axErr
	}

	ssn, axErr = ssn.Create()
	if axErr != nil {
		return nil, nil, axErr
	}

	return u, ssn, axErr
}

func (s *NativeScheme) CreateRequest(data map[string]string) (*auth.AuthRequest, *axerror.AXError) {
	return &auth.AuthRequest{
		Scheme: "native",
	}, nil
}

func (s *NativeScheme) ChangePassword(u *user.User, old, new string) *axerror.AXError {
	if axErr := ValidateNative(u); axErr != nil {
		return axErr
	}

	if axErr := session.DeleteSessionsByUsername(u.Username); axErr != nil {
		return axErr
	}

	return u.ChangePassword(old, new)
}

func (s *NativeScheme) StartPasswordReset(u *user.User) *axerror.AXError {
	if axErr := ValidateNative(u); axErr != nil {
		return axErr
	}

	return u.StartPasswordReset()
}

func (s *NativeScheme) ResetPassword(u *user.User, new string) *axerror.AXError {
	if axErr := ValidateNative(u); axErr != nil {
		return axErr
	}

	if axErr := session.DeleteSessionsByUsername(u.Username); axErr != nil {
		return axErr
	}

	return u.ResetPassword(new, true)
}

func ValidateNative(u *user.User) *axerror.AXError {
	if u.AuthSchemes == nil {
		return ErrNotNativeNotSupported
	}
	for _, scheme := range u.AuthSchemes {
		if scheme == "native" {
			return nil
		}
	}
	return ErrNotNativeNotSupported
}
