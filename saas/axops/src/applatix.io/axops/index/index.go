package index

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"time"
)

type SearchIndex struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Ctime int64  `json:"ctime,omitempty"`
	Mtime int64  `json:"mtime,omitempty"`
}

func (v *SearchIndex) Validate() (*axerror.AXError, int) {
	if v.Type == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("type can not be empty"), axerror.REST_BAD_REQ
	}

	if v.Key == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("key can not be empty"), axerror.REST_BAD_REQ
	}

	if v.Value == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("value can not be empty"), axerror.REST_BAD_REQ
	}

	return nil, axerror.REST_STATUS_OK
}

func (v *SearchIndex) Create() (*SearchIndex, *axerror.AXError, int) {
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

func (v *SearchIndex) Update() (*SearchIndex, *axerror.AXError, int) {
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

func (v *SearchIndex) create() *axerror.AXError {
	if _, axErr := utils.Dbcl.Post(axdb.AXDBAppAXOPS, SearchIndexTable, v); axErr != nil {
		return axErr
	}

	return nil
}

func (v *SearchIndex) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, SearchIndexTable, v); axErr != nil {
		return axErr
	}

	return nil
}

func (v *SearchIndex) Delete() (*axerror.AXError, int) {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, SearchIndexTable, []*SearchIndex{v})
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}

	return nil, axerror.REST_STATUS_OK
}

func GetSearchIndexesByType(typeStr string) ([]*SearchIndex, *axerror.AXError) {
	params := map[string]interface{}{SearchIndexType: typeStr}
	indexes, axErr := GetSearchIndexes(params)
	if axErr != nil {
		return nil, axErr
	}
	return indexes, nil
}

func GetSearchIndexes(params map[string]interface{}) ([]*SearchIndex, *axerror.AXError) {
	indexes := []*SearchIndex{}

	if params == nil {
		params = map[string]interface{}{}
	}

	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, SearchIndexTable, params, &indexes)
	if dbErr != nil {
		return nil, dbErr
	}
	return indexes, nil
}

func CreateSearchIndex(typeStr, key, value string) *axerror.AXError {
	index := SearchIndex{
		Type:  typeStr,
		Key:   key,
		Value: value,
	}
	_, axErr, _ := index.Create()
	return axErr
}
