package cluster

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"time"
)

type ClusterSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ctime int64  `json:"ctime,omitempty"`
	Mtime int64  `json:"mtime,omitempty"`
}

func (v *ClusterSetting) Validate() (*axerror.AXError, int) {
	return nil, axerror.REST_STATUS_OK
}

func (v *ClusterSetting) Create() (*ClusterSetting, *axerror.AXError, int) {
	v.Ctime = time.Now().Unix()
	v.Mtime = time.Now().Unix()

	axErr, code := v.Validate()
	if axErr != nil {
		return nil, axErr, code
	}

	if axErr := v.create(); axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return v, nil, axerror.REST_CREATE_OK
}

func (v *ClusterSetting) Update() (*ClusterSetting, *axerror.AXError, int) {
	axErr, code := v.Validate()
	if axErr != nil {
		return nil, axErr, code
	}

	v.Mtime = time.Now().Unix()
	if axErr := v.update(); axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}
	return v, nil, axerror.REST_STATUS_OK
}

func (v *ClusterSetting) create() *axerror.AXError {
	if _, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, ClusterSettingTable, v); axErr != nil {
		return axErr
	}

	ClusterSettings[v.Key] = v.Value

	return nil
}

func (v *ClusterSetting) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, ClusterSettingTable, v); axErr != nil {
		return axErr
	}

	ClusterSettings[v.Key] = v.Value

	return nil
}

func (v *ClusterSetting) Delete() (*axerror.AXError, int) {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, ClusterSettingTable, []*ClusterSetting{v})
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	delete(ClusterSettings, v.Key)
	return nil, axerror.REST_STATUS_OK
}

func GetClusterSetting(key string) (*ClusterSetting, *axerror.AXError) {
	params := map[string]interface{}{ClusterKey: key}
	views, axErr := GetClusterSettings(params)
	if axErr != nil {
		return nil, axErr
	}

	if len(views) != 0 {
		return views[0], nil
	} else {
		return nil, nil
	}
}

func GetClusterSettings(params map[string]interface{}) ([]*ClusterSetting, *axerror.AXError) {
	settings := []*ClusterSetting{}

	if params == nil {
		params = map[string]interface{}{}
	}

	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, ClusterSettingTable, params, &settings)
	if dbErr != nil {
		return nil, dbErr
	}
	return settings, nil
}
