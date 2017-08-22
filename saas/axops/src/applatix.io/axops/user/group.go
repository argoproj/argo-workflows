package user

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
)

const (
	GroupSuperAdmin = "super_admin"
	GroupAdmin      = "admin"
	GroupDeveloper  = "developer"
)

type Group struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Usernames []string `json:"usernames, omitempty"`
}

func (g *Group) Update() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, GroupTableName, g); axErr != nil {
		return axErr
	}
	return nil
}

func (g *Group) Validate() *axerror.AXError {
	_, axErr := g.Reload()
	return axErr
}

func (g *Group) Reload() (*Group, *axerror.AXError) {
	if g.Name == "" {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("Missing group name.")
	}

	group, axErr := GetGroupByName(g.Name)
	if axErr != nil {
		return nil, axErr
	}

	if group == nil {
		return nil, axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessagef("Cannot find group with name: %v", g.Name)
	}

	return group, nil
}

func (g *Group) Delete() *axerror.AXError {
	if len(g.Usernames) != 0 {
		return axerror.ERR_API_INVALID_REQ.NewWithMessage("Cannot delete the group. There are still users belongs to this group.")
	}
	return g.delete()
}

func (g *Group) delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, GroupTableName, []*Group{g})
	if axErr != nil {
		utils.ErrorLog.Printf("Delete group failed:%v\n", axErr)
	}
	return nil
}

func (g *Group) Create() (*Group, *axerror.AXError) {

	if group, axErr := GetGroupByName(g.Name); axErr != nil {
		return nil, axErr
	} else {
		if group != nil {
			return nil, axerror.ERR_API_DUP_GROUPNAME.New()
		}
	}

	g.ID = utils.GenerateUUIDv1()

	if axErr := g.Update(); axErr != nil {
		return nil, axErr
	}

	return g, nil
}

func (g *Group) HasUser(username string) bool {
	for _, u := range g.Usernames {
		if u == username {
			return true
		}
	}
	return false
}

func GetGroupById(id string) (*Group, *axerror.AXError) {
	groups, axErr := GetGroups(map[string]interface{}{
		GroupID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(groups) == 0 {
		return nil, nil
	}

	group := groups[0]
	return &group, nil
}

func GetGroupByName(name string) (*Group, *axerror.AXError) {
	groups, axErr := GetGroups(map[string]interface{}{
		GroupName: name,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(groups) == 0 {
		return nil, nil
	}

	group := groups[0]
	return &group, nil
}

func GetGroups(params map[string]interface{}) ([]Group, *axerror.AXError) {
	groups := []Group{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, GroupTableName, params, &groups)
	if axErr != nil {
		return nil, axErr
	}

	return groups, nil
}
