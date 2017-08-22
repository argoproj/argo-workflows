// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi User API [/users]
package axops

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/auth"
	"applatix.io/axops/sandbox"
	"applatix.io/axops/session"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

var (
	ErrChangeOthersPasswordNotAllowed = axerror.ERR_API_FORBIDDEN_REQ.NewWithMessage("It is not allowed to change other's password.")
	ErrInvalidOrExpiredLink           = axerror.ERR_API_INVALID_REQ.NewWithMessage("The link is expired or invalid.")
)

type UsersData struct {
	Data []user.User `json:"data"`
}

// @Title ListUsers
// @Description List users
// @Accept  json
// @Param   username  	 query   string     false       "Username."
// @Param   first_name	 query   string     false       "First Name."
// @Param   last_name	 query   string     false       "Last Name."
// @Param   state	 query   int        false       "State."
// @Param   search	 query   string     false       "Search."
// @Success 200 {object} UsersData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users [GET]
func ListUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && user.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		params, axErr := GetContextParams(c,
			[]string{
				user.UserFirstName,
				user.UserLastName,
				user.UserName,
			},
			[]string{},
			[]string{user.UserState},
			[]string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		if users, axErr := user.GetUsers(params); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			newUsers := []user.User{}
			loggedInUser := GetContextUser(c)
			for i, _ := range users {
				users[i].OmitPassword()
				username := users[i].Username
				// in sandbox mode, developer can't see Username and Lastname
				if sandbox.IsSandboxEnabled() && loggedInUser != nil && loggedInUser.IsDeveloper() {
					users[i].LastName = ""
					users[i].Username = ""
				}
				if username != "admin@internal" {
					newUsers = append(newUsers, users[i])
				}
			}

			usersData := UsersData{
				Data: newUsers,
			}

			c.Header("ETag", user.GetETag())

			c.JSON(axerror.REST_STATUS_OK, usersData)
		}
	}
}

// @Title GetUser
// @Description Get user by username
// @Accept  json
// @Param   username     path    string     true        "User Name(Email)"
// @Success 200 {object} user.User
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username} [GET]
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if username == "session" {
			GetSessionUser()(c)
			return
		}

		if etag := c.Request.Header.Get("If-None-Match"); len(etag) > 0 && user.GetETag() == etag {
			c.Status(http.StatusNotModified)
			return
		}

		if u, axErr := user.GetUserByName(username); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if u != nil {
				u.OmitPassword()
				c.Header("ETag", user.GetETag())
				c.JSON(axdb.RestStatusOK, u)
				return
			} else {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
				return
			}
		}
	}
}

func GetSessionUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := GetContextUser(c)
		u.OmitPassword()
		c.JSON(axdb.RestStatusOK, u)
		return
	}
}

// @Title UpdateUser
// @Description Get user by username
// @Accept  json
// @Param   username     path    string     true        "User Name(Email)"
// @Param   user         body    user.User  true        "User object"
// @Success 200 {object} user.User
// @Failure 400 {object} axerror.AXError "Bad request body"
// @Failure 404 {object} axerror.AXError "Resource not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username} [PUT]
func PutUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		if exUser, axErr := user.GetUserByName(username); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if exUser != nil {
				u := user.User{}
				err := utils.GetUnmarshalledBody(c, &u)
				if err != nil {
					c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
					return
				}

				ctxUser := GetContextUser(c)

				clearSession := false
				viewChangeOnly := false

				// self can update names, labels and settings
				if ctxUser.Username == username {
					viewChangeOnly = exUser.OnlyViewPreferenceChanged(&u)

					if u.FirstName != "" {
						exUser.FirstName = u.FirstName
					}

					if u.LastName != "" {
						exUser.LastName = u.LastName
					}

					if len(u.Settings) != 0 {
						exUser.Settings = u.Settings
					}

					if len(u.ViewPreferences) != 0 {
						exUser.ViewPreferences = u.ViewPreferences
					}

					if len(u.Labels) != 0 {
						exUser.Labels = u.Labels
					}
				} else {
					if exUser.IsSuperAdmin() {
						c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
						return
					}

					if ctxUser.IsDeveloper() {
						c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
						return
					}

					// admins can do anything for non super admins except for making another person super admin
					if (ctxUser.IsAdmin() || ctxUser.IsSuperAdmin()) && !exUser.IsSuperAdmin() {

						if u.FirstName != "" {
							exUser.FirstName = u.FirstName
						}

						if u.LastName != "" {
							exUser.LastName = u.LastName
						}

						if len(u.Settings) != 0 {
							exUser.Settings = u.Settings
						}

						if len(u.ViewPreferences) != 0 {
							exUser.ViewPreferences = u.ViewPreferences
						}

						if len(u.Labels) != 0 {
							exUser.Labels = u.Labels
						}

						// can not promote anyone to super admin
						if u.HasGroup(user.GroupSuperAdmin) {
							c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
							return
						}

						exUser.State = u.State

						// clear the user sessions when group is altered
						if len(exUser.Groups) != len(u.Groups) {
							clearSession = true
						} else {
							sort.Sort(sort.StringSlice(exUser.Groups))
							sort.Sort(sort.StringSlice(u.Groups))
							for i, _ := range exUser.Groups {
								if exUser.Groups[i] != u.Groups[i] {
									clearSession = true
									break
								}
							}
						}

						exUser.Groups = u.Groups
					}
				}

				if viewChangeOnly {
					if axErr := exUser.SimpleUpdate(); axErr != nil {
						c.JSON(axerror.REST_INTERNAL_ERR, axErr)
						return
					}

				} else {
					if axErr := exUser.Update(); axErr != nil {
						c.JSON(axerror.REST_INTERNAL_ERR, axErr)
						return
					}
				}

				if clearSession {
					if axErr := session.DeleteSessionsByUsername(username); axErr != nil {
						c.JSON(axerror.REST_INTERNAL_ERR, axErr)
						return
					}
				}

				exUser.OmitPassword()
				c.JSON(axdb.RestStatusOK, exUser)
				return
			} else {
				c.JSON(axdb.RestStatusNotFound, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
				return
			}
		}
	}
}

// @Title RegisterUser
// @Description Register user
// @Accept  json
// @Param   username     path    string     	true       "User name(email) to register"
// @Param   token        path    string     	true       "Token"
// @Param   user         body    user.User	true        "User object"
// @Success 201 {object} user.User
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/register/{token} [POST]
func CreateUserWithToken() gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.Param("token")

		if token == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing token."))
			return
		}

		r, axErr := user.GetSysReqById(token)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if r == nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		if axErr = r.Validate(); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		var u *user.User
		err := utils.GetUnmarshalledBody(c, &u)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		// populate group from the invitation request
		group := r.Data["group"]
		if group == user.GroupSuperAdmin {
			c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
			return
		}

		isSingleUser := r.Data["singleUser"] == "true"

		u.Groups = []string{group}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			u.AuthSchemes = []string{"native"}
			if isSingleUser {
				u.Username = r.Target
				u.FirstName = r.Data["firstName"]
				u.LastName = r.Data["lastName"]
				u.State = user.UserStateActive
			} else {
				u.State = user.UserStateInit
			}
			if u, axErr = scheme.CreateUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}
		u.OmitPassword()
		c.JSON(axerror.REST_CREATE_OK, u)
		return
	}
}

// @Title CreateUser
// @Description Create user account
// @Accept  json
// @Param   user         body    user.User  true        "User object"
// @Success 201 {object} user.User
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users [POST]
func AdminCreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var u *user.User
		err := utils.GetUnmarshalledBody(c, &u)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			u.AuthSchemes = []string{"native"}
			if u, axErr = scheme.CreateUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}
		u.OmitPassword()
		c.JSON(axerror.REST_CREATE_OK, u)
		return
	}
}

type ChangePasswordBody struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// @Title ChangePassword
// @Description Change password for user.
// @Accept  json
// @Produce json
// @Param   username     path    string     	true       "User name(email) to reset password"
// @Param   payload      body    ChangePasswordBody   true       "Password payload"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 403 {object} axerror.AXError "Permission denied"
// @Failure 404 {object} axerror.AXError "User not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/change_password [PUT]
func ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var jsonMap map[string]string
		err := utils.GetUnmarshalledBody(c, &jsonMap)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		username := c.Param("username")
		ssnUser := GetContextUser(c)
		if ssnUser.Username != username {
			c.JSON(axerror.REST_FORBIDDEN, ErrChangeOthersPasswordNotAllowed)
			return
		}

		scheme, axErr := auth.GetScheme("native")
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		oldPass, _ := jsonMap["old_password"]
		newPass, _ := jsonMap["new_password"]
		confirmPass, _ := jsonMap["confirm_password"]
		if newPass != confirmPass {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("The provided new passwords doesn't match."))
			return
		}

		native := scheme.(auth.ManagedScheme)
		if axErr = native.ChangePassword(ssnUser, oldPass, newPass); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title ForgetPassword
// @Description Forget password for user.
// @Accept  json
// @Produce json
// @Param   username     path    string     	true       "User name(email) to start reset password process"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "User not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/forget_password [POST]
func ForgetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			native := scheme.(auth.ManagedScheme)
			if axErr = native.StartPasswordReset(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

type ResetPasswordBody struct {
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// @Title ResetPassword
// @Description Reset password for user.
// @Accept  json
// @Produce json
// @Param   username     path    string     	true       "User name(email) to reset password"
// @Param   token        path    string     	true       "Token"
// @Param   payload      body    ResetPasswordBody   true       "Password payload"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "User not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/reset_password/{token} [PUT]
func ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var jsonMap map[string]string
		err := utils.GetUnmarshalledBody(c, &jsonMap)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		token := c.Param("token")

		if token == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing token."))
			return
		}

		r, axErr := user.GetSysReqById(token)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if r == nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		if axErr = r.Validate(); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		u, axErr := user.GetUserByName(r.Target)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		username := c.Param("username")
		if username != r.Target {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			newPass, _ := jsonMap["new_password"]
			confirmPass, _ := jsonMap["confirm_password"]

			if newPass != confirmPass {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("The provided new passwords doesn't match."))
				return
			}

			native := scheme.(auth.ManagedScheme)
			if axErr = native.ResetPassword(u, newPass); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		r.Delete()
		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title BanUser
// @Description Ban user by username
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to ban."
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "User not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/ban [PUT]
func AdminBanUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		ssnU := GetContextUser(c)

		if username == ssnU.Username {
			c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
			return
		}

		if username == "admin@internal" {
			c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if axErr = scheme.BanUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title ActivateUser
// @Description Activate user by username
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to unban."
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "User not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/activate [PUT]
func AdminActivateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if axErr = scheme.ActiveUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title DeleteUser
// @Description Delete user by username
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to delete."
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username} [DELETE]
func AdminDeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil || u.State == user.UserStateDeleted {
			c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
			return
		}

		ssnU := GetContextUser(c)

		if username == ssnU.Username {
			c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if axErr = scheme.DeleteUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title ResendUserConfirmation
// @Description Resend user confirmation email
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to resend confirmation email."
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 404 {object} axerror.AXError "User is not found"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/resend_confirm [POST]
func ResendUserConfirm() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_NOT_FOUND, axerror.ERR_API_RESOURCE_NOT_FOUND.New())
			return
		}

		axErr = u.StartConfirmation()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}

// @Title ConfirmUser
// @Description Confirm user
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to invite."
// @Param   token  	 path    string     true       "User confirmation token."
// @Success 302 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/confirm/{token} [GET]
func ConfirmUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")

		if token == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing token."))
			return
		}

		r, axErr := user.GetSysReqById(token)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if r == nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		if axErr = r.Validate(); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		u, axErr := user.GetUserByName(r.Target)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, ErrInvalidOrExpiredLink)
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if axErr = scheme.ActiveUser(u); axErr != nil {
				c.JSON(axerror.REST_BAD_REQ, axErr)
				return
			}
		}

		r.Delete()
		c.Redirect(302, "https://"+common.GetPublicDNS())
		return
	}
}

// @Title InviteUser
// @Description Invite user
// @Accept  json
// @Produce json
// @Param   username     path    string     true       "User name(email) to invite"
// @Param   group  	 query   string     true       "Group the new user to invite to join"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /users
// @Router /users/{username}/invite [POST]
func AdminInviteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		firstName := c.Request.URL.Query().Get("first_name")
		lastName := c.Request.URL.Query().Get("last_name")
		group := c.Request.URL.Query().Get("group")
		isSingleUser := c.Request.URL.Query().Get("single_user") == "true"

		if !user.ValidateEmail(username) {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("The username %s to invite is not a valid email address.", username))
			return
		}

		if group == "" {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Missing group information."))
			return
		}

		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if u != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("The user account already exists with the email address %v.", username))
			return
		}

		g, axErr := user.GetGroupByName(group)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if g == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Invalid group name: %s.", group))
			return
		}

		if g.Name == user.GroupSuperAdmin {
			c.JSON(axerror.REST_FORBIDDEN, user.ErrNotAllowedOperation)
			return
		}

		ctxUser := GetContextUser(c)

		// create and persist the invitation link
		r := &user.SystemRequest{
			UserID:   ctxUser.ID,
			Username: ctxUser.Username,
			Target:   username,
			Type:     user.SysReqUserInvite,
			Data: map[string]string{
				"group":      group,
				"firstName":  firstName,
				"lastName":   lastName,
				"singleUser": strconv.FormatBool(isSingleUser),
				"sandbox":    strconv.FormatBool(sandbox.IsSandboxEnabled()),
			},
		}

		if r, axErr = r.Create(2 * 7 * 24 * time.Hour); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		// send the invitation email synchronously
		if axErr = r.SendRequest(); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		c.JSON(axerror.REST_STATUS_OK, utils.NullMap)
		return
	}
}
