// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package configuration

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"regexp"
)

type ConfigurationData struct {
	ConfigurationUser        string            `json:"user"`
	ConfigurationName        string            `json:"name"`
	ConfigurationDesc        string            `json:"description,omitempty"`
	ConfigurationValue       map[string]string `json:"value,omitempty"`
	ConfigurationDateCreated int64             `json:"ctime,omitempty"`
	ConfigurationLastUpdated int64             `json:"mtime,omitempty"`
}

type ConfigurationContext struct {
	User string
	Name string
	Key  string
}

const (
	ConfigurationStrRegex = "^%%config\\.([^ ]*)\\.([-0-9A-Za-z_]+)\\.([-0-9A-Za-z_]+)%%$"
)

func GetConfigurations(params map[string]interface{}) ([]ConfigurationData, *axerror.AXError) {
	configs := []ConfigurationData{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, ConfigurationTableName, params, &configs)
	if axErr != nil {
		return nil, axErr
	}
	return configs, nil
}

func GetConfigurationsByUser(user string) ([]ConfigurationData, *axerror.AXError) {
	configs, axErr := GetConfigurations(map[string]interface{}{
		ConfigurationUserName: user,
	})
	if axErr != nil {
		return nil, axErr
	}
	return configs, nil
}

func GetConfigurationsByUserName(user string, name string) ([]ConfigurationData, *axerror.AXError) {
	configs, axErr := GetConfigurations(map[string]interface{}{
		ConfigurationName:     name,
		ConfigurationUserName: user,
	})
	if axErr != nil {
		return nil, axErr
	}
	return configs, nil
}

func CreateConfiguration(config *ConfigurationData) *axerror.AXError {
	_, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, ConfigurationTableName, config)
	if axErr != nil {
		return axErr
	}
	return nil
}

func UpdateConfiguration(config *ConfigurationData) *axerror.AXError {
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, ConfigurationTableName, config)
	if axErr != nil {
		return axErr
	}
	return nil
}

func DeleteConfiguration(config *ConfigurationData) *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, ConfigurationTableName, []*ConfigurationData{config})
	if axErr != nil {
		return axErr
	}
	return nil
}

// Validate if a string is a valid configuration string. e.g. %%config.joe@example.com.sql.username%%
func ValidateConfigurationStr(config string) (matched bool, err error) {
	matched, err = regexp.MatchString(ConfigurationStrRegex, config)
	return matched, err
}

func RetrieveConfigurationValue(configContext ConfigurationContext) (string, *axerror.AXError) {
	configs, axErr := GetConfigurationsByUserName(configContext.User, configContext.Name)
	if axErr != nil {
		return "", axErr
	}
	if len(configs) == 0 {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Configuration does not exist with user %s, name %s and key %s", configContext.User, configContext.Name, configContext.Key)
	}
	if len(configs) != 1 {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("More than one configuration exist with user %s, name %s and key %s", configContext.User, configContext.Name, configContext.Key)
	}
	value := configs[0].ConfigurationValue
	configValue, ok := value[configContext.Key]
	if !ok {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Configuration exists under user %s, name %s but does not have key %s", configContext.User, configContext.Name, configContext.Key)
	}
	return configValue, nil
}

func ProcessConfigurationStr(configStr string) (string, *axerror.AXError) {
	matched, err := ValidateConfigurationStr(configStr)
	if err != nil {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to validate if %s is a valid configuration variable: %v", configStr, err)
	}
	if matched == false {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("%s is an invalid configuration variable expression", configStr)
	}

	re := regexp.MustCompile(ConfigurationStrRegex)
	matched_string := re.FindStringSubmatch(configStr)

	if len(matched_string) != 4 {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("%s is an invalid configuration variable expression", configStr)
	}
	config := ConfigurationContext{
		User: matched_string[1],
		Name: matched_string[2],
		Key:  matched_string[3],
	}

	configValue, axErr := RetrieveConfigurationValue(config)
	if axErr != nil {
		return "", axErr
	}
	return configValue, nil
}
