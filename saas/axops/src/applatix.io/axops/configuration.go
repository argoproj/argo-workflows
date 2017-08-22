package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/configuration"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"github.com/gin-gonic/gin"
	"regexp"
	"time"
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
		} else {
			c.JSON(axerror.REST_STATUS_OK, configList)
			return
		}
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
		} else {
			c.JSON(axerror.REST_STATUS_OK, configList)
			return
		}
	}
}

// @Title GetConfigurationsByUser
// @Description Get configurations by user
// @Accept  json
// @Param   user       path    string   true        "user"
// @Param   name       path    string   true        "configuration name"
// @Success 200 {object} configuration.ConfigurationData
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [GET]
func GetConfigurationsByUserName() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		name := c.Param("name")
		configList, axErr := configuration.GetConfigurationsByUserName(username, name)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			c.JSON(axerror.REST_STATUS_OK, configList)
			return
		}
	}
}

// @Title CreateConfiguration
// @Description Create configuration
// @Accept  json
// @Param   user             path    string   true        "user"
// @Param   name             path    string   true        "configuration name"
// @Param   value            body    MapType   true        "configuration value (key value paired)"
// @Param   description      query   string   False       "configuration description"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [POST]
func CreateConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		name := c.Param("name")
		description := c.Request.URL.Query().Get("description")

		//Verify user exists in the system
		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Fail to retrieve users from db"))
			return
		}
		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("User %s does not exist", username))
			return
		}

		//Verify name
		pass, err := verifyInput(name)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
			return
		}
		if !pass {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Name is not valid, as it has to comply with %v", InputStringRegex))
			return
		}

		var v map[string]string
		jsonErr := utils.GetUnmarshalledBody(c, &v)

		if jsonErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Cannot marshall configuration value into map, %v", jsonErr))
			return
		}

		//Verify key
		for k := range v {
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
		config := &configuration.ConfigurationData{
			ConfigurationUser:        username,
			ConfigurationName:        name,
			ConfigurationDesc:        description,
			ConfigurationValue:       v,
			ConfigurationDateCreated: time.Now().Unix(),
			ConfigurationLastUpdated: time.Now().Unix(),
		}

		axErr = configuration.CreateConfiguration(config)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			c.JSON(axerror.REST_CREATE_OK, common.NullMap)
			return
		}
	}
}

// @Title ModifyConfiguration
// @Description Modify configuration
// @Accept  json
// @Param   user             path    string   true        "user"
// @Param   name             path    string   true        "configuration name"
// @Param   value            body    MapType  true        "configuration value (key value paired)"
// @Param   description      query   string   False       "configuration description"
// @Success 200 {object} MapType
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /configurations
// @Router /configurations/{user}/{name} [PUT]
func ModifyConfiguration() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		name := c.Param("name")
		description := c.Request.URL.Query().Get("description")

		//Verify user exists in the system
		u, axErr := user.GetUserByName(username)
		if axErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Fail to retrieve users from db"))
			return
		}

		if u == nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("User %s does not exist", username))
			return
		}

		//Verify name
		pass, err := verifyInput(name)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Fail to verify name of configuration, %v", err))
			return
		}
		if !pass {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessagef("Name is not valid, as it has to comply with %v", InputStringRegex))
			return
		}

		var v map[string]string
		jsonErr := utils.GetUnmarshalledBody(c, &v)

		if jsonErr != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.NewWithMessage("Cannot marshall configuration value into map"))
			return
		}

		//Verify key
		for k := range v {
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
		config := &configuration.ConfigurationData{
			ConfigurationUser:        username,
			ConfigurationName:        name,
			ConfigurationDesc:        description,
			ConfigurationValue:       v,
			ConfigurationLastUpdated: time.Now().Unix(),
			ConfigurationDateCreated: 0,
		}

		axErr = configuration.UpdateConfiguration(config)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		} else {
			c.JSON(axerror.REST_STATUS_OK, common.NullMap)
			return
		}
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
		configs, axErr := configuration.GetConfigurationsByUserName(username, name)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axErr)
			return
		}

		if configs == nil || len(configs) == 0 {
			c.JSON(axerror.REST_STATUS_OK, common.NullMap)
			return
		}

		if len(configs) > 1 {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INVALID_REQ.NewWithMessage("More than one configurations with the user and name"))
			return
		}

		axErr = configuration.DeleteConfiguration(&configs[0])
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
