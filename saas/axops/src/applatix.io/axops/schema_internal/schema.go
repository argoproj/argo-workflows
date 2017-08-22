package schema_internal

import (
	"applatix.io/axdb"
	"applatix.io/axerror"

	"applatix.io/axdb/axdbcl"
)

const (
	AppTable = "app"
	AppName  = "app_name"
	AppKey   = "key"
	AppValue = "value"
)

var AppSchema = axdb.Table{
	AppName: axdb.AXDBAppApp,
	Name:    AppTable,
	Type:    axdb.TableTypeKeyValue,
	Columns: map[string]axdb.Column{
		AppName:  axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexPartition},
		AppKey:   axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexClusteringStrong},
		AppValue: axdb.Column{Type: axdb.ColumnTypeString, Index: axdb.ColumnIndexNone},
	},
}

type AppKeyValue struct {
	AppName string `json:"app_name"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func GetAppKeyValsByAppName(appName string, client *axdbcl.AXDBClient) (map[string]string, *axerror.AXError) {
	kvMap := map[string]string{}

	keyVals, axErr := GetAppKeyVals(
		map[string]interface{}{
			AppName: appName,
		}, client,
	)

	if axErr != nil {
		return kvMap, axErr
	}

	for _, keyVal := range keyVals {
		kvMap[keyVal.Key] = keyVal.Value
	}

	return kvMap, nil
}

func GetAppKeyVals(params map[string]interface{}, client *axdbcl.AXDBClient) ([]AppKeyValue, *axerror.AXError) {
	keyVals := []AppKeyValue{}
	axErr := client.Get(axdb.AXDBAppApp, AppTable, params, &keyVals)
	if axErr != nil {
		return nil, axErr
	}
	return keyVals, nil
}
