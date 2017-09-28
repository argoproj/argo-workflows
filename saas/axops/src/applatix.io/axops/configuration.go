package axops

import (
	"encoding/json"
	"io/ioutil"
	"regexp"

	"applatix.io/axerror"
	"applatix.io/axops/configuration"
	"applatix.io/axops/user"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
)

const (
	InputStringRegex = "^[-0-9A-Za-z_]+$"
)

// @Title GetConfigurations
// @Description List configurations
// @Accept  json
// @Param   user       query    string   false        "user"
// @Param   name       query    string   false        "configuration name"
// @Success 200 {object} configuration.ConfigurationData
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations [GET]
func ListConfigurations() gin.HandlerFunc {
	return func(c *gin.Context) {
		params, axErr := GetContextParams(c,
			[]string{
				configuration.ConfigurationUserName,
				configuration.ConfigurationName,
			},
			[]string{},
			[]string{},
			[]string{})
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axErr)
			return
		}
		configList, axErr := configuration.GetConfigurations(params)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, configList)
		return
	}
}

// @Title GetConfigurationsByUser
// @Description Get configurations by user
// @Accept  json
// @Param   user       path    string   true        "user"
// @Success 200 {object} configuration.ConfigurationData
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user} [GET]
func GetConfigurationsByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		configList, axErr := configuration.GetConfigurationsByUser(username)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, configList)
		return
	}
}

// @Title GetConfiguration
// @Description Get configurations by user
// @Accept  json
// @Param   user       path    string   true        "user"
// @Param   name       path    string   true        "configuration name"
// @Success 200 {object} configuration.ConfigurationData
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [GET]
func GetConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		name := c.Param("name")
		config, axErr := configuration.GetConfiguration(username, name)
		if axErr != nil {
			httpCode := axerror.REST_INTERNAL_ERR
			if axErr.Code == axerror.ERR_API_RESOURCE_NOT_FOUND.Code {
				httpCode = axerror.REST_NOT_FOUND
			}
			c.JSON(httpCode, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, config)
	}
}

// @Title CreateConfiguration
// @Description Create configuration
// @Accept  json
// @Param   configuration    body    configuration.ConfigurationData   true   "configuration object"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations [POST]
func CreateConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Failed to read request body, err: "+err.Error()))
			return
		}
		var config configuration.ConfigurationData
		err = json.Unmarshal(body, &config)

		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid configuration object, err: "+err.Error()))
			return
		}

		//Verify user exists in the system
		u, axErr := user.GetUserByName(config.ConfigurationUser)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Fail to retrieve users from db"))
			return
		}

		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("User %s does not exist", config.ConfigurationUser))
			return
		}

		//Verify name
		pass, err := verifyInput(config.ConfigurationName)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
			return
		}
		if !pass {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Name is not valid, as it has to comply with %v", InputStringRegex))
			return
		}

		//Verify key
		for k := range config.ConfigurationValue {
			pass, err := verifyInput(k)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
				return
			}
			if !pass {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Configuration key %s is not valid, as it has to comply with %v", k, InputStringRegex))
				return
			}
		}

		//Verify if config already exists
		configList, axErr := configuration.GetConfigurationsByUserName(config.ConfigurationUser, config.ConfigurationName)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		if len(configList) != 0 {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Failed to create as configuration with namespace %v and name %v already exists.",
				config.ConfigurationUser, config.ConfigurationName))
			return
		}

		axErr = configuration.CreateConfiguration(&config)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_CREATE_OK, common.NullMap)
		return
	}
}

// @Title ModifyConfiguration
// @Description Modify configuration
// @Accept  json
// @Param   configuration    body    configuration.ConfigurationData   true   "configuration object"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [PUT]
func ModifyConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Failed to read request body, err: "+err.Error()))
			return
		}
		var config configuration.ConfigurationData
		err = json.Unmarshal(body, &config)

		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid configuration object, err: "+err.Error()))
			return
		}
		//Verify user exists in the system
		u, axErr := user.GetUserByName(config.ConfigurationUser)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Fail to retrieve users from db"))
			return
		}

		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("User %s does not exist", config.ConfigurationUser))
			return
		}

		//Verify name
		pass, err := verifyInput(config.ConfigurationName)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
			return
		}
		if !pass {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Name is not valid, as it has to comply with %v", InputStringRegex))
			return
		}

		//Verify key
		for k := range config.ConfigurationValue {
			pass, err := verifyInput(k)
			if err != nil {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
				return
			}
			if !pass {
				c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Configuration key %s is not valid, as it has to comply with %v", k, InputStringRegex))
				return
			}
		}

		//Verify if config already exists
		prevConfig, axErr := configuration.GetConfiguration(config.ConfigurationUser, config.ConfigurationName)
		if axErr != nil {
			if axErr.Code == axerror.ERR_API_RESOURCE_NOT_FOUND.Code {
				c.JSON(axerror.REST_NOT_FOUND, axErr)
			} else {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}
			return
		}
		if prevConfig.ConfigurationIsSecrets != config.ConfigurationIsSecrets {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Converting between secret and configurations is not allowed for %v/%v.",
				config.ConfigurationUser, config.ConfigurationName))
			return
		}

		axErr = configuration.UpdateConfiguration(&config)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, common.NullMap)
		return
	}
}

// @Title DeleteConfiguration
// @Description Delete configuration
// @Accept  json
// @Param   user             path    string   true        "user"
// @Param   name             path    string   true        "configuration name"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [DELETE]
func DeleteConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		name := c.Param("name")
		config, axErr := configuration.GetConfiguration(username, name)
		if axErr != nil {
			if axErr.Code == axerror.ERR_API_RESOURCE_NOT_FOUND.Code {
				c.JSON(axerror.REST_STATUS_OK, common.NullMap)
			} else {
				c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			}
			return
		}
		axErr = configuration.DeleteConfiguration(config)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}
		c.JSON(axerror.REST_STATUS_OK, common.NullMap)
		return
	}
}

func verifyInput(s string) (matched bool, err error) {
	matched, err = regexp.MatchString(InputStringRegex, s)
	return matched, err
}
