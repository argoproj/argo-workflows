// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi Authentication API [/auth]
package axops

import (
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/auth"
	"applatix.io/axops/auth/native"
	"applatix.io/axops/cluster"
	"applatix.io/axops/session"
	"applatix.io/axops/tool"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

type LoginCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInfo struct {
	SessionId string `json:"session"`
	UserId    string `json:"user_id"`
}

// @Title Login
// @Description Login with credentials
// @Accept  json
// @Param   credentials	body    LoginCredential	true	"Login credential."
// @Success 200 {object} LoginInfo
// @Failure 400 {object} axerror.AXError "Invalid request body"
// @Failure 401 {object} axerror.AXError "Unauthorized"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /auth
// @Router /auth/login [POST]
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		jsonMap := map[string]string{}
		err := utils.GetUnmarshalledBody(c, &jsonMap)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {

			if u, ssn, axErr := scheme.Login(jsonMap); axErr != nil {
				c.JSON(axerror.REST_AUTH_DENIED, axErr)
				return
			} else {
				c.SetCookie(session.COOKIE_SESSION_TOKEN, ssn.ID, int(session.SESSION_RETENTION_SEC), "", "", false, false)
				resultMap := LoginInfo{
					SessionId: ssn.ID,
					UserId:    u.ID,
				}
				c.JSON(axerror.REST_STATUS_OK, resultMap)
				return
			}
		}
	}
}

func Auth(internal bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if internal {
			// Create fake Session and Session User information
			SetContextUser(c, &user.User{
				ID:          "00000000-0000-0000-0000-000000000000",
				Username:    "system",
				State:       user.UserStateActive,
				AuthSchemes: []string{"native"},
				Groups:      []string{user.GroupAdmin},
			})

			SetContextSession(c, &session.Session{
				ID:       "00000000-0000-0000-0000-000000000000",
				UserID:   "00000000-0000-0000-0000-000000000000",
				Username: "system",
				State:    user.UserStateActive,
				Scheme:   "native",
				Ctime:    time.Now().Unix(),
				Expiry:   time.Now().Add(time.Hour).Unix(),
			})
			return
		}

		//utils.DebugLog.Printf("[AUTH] Cluster Settings: %v\n", cluster.ClusterSettings)
		if cluster.ClusterSettings[cluster.PublicReadEnabledKey] == "true" {
			utils.DebugLog.Printf("[AUTH] Request %v %v\n", c.Request.Method, c.Request.RequestURI)
			if c.Request.Method == "GET" {
				if strings.HasPrefix(c.Request.RequestURI, "/v1/service/events") ||
					(strings.HasPrefix(c.Request.RequestURI, "/v1/services") && !strings.Contains(c.Request.RequestURI, "/exec")) ||
					strings.HasPrefix(c.Request.RequestURI, "/v1/commits") ||
					strings.HasPrefix(c.Request.RequestURI, "/v1/branches") ||
					strings.HasPrefix(c.Request.RequestURI, "/v1/repos") {

					// Create fake Session and Session User information for anonymous users
					SetContextUser(c, &user.User{
						ID:          "00000000-0000-0000-0000-000000000002",
						Username:    "anonymous",
						State:       user.UserStateActive,
						AuthSchemes: []string{"native"},
						Groups:      []string{user.GroupDeveloper},
					})

					SetContextSession(c, &session.Session{
						ID:       "00000000-0000-0000-0000-000000000002",
						UserID:   "00000000-0000-0000-0000-000000000002",
						Username: "system",
						State:    user.UserStateActive,
						Scheme:   "native",
						Ctime:    time.Now().Unix(),
						Expiry:   time.Now().Add(time.Hour).Unix(),
					})

					c.Next()
					return
				}
			}
		}
		scheme, axErr := auth.GetScheme("native")
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		// Session token authentication
		token, _ := c.Cookie(session.COOKIE_SESSION_TOKEN)
		if token != "" {
			u, ssn, axErr := scheme.Auth(token)
			if axErr != nil {
				c.SetCookie(session.COOKIE_SESSION_TOKEN, "", -1000000, "", "", false, false)
				c.JSON(axerror.REST_AUTH_DENIED, axErr)
				c.Abort()
				return
			}

			SetContextSession(c, ssn)
			SetContextUser(c, u)
			c.SetCookie(session.COOKIE_SESSION_TOKEN, ssn.ID, int(session.SESSION_RETENTION_SEC), "", "", false, false)
			return
		}

		// Basic Auth
		if username, password, ok := c.Request.BasicAuth(); ok {
			native := scheme.(*native.NativeScheme)
			u, ssn, axErr := native.Verify(username, password)
			if axErr != nil {
				c.JSON(axerror.REST_AUTH_DENIED, axErr)
				return
			}
			SetContextSession(c, ssn)
			SetContextUser(c, u)
			return
		}

		c.JSON(axerror.REST_AUTH_DENIED, axerror.ERR_API_INVALID_SESSION.NewWithMessage("Session information is not found."))
		c.Abort()
		return
	}
}

// @Title Logout
// @Description Logout
// @Accept  json
// @Success 200 {object} MapType
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /auth
// @Router /auth/logout [POST]
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := GetContextSession(c)

		if s == nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Missing session information."))
			return
		}

		if scheme, axErr := auth.GetScheme("native"); axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			if axErr = scheme.Logout(s); axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
		}

		c.SetCookie(session.COOKIE_SESSION_TOKEN, "", -1000000, "", "", false, false)
		c.JSON(axdb.RestStatusOK, utils.NullMap)
	}
}

type AuthSchemeData struct {
	Data []auth.SchemeSummary `json:"data"`
}

// @Title GetAuthSchemes
// @Description List auth schemes supported
// @Accept  json
// @Success 200 {object} AuthSchemeData
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /auth
// @Router /auth/schemes [GET]
func GetSchemes() gin.HandlerFunc {
	return func(c *gin.Context) {
		schemes, axErr := auth.GetAllSchemes()
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		summaries := []map[string]interface{}{}
		for _, scheme := range schemes {
			summary := scheme.Scheme()
			summaries = append(summaries, summary)
		}

		resultMap := map[string]interface{}{
			RestData: summaries,
		}
		c.JSON(axerror.REST_STATUS_OK, resultMap)
	}
}

func SAMLMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		if scheme, axErr := auth.GetScheme("saml"); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		} else {
			metadata, axErr := scheme.Metadata()
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
			c.XML(axerror.REST_STATUS_OK, metadata)
			return
		}
	}
}

// @Title GetSAMLRequest
// @Description Get SAML request
// @Accept  json
// @Param   redirect_url  	 query   string     false       "Redirect URL."
// @Success 200 {object} auth.AuthRequest
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /auth
// @Router /auth/saml/request [GET]
func SAMLRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		if scheme, axErr := auth.GetScheme("saml"); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			c.Redirect(302, "https://")
			return
		} else {
			data := map[string]string{}
			url := c.Request.URL.Query().Get("redirect_url")
			if url != "" {
				data["redirect_url"] = url
			}
			request, axErr := scheme.CreateRequest(data)
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
				return
			}
			c.JSON(axerror.REST_STATUS_OK, request)
			return
		}
	}
}

type SamlInfo struct {
	EntityID       string `json:"entity_id"`
	SsoCallbackUrl string `json:"sso_callback_url"`
	PublicCert     string `json:"public_cert"`
}

// @Title GetSAMLInfo
// @Description Get SAML information
// @Accept  json
// @Success 200 {object} SamlInfo
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /auth
// @Router /auth/saml/info [GET]
func SAMLInfo() gin.HandlerFunc {
	return func(c *gin.Context) {

		info := SamlInfo{
			EntityID:       utils.GetEntityID(),
			SsoCallbackUrl: utils.GetSSOURL(),
		}

		certs, axErr := tool.GetToolsByType(tool.TypeServer)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if len(certs) == 0 {
			if axErr != nil {
				c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("The system certificate is missing."))
				return
			}
		} else {
			info.PublicCert = certs[0].(*tool.ServerCertConfig).PublicCert
		}

		c.JSON(axerror.REST_STATUS_OK, info)
		return
	}
}

func SAMLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if scheme, axErr := auth.GetScheme("saml"); axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		} else {
			response, _ := c.GetPostForm("SAMLResponse")
			params := map[string]string{
				"response": response,
			}

			_, ssn, axErr := scheme.Login(params)
			if axErr != nil {
				c.Redirect(302, utils.GetErrorURL(axerror.REST_AUTH_DENIED, axErr))
				return
			}

			c.SetCookie(session.COOKIE_SESSION_TOKEN, ssn.ID, int(session.SESSION_RETENTION_SEC), "", "", false, false)

			if url, ok := params["redirect_url"]; ok {
				c.Redirect(302, url)
			} else {
				c.Redirect(302, "https://"+common.GetPublicDNS()+"/#/app/commits/overview")
			}
			return
		}
	}
}

func MustGroupAdmins() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := GetContextUser(c)
		if !u.IsSuperAdmin() && !u.IsAdmin() {
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_AUTH_PERMISSION_DENIED.NewWithMessage("You don't have enough privilege to perform this operation."))
			c.Abort()
			return
		}
	}
}

func MustSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := GetContextUser(c)
		if !u.IsSuperAdmin() {
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_AUTH_PERMISSION_DENIED.NewWithMessage("You don't have enough privilege to perform this operation."))
			c.Abort()
			return
		}
	}
}

func MustSelf() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		u := GetContextUser(c)
		if username != u.Username {
			c.JSON(axerror.REST_FORBIDDEN, axerror.ERR_API_AUTH_PERMISSION_DENIED.NewWithMessage("You don't have enough privilege to perform this operation."))
			c.Abort()
			return
		}
	}
}

func SetContextUser(c *gin.Context, u *user.User) {
	c.Set("user", u)
}

func SetContextSession(c *gin.Context, s *session.Session) {
	c.Set("session", s)
}

func GetContextUser(c *gin.Context) *user.User {
	v, _ := c.Get("user")
	u, _ := v.(*user.User)
	return u
}

func GetContextSession(c *gin.Context) *session.Session {
	v, _ := c.Get("session")
	s, _ := v.(*session.Session)
	return s
}
