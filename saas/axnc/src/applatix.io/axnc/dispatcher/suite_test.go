package dispatcher

import (
	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axdb/core"
	"applatix.io/axnc"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/common"

	"gopkg.in/check.v1"

	"testing"
	"time"
)

const (
	axdbAddr = "http://localhost:8080/v1"
)

type S struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&S{})

var tables = []axdb.Table{
	axnc.CodeSchema,
	axnc.RuleSchema,
	axnc.EventSchema,
	user.UserSchema,
	user.GroupSchema,
}

func (s *S) SetUpSuite(c *check.C) {
	common.InitLoggers("AXNC")

	core.InitLoggers()
	core.InitDB(1)
	core.ReloadDBTable()
	go core.StartRouter(true)

	utils.Dbcl = axdbcl.NewAXDBClientWithTimeout(axdbAddr, 60*time.Minute)
	var response []interface{}
	for utils.Dbcl.Get("axdb", "version", nil, &response) != nil {
		time.Sleep(5 * time.Second)
	}

	// Create tables
	for _, table := range tables {
		_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXDB, axdb.AXDBOpUpdateTable, table)
		c.Assert(axErr, check.IsNil)
		common.InfoLog.Printf("Updated table (name: %s)", table.Name)
	}

	// Create groups
	testGroup := map[string]interface{}{
		user.GroupID:        "00000000-0000-0000-0000-000000000000",
		user.GroupName:      user.GroupSuperAdmin,
		user.GroupUserNames: []string{"admin@internal", "tester@email.com"},
	}
	_, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, user.GroupTableName, testGroup)
	if axErr != nil {
		c.Fatalf("Failed to create test group (group: %s, err: %v)", testGroup["name"].(string), axErr)
	}

	// Create users
	testUsers := []map[string]interface{}{
		map[string]interface{}{
			user.UserID:              "00000000-0000-0000-0000-000000000000",
			user.UserName:            "admin@internal",
			user.UserFirstName:       "Admin",
			user.UserLastName:        "",
			user.UserPassword:        common.GenerateUUIDv1(),
			user.UserState:           user.UserStateActive,
			user.UserAuthSchemes:     []string{"native"},
			user.UserGroups:          []string{user.GroupSuperAdmin},
			user.UserSettings:        map[string]string{axnc.TopicEmail: "yes", axnc.TopicSlack: "yes"},
			user.UserViewPreferences: map[string]string{},
			user.UserLabels:          []string{},
			user.UserCtime:           time.Now().Unix(),
			user.UserMtime:           time.Now().Unix(),
		},
		map[string]interface{}{
			user.UserID:              "00000000-0000-0000-0000-000000000001",
			user.UserName:            "tester@email.com",
			user.UserFirstName:       "Tester",
			user.UserLastName:        "Tester",
			user.UserPassword:        common.GenerateUUIDv1(),
			user.UserState:           user.UserStateActive,
			user.UserAuthSchemes:     []string{"native"},
			user.UserGroups:          []string{user.GroupSuperAdmin},
			user.UserSettings:        map[string]string{axnc.TopicEmail: "yes", axnc.TopicSlack: "yes"},
			user.UserViewPreferences: map[string]string{},
			user.UserLabels:          []string{},
			user.UserCtime:           time.Now().Unix(),
			user.UserMtime:           time.Now().Unix(),
		},
	}

	for _, u := range testUsers {
		_, axErr = utils.Dbcl.Put(axdb.AXDBAppAXOPS, user.UserTableName, u)
		if axErr != nil {
			c.Fatalf("Failed to create test user (user: %s, err: %v)", u["username"].(string), axErr)
		}
	}
}

func (s *S) TearDownSuite(c *check.C) {}
