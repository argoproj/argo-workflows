// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"applatix.io/axerror"
	"applatix.io/axops/session"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gzip"
)

func GetScmWebHookRounter() *gin.Engine {
	router := gin.Default()
	router.GET("ping", func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, "pong")
	})

	router.Use(NoCache)
	router.Use(getGzipHandler())

	v1 := router.Group("v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(axerror.REST_STATUS_OK, "pong")
		})

		v1.POST("webhooks/scm", GatewaySCMWebhook())
	}

	return router
}

func GetJiraWebHookRounter() *gin.Engine {
	router := gin.Default()
	router.GET("ping", func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, "pong")
	})

	router.Use(NoCache)
	router.Use(getGzipHandler())

	v1 := router.Group("v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(axerror.REST_STATUS_OK, "pong")
		})
		v1.POST("webhooks/jira", GatewayJiraWebhook())
	}

	return router
}

func GetRounter(internal bool) *gin.Engine {
	router := gin.Default()
	router.GET("ping", func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, "pong")
	})

	router.Use(NoCache)
	router.Use(getGzipHandler())

	v1 := router.Group("v1")
	{
		v1.GET("ping", func(c *gin.Context) {
			c.JSON(axerror.REST_STATUS_OK, "pong")
		})

		v1.POST("webhooks/scm", GatewaySCMWebhook())
		v1.POST("webhooks/jira", GatewayJiraWebhook())
		v1.GET("results/:id/approval", GatewayApproval())

		auth := v1.Group("auth")
		{
			auth.POST("login", Login())
			auth.GET("schemes", GetSchemes())

			saml := auth.Group("saml")
			{
				saml.GET("metadata", SAMLMetadata())
				saml.GET("request", SAMLRequest())
				saml.POST("consume", SAMLogin())
				saml.GET("info", SAMLInfo())
			}

			auth.Use(Auth(internal))
			{
				auth.POST("logout", Logout())
			}
		}

		v1.POST("users/:username/register/:token", CreateUserWithToken())
		v1.GET("users/:username/confirm/:token", ConfirmUser())
		v1.POST("users/:username/forget_password", ForgetPassword())
		v1.PUT("users/:username/reset_password/:token", ResetPassword())
		v1.POST("users/:username/resend_confirm", ResendUserConfirm())

		v1.GET("system/version", GetSystemVersion())

		// Session validation
		v1.Use(Auth(internal))
		v1.Use(AddProfiler(v1.BasePath()))
		v1.GET("logout", Logout())
		v1.POST("logout", Logout())

		users := v1.Group("users")
		{
			users.GET("", ListUsers())
			users.GET("/:username", GetUser())
			users.PUT("/:username", PutUser())

			userSelf := users.Group("")
			userSelf.Use(MustSelf())
			{
				userSelf.PUT("/:username/change_password", ChangePassword())
			}

			userAdmin := users.Group("")
			userAdmin.Use(MustGroupAdmins())
			{
				userAdmin.POST("", AdminCreateUser())
				userAdmin.PUT("/:username/ban", AdminBanUser())
				userAdmin.PUT("/:username/activate", AdminActivateUser())
				userAdmin.POST("/:username/invite", AdminInviteUser())
				userAdmin.DELETE("/:username", AdminDeleteUser())
			}
		}

		requests := v1.Group("system_requests")
		{
			requests.Use(MustGroupAdmins())
			requests.GET("", GetSystemRequests())
		}

		labels := v1.Group("labels")
		{
			labels.GET("", ListLabels())
			labels.GET("/:id", GetLabel())
			labels.POST("", CreateLabel())
			labels.DELETE("/:id", DeleteLabel())
		}

		groups := v1.Group("groups")
		{
			groups.Use(MustGroupAdmins())
			groups.GET("", ListGroups())
		}

		// Only active user can make the following calls
		// router.Use(MustActive())

		// System info related APIs that have no object group affiliation
		v1.GET("apps", func(c *gin.Context) {
			resMap := map[string]interface{}{RestData: []string{"devops", "platform"}}
			c.JSON(axerror.REST_STATUS_OK, resMap)
		})

		templates := v1.Group("templates")
		{
			templates.GET("", func(c *gin.Context) {
				TemplateListHandler(c)
			})

			templates.GET("/:templateId", func(c *gin.Context) {
				TemplateGetHandler(c, c.Param("templateId"))
			})

			templates.DELETE("/:templateId", func(c *gin.Context) {
				TemplateDeleteHandler(c, c.Param("templateId"))
			})
		}

		services := v1.Group("services")
		{
			services.GET("", func(c *gin.Context) {
				ServiceListHandler(c)
			})

			services.GET("/:serviceId", func(c *gin.Context) {
				ServiceDetailHandler(c, c.Param("serviceId"))
			})

			services.GET("/:serviceId/outputs/:outputName", func(c *gin.Context) {
				ServiceOutputHandler(c, c.Param("serviceId"), c.Param("outputName"))
			})

			services.GET("/:serviceId/logs", func(c *gin.Context) {
				ServiceLogHandler(c, c.Param("serviceId"))
			})

			services.GET("/:serviceId/events", func(c *gin.Context) {
				ServiceEventHandler(c, c.Param("serviceId"))
			})

			services.GET("/:serviceId/issues", func(c *gin.Context) {
				ServiceJiraListHandler(c, c.Param("serviceId"))
			})

			services.POST("", func(c *gin.Context) {
				ServicePostHandler(c)
			})

			services.PUT("/:serviceId", func(c *gin.Context) {
				ServicePutHandler(c, c.Param("serviceId"))
			})

			services.DELETE("/:serviceId", func(c *gin.Context) {
				ServiceDeleteHandler(c, c.Param("serviceId"))
			})

			services.GET("/:serviceId/exec", ServiceConsoleWebSocket())
		}

		svc := v1.Group("service")
		{
			svc.GET("/events", func(c *gin.Context) {
				ServiceEventsHandler(c)
			})
		}

		system := v1.Group("system")
		{
			system.GET("status", func(c *gin.Context) {
				SystemStatusHandler(c)
			})

			system.GET("hosts", GetHostList())

			settings := system.Group("settings")
			{
				settings.GET("dnsname", GetDnsName())
				settings.PUT("dnsname", SetDnsName())

				settings.GET("spot_instance_config", GetSpotInstanceConfig())
				settings.PUT("spot_instance_config", SetSpotInstanceConfig())

				settings.GET("security_groups_config", GetSecurityGroupsConfig())
				settings.PUT("security_groups_config", SetSecurityGroupsConfig())

			}
		}

		cluster := v1.Group("cluster/settings")
		{
			cluster.GET("", ListClusterSettings())
			cluster.GET("/:key", GetClusterSetting())
			cluster.POST("", CreateClusterSetting())
			cluster.PUT("/:key", UpdateClusterSetting())
			cluster.DELETE("/:key", DeleteClusterSetting())
		}

		// XXX the UI needs to be redesign so that we can retire the concept of
		// build and test types. We should always use template names.
		v1.GET("builds/perf/:interval", func(c *gin.Context) {
			ServicePerfHandler(c, c.Param("interval"), "axbuild")
		})

		v1.GET("tests/perf/:interval", func(c *gin.Context) {
			ServicePerfHandler(c, c.Param("interval"), "axbuild") // XXX change axbuild to the proper test template
		})

		v1.GET("spendings/perf/:interval", func(c *gin.Context) {
			SpendingPerfHandler(c, c.Param("interval"))
		})

		v1.GET("spendings/breakdown/:interval", func(c *gin.Context) {
			SpendingPerfBreakDownHandler(c, c.Param("interval"))
		})

		v1.GET("spendings/detail/:start/:end", func(c *gin.Context) {
			SpendingDetailHandler(c, c.Param("start"), c.Param("end"))
		})

		repos := v1.Group("repos")
		{
			repos.GET("", GetRepoList())
		}

		fixture := v1.Group("fixture")
		{
			templates := fixture.Group("templates")
			{
				templates.GET("", ListFixtureTemplates())
				templates.GET("/:id", GetFixtureTemplateByID())
			}

			classes := fixture.Group("classes")
			{
				classes.POST("", CreateFixtureClass())
				classes.GET("", ListFixtureClasses())
				classes.GET("/:id", GetFixtureClassByID())
				classes.PUT("/:id", UpdateFixtureClass())
				classes.DELETE("/:id", DeleteFixtureClass())
			}

			instances := fixture.Group("instances")
			{
				instances.GET("", ListFixtureInstances())
				instances.POST("", CreateFixtureInstance())
				instances.GET("/:id", GetFixtureInstanceByID())
				instances.PUT("/:id", UpdateFixtureInstance())
				instances.DELETE("/:id", DeleteFixtureInstance())
				instances.POST("/:id/action", PerformFixtureInstanceAction())
			}
			summary := fixture.Group("summary")
			{
				summary.GET("", GetFixtureSummary())
			}
		}

		storage := v1.Group("storage")
		{
			classes := storage.Group("classes")
			{
				classes.GET("", ListStorageClasses())
			}
			volumes := storage.Group("volumes")
			{
				volumes.POST("", CreateVolume())
				volumes.GET("", ListVolumes())
				volumes.GET("/:id", GetVolume())
				volumes.GET("/:id/stats", GetVolumeStats())
				volumes.PUT("/:id", UpdateVolume())
				volumes.DELETE("/:id", DeleteVolume())

			}
		}

		customView := v1.Group("custom_views")
		{
			customView.GET("", ListCustomViews())
			customView.GET("/:id", GetCustomView())
			customView.POST("", CreateCustomView())
			customView.PUT("/:id", UpdateCustomView())
			customView.DELETE("/:id", DeleteCustomView())
		}

		search := v1.Group("search")
		{
			indexes := search.Group("indexes")
			{
				indexes.GET("", ListSearchIndexes())
			}
		}

		branches := v1.Group("branches")
		{
			branches.GET("", GetBranchList())

		}

		commits := v1.Group("commits")
		{
			commits.GET("", GetCommitList())
			commits.GET("/:revision", GetCommit())
		}

		policies := v1.Group("policies")
		{
			policies.GET("", ListPolicies())
			policies.GET("/:id", GetPolicy())
			policies.PUT("/:id/enable", EnablePolicy())
			policies.PUT("/:id/disable", DisablePolicy())
			policies.DELETE("/:id", DeletePolicy())
		}

		projects := v1.Group("projects")
		{
			projects.GET("", ListProjects())
			projects.GET("/:id", GetProject())
			projects.GET("/:id/icon", GetProjectIcon())
			projects.GET("/:id/publisher_icon", GetProjectPublisherIcon())
		}

		sandbox := v1.Group("sandbox")
		{
			sandbox.GET("/status", getSandboxStatus())
		}

		tools_get := v1.Group("/tools")
		{
			tools_get.GET("", GetToolList(internal))
			tools_get.GET("/:id", GetTool(internal))
		}

		tools := v1.Group("/tools")
		tools.Use(MustGroupAdmins())
		{
			tools.POST("", CreateTool())
			tools.POST("/test", TestTool())

			tools.PUT("/:id", PutTool(""))
			tools.DELETE("/:id", DeleteTool(""))

			tools.POST("/scm/git", CreateScmGit())
			tools.POST("/scm/github", CreateScmGitHub())
			tools.POST("/scm/bitbucket", CreateScmBitbucket())
			tools.POST("/scm/gitlab", CreateGitLab())
			tools.POST("/scm/codecommit", CreateScmCodeCommit())
			tools.POST("/notification/smtp", CreateNotificationSMTP())
			tools.POST("/notification/slack", CreateNotificationSlack())
			tools.POST("/notification/splunk", CreateNotificationSplunk())
			tools.POST("/authentication/saml", CreateAuthenticationSAML())
			tools.POST("/certificate/server", CreateCertificateServer())
			tools.POST("/registry/dockerhub", CreateRegistryDockerHub())
			tools.POST("/registry/private_registry", CreateRegistryPrivate())
			tools.POST("/domain_management/route53", CreateDomain())
			tools.POST("/artifact_management/nexus", CreateNexus())
			tools.POST("/issue_management/jira", CreateJira())
		}

		slack := v1.Group("/slack")
		slack.Use(MustGroupAdmins())
		{

			slack.GET("/channels", GetSlackChannels())
		}

		// for access API histogram
		profile := v1.Group("/profile")
		tools.Use(MustGroupAdmins())
		{
			profile.GET("/:interval", func(c *gin.Context) {
				ProfileHandler(c, c.Param("interval"), "json")
			})
		}

		profile_html := v1.Group("/profile_html")
		tools.Use(MustGroupAdmins())
		{
			profile_html.GET("/:interval", func(c *gin.Context) {
				ProfileHandler(c, c.Param("interval"), "html")
			})
		}

		if internal {
			v1.POST("/yamls", PostYAMLs())
		}

		secret := v1.Group("secret")
		{
			secret.POST("/encrypt", func(c *gin.Context) {
				SecretEncryptHandler(c)
			})
			if internal {
				secret.POST("/decrypt", func(c *gin.Context) {
					SecretDecryptHandler(c)
				})
			}
			secret.Use(MustSuperAdmin())
			{
				secret.POST("/key", func(c *gin.Context) {
					SecretDownloadKeyHandler(c)
				})
				secret.PUT("/key", func(c *gin.Context) {
					SecretUpdateKeyHandler(c)
				})
			}

		}

		configuration := v1.Group("configurations")
		{
			configuration.GET("", ListConfigurations())
			configuration.GET("/:user", GetConfigurationsByUser())
			configuration.GET("/:user/:name", GetConfigurationsByUserName())
			configuration.POST("", CreateConfiguration())
			configuration.PUT("", ModifyConfiguration())
			configuration.DELETE("/:user/:name", DeleteConfiguration())
		}

		applications := v1.Group("applications")
		{
			applications.GET("", ListApplications())
			applications.GET("/:id", GetApplication())
			applications.POST("", PostApplication())
			applications.PUT("/:id", PutApplication())
			applications.DELETE("/:id", DeleteApplication())

			applications.POST("/:id/start", StartApplication())
			applications.POST("/:id/stop", StopApplication())
		}

		application := v1.Group("application")
		{
			application.GET("/histories", ListApplicationHistories())
			application.GET("/events", AppEventsHandler())
		}

		deployments := v1.Group("deployments")
		{
			deployments.GET("", ListDeployments())
			deployments.GET("/:id", GetDeployment())
			deployments.POST("", PostDeployment())
			deployments.PUT("/:id", UpdateDeployment())
			deployments.DELETE("/:id", DeleteDeployment())

			deployments.POST("/:id/start", StartDeployment())
			deployments.POST("/:id/stop", StopDeployment())
			deployments.POST("/:id/scale", ScaleDeployment())
			deployments.GET("/:id/livelog", GetDeploymentLiveLog())
			deployments.GET("/:id/exec", DeploymentConsoleWebSocket())
		}

		deployment := v1.Group("deployment")
		{
			deployment.GET("/histories", ListDeploymentsHistory())
			deployment.GET("/events", DeploymentEventsHandler())
		}

		artifacts := v1.Group("artifacts")
		{
			artifacts.GET("", ListArtifacts())
			//artifacts.GET("/:artifact_id", ListArtifact())

			artifacts.GET("/browse", BrowseArtifact())
			artifacts.GET("/download", DownloadArtifact())
			//artifacts.PUT("/delete", DeleteArtifacts())
			//artifacts.PUT("/restore", RestoreArtifact())
			artifacts.PUT("", func(c *gin.Context) {
				ArtifactOperationHandler(c)
			})
		}

		rention_policy := v1.Group("retention_policies")
		{
			rention_policy.GET("", ListRetentionPolicies())
			rention_policy.GET("/:name", GetRetentionPolicy())
			rention_policy.POST("", CreateRetentionPolicy())
			rention_policy.PUT("/:name", UpdateRetentionPolicy())
			rention_policy.DELETE("/:name", DeleteRetentionPolicy())
		}

		workflows := v1.Group("workflows")
		{
			//workflows.PUT("/tag", ArtifactProcessor())
			//workflows.PUT("/untag", ArtifactProcessor())
			workflows.PUT("/tag", TagWorkflows())
			workflows.PUT("/untag", UntagWorkflows())
		}

		tags := v1.Group("tags")
		{
			tags.GET("", ArtifactProcessor())
		}

		notificationCenter := v1.Group("notification_center")
		{
			notificationCenter.GET("/channels", func(c *gin.Context) { ChannelListHandler(c) })
			notificationCenter.GET("/severities", func(c *gin.Context) { SeverityListHandler(c) })
			notificationCenter.GET("/rules", func(c *gin.Context) { RuleListHandler(c) })
			notificationCenter.POST("/rules", func(c *gin.Context) { RuleCreateHandler(c) })
			notificationCenter.PUT("/rules/:id", func(c *gin.Context) { RuleUpdateHandler(c, c.Param("id")) })
			notificationCenter.DELETE("/rules/:id", func(c *gin.Context) { RuleDeleteHandler(c, c.Param("id")) })
			notificationCenter.GET("/events", func(c *gin.Context) { EventListHandler(c) })
			notificationCenter.PUT("/events/:id/read", func(c *gin.Context) { MarkEventAsRead(c) })
		}

		jira := v1.Group("jira")
		{
			jira.GET("/users", ListJiraUsers())
			jira.GET("/projects", ListJiraProjects())
			jira.GET("/projects/:key", GetJiraProject())
			jira.GET("/issues", ListJiraIssues())
			jira.POST("/issues", func(c *gin.Context) { CreateJiraIssue(c) })
			jira.GET("/issues/:key", GetJiraIssue())
			jira.GET("/issuetypes", ListJiraIssueTypes())
			jira.GET("/issuetypes/:key", GetJiraIssueType())
			jira.GET("/issues/:key/getcomments", GetJiraIssueComments())
			jira.POST("/issues/:key/addcomment", AddJiraIssueComment())
			jira.DELETE("/issues/:id", func(c *gin.Context) { DeleteJiraIssue(c, c.Param("id")) })
			jira.PUT("/issues", func(c *gin.Context) { UpdateJiraIssue(c) })
			jira.PUT("/issues/:jiraId/service/:serviceId", func(c *gin.Context) {
				JiraServiceHandler(c, c.Param("serviceId"), c.Param("jiraId"))
			})
			jira.PUT("/issues/:jiraId/application/:appId", func(c *gin.Context) {
				JiraApplicationHandler(c, c.Param("appId"), c.Param("jiraId"))
			})
			jira.PUT("/issues/:jiraId/deployment/:deployId", func(c *gin.Context) {
				JiraDeploymentHandler(c, c.Param("deployId"), c.Param("jiraId"))
			})

		}
	}

	// Internal doesn't need UI
	if !internal {
		staticHandler := static.Serve("/", static.LocalFile("../public", true))
		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.RequestURI, "/assets") || strings.HasPrefix(c.Request.RequestURI, "/fonts") || strings.HasPrefix(c.Request.RequestURI, "/.well-known") || strings.HasPrefix(c.Request.RequestURI, "/files") {
				staticHandler(c)
			} else {
				http.ServeFile(c.Writer, c.Request, "../public/index.html")
			}
		})
	}

	return router
}

func NoCache(c *gin.Context) {
	if strings.HasPrefix(c.Request.RequestURI, "/assets") &&
		(strings.HasSuffix(c.Request.RequestURI, ".js") || strings.HasSuffix(c.Request.RequestURI, ".css")) {
		c.Header("Cache-Control", "max-age:290304000, public")
	} else if strings.HasPrefix(c.Request.RequestURI, "/v1/applications") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/application/histories") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/deployments") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/deployment/histories") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/artifacts") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/system/settings") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/retention_policies") ||
		strings.HasPrefix(c.Request.RequestURI, "/v1/jira") {
	} else {
		//c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		c.Header("Cache-Control", "max-age=0, must-revalidate")
		//c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	}
	c.Next()
}

func GatewaySCMWebhook() gin.HandlerFunc {
	scmWebhookUrl := DevopsCl.GetRootUrl() + "/scm/events"
	url, err := url.Parse(scmWebhookUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the gateway webhook url: %v", scmWebhookUrl))
	}
	fmt.Println(scmWebhookUrl)
	return gin.WrapH(NewSingleHostReverseProxy(url))
}

func GatewayJiraWebhook() gin.HandlerFunc {
	jiraWebhookUrl := DevopsCl.GetRootUrl() + "/jira/events"
	url, err := url.Parse(jiraWebhookUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the gateway webhook url: %v", jiraWebhookUrl))
	}
	fmt.Println(jiraWebhookUrl)
	return gin.WrapH(NewSingleHostReverseProxy(url))
}

func GatewayApproval() gin.HandlerFunc {
	approvalUrl := DevopsCl.GetRootUrl()
	url, err := url.Parse(approvalUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the approval url: %v", approvalUrl))
	}
	fmt.Println(approvalUrl)
	return gin.WrapH(NewSingleHostReverseProxyApproval(url))
}

//func ArtifactProcessor(c *gin.Context, artId string, op string) gin.HandlerFunc {
//
//	artifactUrl := ArtifactCl.GetRootUrl() + "/artifacts"
//	if len(artId) != 0 {
//		artifactUrl = artifactUrl + "/" + artId
//		if len(op) != 0 {
//			artifactUrl = artifactUrl + "/" + op
//		}
//	}
//	url, err := url.Parse(artifactUrl)
//	if err != nil {
//		panic(fmt.Sprintf("Can not parse the artifact manager url: %v", artifactUrl))
//	}
//	url.RawQuery = c.Request.URL.RawQuery
//	fmt.Printf("url query: %v, path = %v, schema = %v\n", url.RawQuery, url.Path, url.Scheme)
//	fmt.Println(artifactUrl)
//	//return gin.WrapH(NewSingleHostReverseProxyArtifact(url))
//	return gin.WrapH(httputil.NewSingleHostReverseProxy(url))
//}

func ArtifactProcessor() gin.HandlerFunc {
	artifactUrl := "http://axartifactmanager:9892/"
	url, err := url.Parse(artifactUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the artifact manager url: %v", artifactUrl))
	}
	fmt.Printf(artifactUrl)
	return gin.WrapH(httputil.NewSingleHostReverseProxy(url))
}

func JiraProcessor() gin.HandlerFunc {
	jiraUrl := "http://gateway:8889/"
	url, err := url.Parse(jiraUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the Jira manager url: %v", jiraUrl))
	}
	fmt.Printf(jiraUrl)
	return gin.WrapH(httputil.NewSingleHostReverseProxy(url))
}

// Override the golang NewSingleHostReverseProxy which take url target path directly
func NewSingleHostReverseProxyArtifact(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		fmt.Printf("url query: %v, path = %v, schema = %v\n", req.URL.RawQuery, req.URL.Path, req.URL.Scheme)
	}
	return &httputil.ReverseProxy{Director: director}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewSingleHostReverseProxyWithUserContext returns a reverse proxy that provides additional user context from current session.
// UserID and Username is retrieved from the session cookie, and is supplied to the target in the X-AXUserID, X-AXUsername HTTP headers
// on a best effort basis. This implementation is identical to httputil.NewSingleHostReverseProxy(), with addditional code to loookup
// AX session info.
func NewSingleHostReverseProxyWithUserContext(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		cookie, err := req.Cookie(session.COOKIE_SESSION_TOKEN)
		if err == nil {
			sessionID, _ := url.QueryUnescape(cookie.Value)
			ssn := &session.Session{
				ID: sessionID,
			}
			ssn, axErr := ssn.Reload()
			if axErr == nil {
				req.Header[HTTPAxUserIDHeader] = []string{ssn.UserID}
				req.Header[HTTPAxUsernameHeader] = []string{ssn.Username}
			} else {
				// This generally should not happen, since we should have already passed a session check against /v1 APIs
				// (unless we are being accessed from axops-internal)
				fmt.Printf("Failed to reload session information from session ID %s: %s", sessionID, axErr)
			}
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func AppMonitorMgrProxy() gin.HandlerFunc {
	ammUrl := "http://axamm:8966/"
	url, err := url.Parse(ammUrl)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the amm url: %v", ammUrl))
	}
	fmt.Printf(ammUrl)
	return gin.WrapH(httputil.NewSingleHostReverseProxy(url))

}

// FixtureManagerProxy is a reverse proxy to fixture manager service which optionally supplies user's session context as HTTP headers
func FixtureManagerProxy(withUserContext bool) gin.HandlerFunc {
	fixMgrURL := "http://fixturemanager:8912/"
	url, err := url.Parse(fixMgrURL)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the fixturemanager url: %v", fixMgrURL))
	}
	fmt.Printf(fixMgrURL)
	if withUserContext {
		return gin.WrapH(NewSingleHostReverseProxyWithUserContext(url))
	}
	return gin.WrapH(httputil.NewSingleHostReverseProxy(url))
}

// Override the golang NewSingleHostReverseProxy which take url target path directly
func NewSingleHostReverseProxyApproval(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		//req.URL.Path = target.Path
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

// Override the golang NewSingleHostReverseProxy which take url target path directly
func NewSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func getGzipHandler() func(c *gin.Context) {
	gzip := gzip.Gzip(gzip.BestCompression)
	return func(c *gin.Context) {
		// Disable gzip for logs/events API to support chunked response
		if strings.HasPrefix(c.Request.RequestURI, "/v1/services") &&
			(strings.HasSuffix(c.Request.RequestURI, "/logs") ||
				strings.HasSuffix(c.Request.RequestURI, "/events") ||
				strings.Contains(c.Request.RequestURI, "/outputs/")) {

			c.Next()

		} else if strings.HasPrefix(c.Request.RequestURI, "/v1/service/events") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/application/events") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/application/histories") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/deployment/events") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/deployment/histories") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/applications") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/deployments") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/artifacts") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/system/settings") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/retention_policies") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/jira") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/fixture") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/storage/volume") {
			c.Next()

		} else {
			gzip(c)
		}
	}
}
