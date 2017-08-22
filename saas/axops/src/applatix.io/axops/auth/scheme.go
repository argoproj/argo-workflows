// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package auth

import (
	"applatix.io/axerror"
	"applatix.io/axops/session"
	"applatix.io/axops/user"
)

var (
	ErrUserBanned       = axerror.ERR_API_AUTH_FAILED.NewWithMessage("The user account is in banned state.")
	ErrUserNotConfirmed = axerror.ERR_API_ACCOUNT_NOT_CONFIRMED.NewWithMessage("The user account is not activated, please check your confirmation email to active your account.")
)

type SchemeSummary struct {
	Name        string `json:"name"`
	ButtonLabel string `json:"button_label"`
	Enabled     bool   `json:"enabled"`
}

type Scheme interface {
	Name() string
	Login(params map[string]string) (*user.User, *session.Session, *axerror.AXError)
	Logout(s *session.Session) *axerror.AXError
	Auth(token string) (*user.User, *session.Session, *axerror.AXError)
	CreateUser(u *user.User) (*user.User, *axerror.AXError)
	DeleteUser(u *user.User) *axerror.AXError
	BanUser(u *user.User) *axerror.AXError
	ActiveUser(u *user.User) *axerror.AXError
	Metadata() (string, *axerror.AXError)
	CreateRequest(data map[string]string) (*AuthRequest, *axerror.AXError)
	Scheme() map[string]interface{}
}

type ManagedScheme interface {
	Scheme
	ChangePassword(u *user.User, old, new string) *axerror.AXError
	StartPasswordReset(u *user.User) *axerror.AXError
	ResetPassword(u *user.User, new string) *axerror.AXError
}

var schemes = make(map[string]Scheme)

func RegisterScheme(name string, scheme Scheme) {
	schemes[name] = scheme
}

func UnregisterScheme(name string) {
	delete(schemes, name)
}

func GetScheme(name string) (Scheme, *axerror.AXError) {
	scheme, ok := schemes[name]
	if !ok {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessagef("Unknown authentication scheme: %v. Please check your authentication configuration.", name)
	}
	return scheme, nil
}

func GetAllSchemes() ([]Scheme, *axerror.AXError) {
	all := []Scheme{}
	for _, scheme := range schemes {
		all = append(all, scheme)
	}
	return all, nil
}

type BaseScheme struct {
}

func (s *BaseScheme) Auth(token string) (*user.User, *session.Session, *axerror.AXError) {
	var axErr *axerror.AXError
	var u *user.User
	ssn := &session.Session{
		ID: token,
	}

	if ssn, axErr = ssn.Reload(); axErr != nil {
		return nil, nil, axErr
	}

	if axErr = ssn.Validate(); axErr != nil {
		return nil, nil, axErr
	}

	if axErr = ssn.Extend(); axErr != nil {
		return nil, nil, axErr
	}

	if u, axErr = user.GetUserById(ssn.UserID); axErr != nil {
		return nil, nil, axErr
	}

	if u == nil || ssn == nil {
		return nil, nil, axerror.ERR_API_AUTH_FAILED.NewWithMessage("Invalid session.")
	}

	if u.State == user.UserStateBanned {
		return nil, nil, ErrUserBanned
	}

	if u.State == user.UserStateInit {
		return nil, nil, ErrUserNotConfirmed
	}

	return u, ssn, nil
}

func (s *BaseScheme) Logout(ssn *session.Session) *axerror.AXError {
	if axErr := ssn.Delete(); axErr != nil {
		return axErr
	}
	return nil
}

func (s *BaseScheme) CreateUser(u *user.User) (*user.User, *axerror.AXError) {

	if u.HasGroup(user.GroupSuperAdmin) {
		return nil, user.ErrNotAllowedOperation
	}

	return u.Create()
}

func (s *BaseScheme) DeleteUser(u *user.User) *axerror.AXError {

	if u.HasGroup(user.GroupSuperAdmin) {
		return user.ErrNotAllowedOperation
	}

	if axErr := session.DeleteSessionsByUsername(u.Username); axErr != nil {
		return axErr
	}
	return u.Delete()
}

func (s *BaseScheme) BanUser(u *user.User) *axerror.AXError {

	if u.HasGroup(user.GroupSuperAdmin) {
		return user.ErrNotAllowedOperation
	}

	if axErr := session.DeleteSessionsByUsername(u.Username); axErr != nil {
		return axErr
	}
	return u.Ban()
}

func (s *BaseScheme) ActiveUser(u *user.User) *axerror.AXError {

	if u.HasGroup(user.GroupSuperAdmin) {
		return user.ErrNotAllowedOperation
	}

	return u.Active()
}

func (s *BaseScheme) Metadata() (string, *axerror.AXError) {
	return "", nil
}
