package custom_view

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"time"
)

type CustomView struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Info     string `json:"info,omitempty"`
	Ctime    int64  `json:"ctime,omitempty"`
	Mtime    int64  `json:"mtime,omitempty"`
}

func (v *CustomView) Validate() (*axerror.AXError, int) {
	if v.Name == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("name is required."), axerror.REST_BAD_REQ
	}
	if v.Type == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("type is required."), axerror.REST_BAD_REQ
	}
	if v.UserID == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("user_id is required."), axerror.REST_BAD_REQ
	}
	if v.Username == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("username is required."), axerror.REST_BAD_REQ
	}
	if v.Info == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("info is required."), axerror.REST_BAD_REQ
	}
	return nil, axerror.REST_STATUS_OK
}

func (v *CustomView) Create() (*CustomView, *axerror.AXError, int) {
	v.ID = utils.GenerateUUIDv1()
	v.Ctime = time.Now().Unix()
	v.Mtime = time.Now().Unix()

	axErr, code := v.Validate()
	if axErr != nil {
		return nil, axErr, code
	}

	if axErr := v.update(); axErr != nil {
		return nil, axErr, axerror.REST_INTERNAL_ERR
	}

	return v, nil, axerror.REST_CREATE_OK
}

func (v *CustomView) Update() (*CustomView, *axerror.AXError, int) {
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

func (v *CustomView) update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, CustomViewTable, v); axErr != nil {
		return axErr
	}
	return nil
}

func (v *CustomView) Delete() (*axerror.AXError, int) {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, CustomViewTable, []*CustomView{v})
	if axErr != nil {
		return axErr, axerror.REST_INTERNAL_ERR
	}
	return nil, axerror.REST_STATUS_OK
}

func GetCustomViewById(id string) (*CustomView, *axerror.AXError) {
	params := map[string]interface{}{CustomViewID: id}
	views, axErr := GetCustomViews(params)
	if axErr != nil {
		return nil, axErr
	}

	if len(views) != 0 {
		return &views[0], nil
	} else {
		return nil, nil
	}
}

func GetCustomViews(params map[string]interface{}) ([]CustomView, *axerror.AXError) {
	var views []CustomView
	if params == nil {
		params = map[string]interface{}{}
	}

	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, CustomViewTable, params, &views)
	if dbErr != nil {
		return nil, dbErr
	}
	return views, nil
}
