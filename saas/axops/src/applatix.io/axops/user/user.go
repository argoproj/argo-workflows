// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/label"
	"applatix.io/axops/utils"
)

var (
	ErrNotAllowedOperation = axerror.ERR_API_INVALID_REQ.NewWithMessage("The opertion is not allowed.")
)

const RedisUserIdKey = "user-id-%v"
const RedisUserNameKey = "user-name-%v"

type User struct {
	ID              string        `json:"id,omitempty" description:"uuid"`
	Username        string        `json:"username,omitempty" description:"username(email)"`
	FirstName       string        `json:"first_name,omitempty"  description:"first name"`
	LastName        string        `json:"last_name,omitempty" description:"last name"`
	Password        string        `json:"password,omitempty" description:"password"`
	State           int           `json:"state,omitempty" description:"account state: 1-init,2-active,3-banned,4-deleted"`
	AuthSchemes     []string      `json:"auth_schemes"  description:"account schemes"`
	Groups          []string      `json:"groups" description:"account groups: admin,developer"`
	Settings        utils.MapType `json:"settings" description:"user settings"`
	ViewPreferences utils.MapType `json:"view_preferences" description:"view preferences"`
	Labels          []string      `json:"labels" description:"user labels"`
	Ctime           int64         `json:"ctime,omitempty" description:"creation time in seconds since epoch"`
	Mtime           int64         `json:"mtime,omitempty" description:"modification time in seconds since epoch"`
}

func (u *User) Update() *axerror.AXError {

	for _, group := range u.Groups {
		g := &Group{
			Name: group,
		}

		if axErr := g.Validate(); axErr != nil {
			return axErr
		}
	}

	for _, lb := range u.Labels {
		l := &label.Label{
			Type:  label.LabelTypeUser,
			Key:   lb,
			Value: "",
		}

		if axErr := l.Validate(); axErr != nil {
			return axErr
		}
	}

	// Update the group username list
	for _, group := range u.Groups {
		g := &Group{
			Name: group,
		}

		g, axErr := g.Reload()
		if axErr != nil {
			return axErr
		}

		g.Usernames = append(g.Usernames, u.Username)
		if axErr := g.Update(); axErr != nil {
			return axErr
		}
	}

	return u.SimpleUpdate()
}

func (u *User) SimpleUpdate() *axerror.AXError {

	if u.Settings == nil {
		u.Settings = map[string]string{}
	}

	if u.ViewPreferences == nil {
		u.ViewPreferences = map[string]string{}
	}

	u.Mtime = time.Now().Unix()

	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, UserTableName, u); axErr != nil {
		return axErr
	}

	UpdateETag()
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserIdKey, u.ID), u, time.Hour*2)
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserNameKey, u.Username), u, time.Hour*2)
	return nil
}

func (u *User) Reload() (*User, *axerror.AXError) {
	if u.Username == "" {
		return nil, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing user name.")
	}

	user, axErr := GetUserByName(u.Username)
	if axErr != nil {
		return nil, axErr
	}

	if user == nil {
		return nil, axerror.ERR_API_AUTH_FAILED.NewWithMessagef("Cannot find user with name: %v", u.Username)
	}

	return user, nil
}

func (u *User) Create() (*User, *axerror.AXError) {

	u.Username = strings.ToLower(u.Username)

	if !ValidateEmail(u.Username) {
		return nil, axerror.ERR_API_INVALID_USERNAME.New()
	}

	if user, axErr := GetUserByName(u.Username); axErr != nil {
		return nil, axErr
	} else {
		if user != nil && !(user.State == UserStateInit || user.State == UserStateDeleted) {
			return nil, axerror.ERR_API_DUP_USERNAME.NewWithMessagef("The user name %v has been taken.", u.Username)
		}
	}

	if len(u.AuthSchemes) == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing auth scheme for user.")
	}

	if len(u.Groups) == 0 {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Missing group for user.")
	}

	if u.State > UserStateBanned || u.State < UserStateDeleted {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Invalid user state: %v.", u.State)
	}

	// Native scheme password is managed locally
	if u.AuthSchemes[0] == "native" {

		if axErr := checkPasswordStrength(u.Password); axErr != nil {
			return nil, axErr
		}

		hash, hashErr := hashPassword(u.Password)
		if hashErr != nil {
			return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Fail to hash the password:%v", hashErr))
		}

		u.Password = hash
	} else {
		u.Password = " "
	}

	u.ID = utils.GenerateUUIDv1()
	if u.State == 0 {
		u.State = UserStateInit
	}

	u.Ctime = time.Now().Unix()
	u.Mtime = time.Now().Unix()

	if axErr := u.Update(); axErr != nil {
		return nil, axErr
	}

	if u.State == UserStateInit {
		if axErr := u.StartConfirmation(); axErr != nil {
			return nil, axErr
		}
	}

	return u, nil
}

func (u *User) Delete() *axerror.AXError {
	u.State = UserStateDeleted
	u.Password = " "
	u.AuthSchemes = []string{}
	for _, group := range u.Groups {
		g := &Group{
			Name: group,
		}

		g, axErr := g.Reload()
		if axErr != nil {
			if axErr.Code != axerror.ERR_API_RESOURCE_NOT_FOUND.Code {
				return axErr
			} else {
				continue
			}
		}

		usernames := []string{}
		for _, username := range g.Usernames {
			if username != u.Username {
				usernames = append(usernames, username)
			}
		}

		g.Usernames = usernames
		if axErr := g.Update(); axErr != nil {
			return axErr
		}
	}
	u.Groups = []string{}
	u.Labels = []string{}

	return u.Update()
}

func (u *User) Ban() *axerror.AXError {
	u.State = UserStateBanned
	return u.Update()
}

func (u *User) Active() *axerror.AXError {
	if u.State == UserStateDeleted {
		return axerror.ERR_API_INVALID_REQ.NewWithMessage("The user doesn't exist.")
	}
	u.State = UserStateActive
	return u.Update()
}

func (u *User) OmitPassword() {
	u.Password = ""
}

func (u *User) CheckPassword(password string) bool {
	return verifyPassword(u.Password, password)
}

func (u *User) ResetPassword(password string, checkStrength bool) *axerror.AXError {

	if err := u.setPassword(password, checkStrength); err != nil {
		return err
	}

	return nil
}

func (u *User) setPassword(password string, checkStrength bool) *axerror.AXError {

	if checkStrength {
		if axErr := checkPasswordStrength(password); axErr != nil {
			return axErr
		}
	}

	hash, hashErr := hashPassword(password)

	if hashErr != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessage(fmt.Sprintf("Fail to hash the password:%v", hashErr))
	}

	u.Password = hash

	if axErr := u.Update(); axErr != nil {
		return axErr
	}

	return nil
}

func (u *User) ChangePassword(old, new string) *axerror.AXError {
	user, axErr := u.Reload()
	if axErr != nil {
		return axErr
	}
	axErr = user.verifyPasswordUpdateAllowed()
	if axErr != nil {
		return axErr
	}
	if !user.CheckPassword(old) {
		return axerror.ERR_API_AUTH_FAILED.NewWithMessage("The old password is wrong.")
	}
	return user.setPassword(new, true)
}

// verifyPasswordUpdateAllowed checks if we allow password changes for this user.
// Users who are portal users, or users who auth scheme is SAML only, are disallowed
func (u *User) verifyPasswordUpdateAllowed() *axerror.AXError {
	// Would prefer to use constant saml.AUTH_SAML_SCHEME, but can't because of import cycle
	if len(u.AuthSchemes) == 1 && u.AuthSchemes[0] == "saml" {
		return axerror.ERR_API_INVALID_REQ.NewWithMessage(fmt.Sprintf("Password must be updated through identity provider"))
	}
	return nil
}

func (u *User) StartPasswordReset() *axerror.AXError {
	axErr := u.verifyPasswordUpdateAllowed()
	if axErr != nil {
		return axErr
	}
	// create and persist the reset password link
	r := &SystemRequest{
		UserID:   u.ID,
		Username: u.Username,
		Target:   u.Username,
		Type:     SysReqPassReset,
	}

	if r, axErr = r.Create(DEFAULT_REQUEST_DURATION); axErr != nil {
		return axErr
	}
	// send the reset password email synchronously
	if axErr = r.SendRequest(); axErr != nil {
		return axErr
	}

	return nil
}

func (u *User) StartConfirmation() *axerror.AXError {
	var axErr *axerror.AXError

	if u.State != UserStateInit {
		return axerror.ERR_API_INVALID_REQ.NewWithMessage("The user is not in initial state.")
	}

	// create and persist the activation link
	r := &SystemRequest{
		UserID:   u.ID,
		Username: u.Username,
		Target:   u.Username,
		Type:     SysReqUserConfirm,
	}

	if r, axErr = r.Create(DEFAULT_REQUEST_DURATION); axErr != nil {
		return axErr
	}

	// send the activation email synchronously
	if axErr = r.SendRequest(); axErr != nil {
		return axErr
	}

	return nil
}

func (u *User) HasGroup(name string) bool {
	for _, g := range u.Groups {
		if g == name {
			return true
		}
	}
	return false
}

func (u *User) IsAdmin() bool {
	return u.HasGroup(GroupAdmin)
}

func (u *User) IsSuperAdmin() bool {
	return u.HasGroup(GroupSuperAdmin)
}

func (u *User) IsDeveloper() bool {
	return u.HasGroup(GroupDeveloper)
}

func (u *User) OnlyViewPreferenceChanged(e *User) bool {
	if e == nil {
		return false
	}

	if u.FirstName != e.FirstName {
		return false
	}

	if u.LastName != e.LastName {
		return false
	}

	if len(u.Labels) != len(e.Labels) {
		return false
	}
	sort.Sort(sort.StringSlice(u.Labels))
	sort.Sort(sort.StringSlice(e.Labels))
	for i, _ := range e.Labels {
		if e.Labels[i] != u.Labels[i] {
			return false
		}
	}

	if len(u.Settings) != len(e.Settings) {
		return false
	}
	for k, _ := range e.Settings {
		if e.Settings[k] != u.Settings[k] {
			return false
		}
	}

	return true
}

func GetUserById(id string) (*User, *axerror.AXError) {
	user := &User{}
	if axErr := utils.RedisCacheCl.GetObj(fmt.Sprintf(RedisUserIdKey, id), user); axErr == nil {
		utils.DebugLog.Printf("[Cache] cache hit for user with id %v\n", id)
		if user.State == UserStateDeleted {
			return nil, nil
		}
		return user, nil
	}

	users, axErr := GetUsers(map[string]interface{}{
		UserID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(users) == 0 || users[0].State == UserStateDeleted {
		return nil, nil
	}

	user = &users[0]
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserIdKey, id), user, time.Hour*2)
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserNameKey, user.Username), user, time.Hour*2)
	return user, nil
}

func GetUserByName(name string) (*User, *axerror.AXError) {

	name = strings.ToLower(name)

	user := &User{}
	if axErr := utils.RedisCacheCl.GetObj(fmt.Sprintf(RedisUserNameKey, name), user); axErr == nil {
		utils.DebugLog.Printf("[Cache] cache hit for user with name %v\n", name)
		if user.State == UserStateDeleted {
			return nil, nil
		}
		return user, nil
	}

	users, axErr := GetUsers(map[string]interface{}{
		UserName: name,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(users) == 0 || users[0].State == UserStateDeleted {
		return nil, nil
	}

	user = &users[0]

	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserIdKey, user.ID), user, time.Hour*2)
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisUserNameKey, user.Username), user, time.Hour*2)
	return user, nil
}

func GetUsersByGroup(group string) ([]User, *axerror.AXError) {

	users, axErr := GetUsers(map[string]interface{}{
		UserGroups: group,
	})

	if axErr != nil {
		return nil, axErr
	}

	return users, nil
}

func GetUserMapByGroup(group string) (map[string]User, *axerror.AXError) {

	users, axErr := GetUsers(map[string]interface{}{
		UserGroups: group,
	})

	if axErr != nil {
		return nil, axErr
	}

	userMap := map[string]User{}
	for i, _ := range users {
		userMap[users[i].Username] = users[i]
	}

	return userMap, nil
}

func GetUsersByLabel(label string) ([]User, *axerror.AXError) {
	users, axErr := GetUsers(map[string]interface{}{
		UserLabels: label,
	})

	if axErr != nil {
		return nil, axErr
	}

	return users, nil
}

func GetUsers(params map[string]interface{}) ([]User, *axerror.AXError) {
	users := []User{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, UserTableName, params, &users)
	if axErr != nil {
		return nil, axErr
	}

	noDeleted := []User{}
	for i, _ := range users {
		if users[i].State != UserStateDeleted {
			noDeleted = append(noDeleted, users[i])
		}
	}

	return noDeleted, nil
}
