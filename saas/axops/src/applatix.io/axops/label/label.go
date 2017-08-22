package label

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"fmt"
	"strings"
	"time"
)

type Label struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Reserved bool   `json:"reserved"`
	Ctime    int64  `json:"ctime,omitempty"`
}

func (l *Label) Create() (*Label, *axerror.AXError) {

	if _, ok := LabelTypeMap[l.Type]; !ok {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Label type %v is a valid type.", l.Type)
	}

	l.Key = strings.TrimSpace(l.Key)
	if l.Key == "" {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Label key is required.")
	} else {
		l.Key = strings.ToLower(l.Key)
	}

	// Ignore the label with key "ax_*", it is used for hack feature
	if strings.HasPrefix(l.Key, "ax_") {
		return nil, nil
	}

	l.Value = strings.TrimSpace(l.Value)
	if l.Value == "" && l.Type != LabelTypeUser {
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Label value is required.")
	} else {
		l.Value = strings.ToLower(l.Value)
	}

	if label, axErr := GetLabel(l.Type, l.Key, l.Value); axErr != nil {
		return nil, axErr
	} else {
		if label != nil {
			return nil, axerror.ERR_API_DUP_LABEL.New()
		}
	}

	l.ID = l.GenerateID()
	l.Ctime = time.Now().Unix() * 1e6
	l.Reserved = false

	if axErr := l.Update(); axErr != nil {
		return nil, axErr
	}

	return l, nil
}

func (l *Label) Update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, LabelTableName, l); axErr != nil {
		return axErr
	}
	return nil
}

func (l *Label) GenerateID() string {
	return utils.GenerateUUIDv5(fmt.Sprintf("%v:%v:%v", l.Type, l.Key, l.Value))
}

func (l *Label) Delete() *axerror.AXError {

	if l.Reserved == true {
		return axerror.ERR_API_INVALID_REQ.NewWithMessage("Cannot delete reserved label")
	}

	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, LabelTableName, []*Label{l})
	if axErr != nil {
		utils.ErrorLog.Printf("Delete label failed:%v\n", axErr)
	}
	return nil
}

func (l *Label) Validate() *axerror.AXError {
	_, axErr := l.Reload()
	return axErr
}

func (l *Label) Reload() (*Label, *axerror.AXError) {
	if l.Type == "" {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("Missing label type.")
	}

	if l.Key == "" {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("Missing label key.")
	}

	if l.Value == "" && l.Type != LabelTypeUser {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("Missing label value.")
	}

	label, axErr := GetLabel(l.Type, l.Key, l.Value)
	if axErr != nil {
		return nil, axErr
	}

	if label == nil {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessagef("Cannot find %v label with key: %v, value: %v", l.Type, l.Key, l.Value)
	}

	return label, nil
}

func GetLabel(Type, key, value string) (*Label, *axerror.AXError) {
	labels, axErr := GetLabels(map[string]interface{}{
		LabelType:  Type,
		LabelKey:   key,
		LabelValue: value,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(labels) == 0 {
		return nil, nil
	}

	label := labels[0]
	return &label, nil
}

func GetLabelByID(id string) (*Label, *axerror.AXError) {
	labels, axErr := GetLabels(map[string]interface{}{
		LabelID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(labels) == 0 {
		return nil, nil
	}

	label := labels[0]
	return &label, nil
}

func GetLabels(params map[string]interface{}) ([]Label, *axerror.AXError) {
	labels := []Label{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, LabelTableName, params, &labels)
	if axErr != nil {
		return nil, axErr
	}

	return labels, nil
}
