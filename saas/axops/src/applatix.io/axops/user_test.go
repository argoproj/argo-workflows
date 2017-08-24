// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops_test

import (
	"fmt"
	"time"

	"applatix.io/axerror"
	"applatix.io/axops/user"
	"applatix.io/axops/utils"
	"applatix.io/restcl"
	"applatix.io/test"
	"gopkg.in/check.v1"
)

func (s *S) TestUserLogin(c *check.C) {
	// create admin dummy user
	user := user.User{
		ID:          "00000000-0000-0000-0000-000000000009",
		Username:    "admin@admin" + test.RandStr(),
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
		State:       user.UserStateActive,
	}
	user.ResetPassword("Test@test100", true)

	// login with right password
	_, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": user.Username,
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)

	// login with wrong password
	_, err = axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": user.Username,
			"password": "applatix",
		})
	c.Assert(err, check.NotNil)
}

func (s *S) TestUserList(c *check.C) {
	uu, err := user.GetUserByName("admin@internal")
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)

	err = uu.ResetPassword("Test@test100", false)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": "admin@internal",
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}
	var users GeneralGetResult
	err = axopsClient.Get("users", params, &users)
	c.Assert(err, check.IsNil)
	c.Assert(len(users.Data), check.Not(check.Equals), 0)
}

func (s *S) TestUserCreateWithoutState(c *check.C) {
	uu, err := user.GetUserByName("admin@internal")
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)

	err = uu.ResetPassword("Test@test100", false)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": "admin@internal",
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	u := &user.User{
		Username: "admin" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu.ID, check.Not(check.Equals), "")
	c.Assert(uu.State, check.Equals, user.UserStateInit)
	c.Assert(uu.AuthSchemes[0], check.Equals, "native")
	c.Assert(uu.Groups[0], check.Equals, user.GroupDeveloper)
	c.Assert(uu.Password, check.Equals, "")
}

func (s *S) TestUserCRUD(c *check.C) {
	uu, err := user.GetUserByName("admin@internal")
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)

	err = uu.ResetPassword("Test@test100", false)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": "admin@internal",
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	u := &user.User{
		Username: "admin" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
		State:    user.UserStateActive,
	}

	// C
	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu.ID, check.Not(check.Equals), "")
	c.Assert(uu.State, check.Equals, user.UserStateActive)
	c.Assert(uu.AuthSchemes[0], check.Equals, "native")
	c.Assert(uu.Groups[0], check.Equals, user.GroupDeveloper)
	c.Assert(uu.Password, check.Equals, "")

	// R
	copy := &user.User{}
	err = axopsClient.Get("users/"+uu.Username, params, copy)
	c.Assert(err, check.IsNil)
	c.Assert(copy.ID, check.Not(check.Equals), "")
	c.Assert(copy.State, check.Equals, user.UserStateActive)
	c.Assert(copy.AuthSchemes[0], check.Equals, "native")
	c.Assert(copy.Groups[0], check.Equals, user.GroupDeveloper)
	c.Assert(copy.Password, check.Equals, "")

	// U
	u = &user.User{}
	uu.State = user.UserStateBanned
	uu.Groups = []string{user.GroupAdmin}
	uu.FirstName = "Docker"
	uu.LastName = "Mac"
	err, _ = axopsExternalClient.Put2("users/"+uu.Username, params, uu, u)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateBanned)
	c.Assert(u.AuthSchemes[0], check.Equals, "native")
	c.Assert(u.Groups[0], check.Equals, user.GroupAdmin)
	c.Assert(u.Password, check.Equals, "")
	c.Assert(u.FirstName, check.Equals, uu.FirstName)
	c.Assert(u.LastName, check.Equals, uu.LastName)

	// D
	_, err = axopsClient.Delete("users/"+uu.Username, nil)
	c.Assert(err, check.IsNil)
	err = axopsClient.Get("users/"+uu.Username, params, copy)
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}

func (s *S) TestUserGetList(c *check.C) {
	c.Skip("Skip the test due to the index timing issue")

	var err *axerror.AXError

	testStr := test.RandStr()

	randStr := "randuser" + testStr + test.RandStr()
	u1 := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "admin@admin" + randStr,
		LastName:    "admin@admin" + randStr,
		FirstName:   "admin@admin" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
		State:       user.UserStateActive,
	}
	err = u1.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + testStr + test.RandStr()
	u2 := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "admin@admin" + randStr,
		LastName:    "admin@admin" + randStr,
		FirstName:   "admin@admin" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{"admin"},
		State:       user.UserStateBanned,
	}
	err = u2.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	time.Sleep(10 * time.Second)

	for _, field := range []string{
		user.UserName,
		user.UserLastName,
		user.UserFirstName,
		"search",
	} {
		params := map[string]interface{}{
			field: u1.Username,
		}

		// Filter by name
		data := &GeneralGetResult{}
		err := axopsClient.Get("users", params, data)
		c.Assert(err, check.IsNil)
		c.Assert(data, check.NotNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get one
		params = map[string]interface{}{
			field: "~" + u1.Username,
		}
		err = axopsClient.Get("users", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data), check.Equals, 1)

		// Search by name to get two
		params = map[string]interface{}{
			field: "~randuser",
		}
		err = axopsClient.Get("users", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)

		// Search by name to get two with limit one
		params = map[string]interface{}{
			field:   "~randuser",
			"limit": 1,
		}
		err = axopsClient.Get("users", params, &data)
		c.Assert(err, check.IsNil)
		fmt.Println("data:", data)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get one
		params = map[string]interface{}{
			"search": "~" + u1.Username,
		}
		err = axopsClient.Get("users", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) == 1, check.Equals, true)

		// Search to get more
		params = map[string]interface{}{
			"search": "~randuser",
		}
		err = axopsClient.Get("users", params, &data)
		c.Assert(err, check.IsNil)
		c.Assert(len(data.Data) >= 2, check.Equals, true)
	}

	params := map[string]interface{}{
		user.UserName: "~randuser" + testStr,
	}
	// Filter by name
	data := &GeneralGetResult{}
	err = axopsClient.Get("users", params, data)
	c.Assert(err, check.IsNil)
	c.Assert(data, check.NotNil)
	c.Assert(len(data.Data), check.Equals, 2)

	params = map[string]interface{}{
		user.UserName:  "~randuser" + testStr,
		user.UserState: user.UserStateBanned,
	}
	// Filter by name + state
	data = &GeneralGetResult{}
	err = axopsClient.Get("users", params, data)
	c.Assert(err, check.IsNil)
	c.Assert(data, check.NotNil)
	c.Assert(len(data.Data), check.Equals, 1)

	params = map[string]interface{}{
		user.UserName:  "~randuser" + testStr,
		user.UserState: user.UserStateActive,
	}
	// Filter by name + state
	data = &GeneralGetResult{}
	err = axopsClient.Get("users", params, data)
	c.Assert(err, check.IsNil)
	c.Assert(data, check.NotNil)
	c.Assert(len(data.Data), check.Equals, 1)
}

func (s *S) TestUserSuperAdmin(c *check.C) {
	var err *axerror.AXError

	randStr := "randuser" + test.RandStr()
	usr := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupSuperAdmin},
		State:       user.UserStateActive,
	}
	err = usr.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	superAdmin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupSuperAdmin},
		State:       user.UserStateActive,
	}
	err = superAdmin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	admin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupAdmin},
		State:       user.UserStateActive,
	}
	err = admin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	developer := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupDeveloper},
		State:       user.UserStateActive,
	}
	err = developer.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": usr.Username,
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	// Update Self
	randStr = test.RandStr()
	usr.LastName = randStr
	uuu := &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+usr.Username, params, usr, uuu)
	c.Assert(err, check.IsNil)
	c.Assert(uuu, check.NotNil)
	c.Assert(uuu.LastName, check.Equals, randStr)

	// Create Super Admin
	u := &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupSuperAdmin},
	}

	uu := &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Create Admin
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupAdmin},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)

	// Create Developer
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)

	// Update Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)

	// Update Admin
	randStr = test.RandStr()
	admin.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)
	c.Assert(uu.LastName, check.Equals, randStr)

	// Update Developer
	randStr = test.RandStr()
	developer.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)
	c.Assert(uu.LastName, check.Equals, randStr)

	// Ban/Unban Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Ban/Unban Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.IsNil)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.IsNil)

	// Ban/Unban Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/ban", params, nil, uu)
	c.Assert(err, check.IsNil)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/activate", params, nil, uu)
	c.Assert(err, check.IsNil)

	// Delete Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)

	// Delete Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.IsNil)

	// Delete Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.IsNil)

	username := "user" + test.RandStr() + "@company.com"
	// Invite SuperAdmin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupSuperAdmin,
	}
	var results map[string]interface{}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Invite Admin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupAdmin,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.IsNil)
	fmt.Println(err)

	// Invite Developer
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupDeveloper,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUserAdmin(c *check.C) {
	var err *axerror.AXError

	randStr := "randuser" + test.RandStr()
	usr := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupAdmin},
		State:       user.UserStateActive,
	}
	err = usr.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	superAdmin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupSuperAdmin},
		State:       user.UserStateActive,
	}
	err = superAdmin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	admin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupAdmin},
		State:       user.UserStateActive,
	}
	err = admin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	developer := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupDeveloper},
		State:       user.UserStateActive,
	}
	err = developer.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": usr.Username,
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	// Update Self
	randStr = test.RandStr()
	usr.LastName = randStr
	uuu := &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+usr.Username, params, usr, uuu)
	c.Assert(err, check.IsNil)
	c.Assert(uuu, check.NotNil)
	c.Assert(uuu.LastName, check.Equals, randStr)

	// Create Super Admin
	u := &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupSuperAdmin},
	}

	uu := &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Create Admin
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupAdmin},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)

	// Create Developer
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.IsNil)

	// Update Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Update Admin
	randStr = test.RandStr()
	admin.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)
	c.Assert(uu.LastName, check.Equals, randStr)

	// Update Developer
	randStr = test.RandStr()
	developer.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)
	c.Assert(uu.LastName, check.Equals, randStr)

	// Ban/Unban Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Ban/Unban Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.IsNil)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.IsNil)

	// Ban/Unban Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/ban", params, nil, uu)
	c.Assert(err, check.IsNil)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/activate", params, nil, uu)
	c.Assert(err, check.IsNil)

	// Delete Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Delete Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.IsNil)

	// Delete Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.IsNil)

	username := "user" + test.RandStr() + "@company.com"
	// Invite SuperAdmin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupSuperAdmin,
	}
	var results map[string]interface{}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Invite Admin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupAdmin,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.IsNil)
	fmt.Println(err)

	// Invite Developer
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupDeveloper,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUserDeveloper(c *check.C) {
	var err *axerror.AXError

	randStr := "randuser" + test.RandStr()
	usr := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupDeveloper},
		State:       user.UserStateActive,
	}
	err = usr.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	superAdmin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupSuperAdmin},
		State:       user.UserStateActive,
	}
	err = superAdmin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	admin := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupAdmin},
		State:       user.UserStateActive,
	}
	err = admin.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	randStr = "randuser" + test.RandStr()
	developer := user.User{
		ID:          utils.GenerateUUIDv1(),
		Username:    "user@" + randStr,
		LastName:    "user@" + randStr,
		FirstName:   "user@" + randStr,
		AuthSchemes: []string{"native"},
		Groups:      []string{user.GroupDeveloper},
		State:       user.UserStateActive,
	}
	err = developer.ResetPassword("Test@test100", true)
	c.Assert(err, check.IsNil)

	result, err := axopsExternalClient.Post("auth/login",
		map[string]string{
			"username": usr.Username,
			"password": "Test@test100",
		})
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	// Update Self
	randStr = test.RandStr()
	usr.LastName = randStr
	uuu := &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+usr.Username, params, usr, uuu)
	c.Assert(err, check.IsNil)
	c.Assert(uuu, check.NotNil)
	c.Assert(uuu.LastName, check.Equals, randStr)

	// Create Super Admin
	u := &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupSuperAdmin},
	}

	uu := &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Create Admin
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupAdmin},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Create Developer
	u = &user.User{
		Username: "user" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
	}

	uu = &user.User{}
	err, _ = axopsExternalClient.Post2("users", params, u, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Update Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Update Admin
	randStr = test.RandStr()
	admin.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Update Developer
	randStr = test.RandStr()
	developer.LastName = randStr
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Ban/Unban Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+superAdmin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Ban/Unban Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/ban", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+admin.Username+"/activate", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Ban/Unban Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/ban", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	uu = &user.User{}
	err, _ = axopsExternalClient.Put2("users/"+developer.Username+"/activate", params, nil, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Delete Super Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+superAdmin.Username, params, superAdmin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Delete Admin
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+admin.Username, params, admin, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Delete Developer
	uu = &user.User{}
	err, _ = axopsExternalClient.Delete2("users/"+developer.Username, params, developer, uu)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	username := "user" + test.RandStr() + "@company.com"
	// Invite SuperAdmin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupSuperAdmin,
	}
	var results map[string]interface{}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Invite Admin
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupAdmin,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.NotNil)
	fmt.Println(err)

	// Invite Developer
	params = map[string]interface{}{
		"session": session,
		"group":   user.GroupDeveloper,
	}
	err, _ = axopsExternalClient.Post2("users/"+username+"/invite", params, nil, &results)
	c.Assert(err, check.NotNil)
	fmt.Println(err)
}

func (s *S) TestUserCRUDwithTriableClient(c *check.C) {

	c.Skip("The retry is time consuiming.")

	var retryConfig *restcl.RetryConfig = &restcl.RetryConfig{
		Timeout:      time.Minute * 5,
		TriableCodes: []string{},
	}

	uu, err := user.GetUserByName("admin@internal")
	c.Assert(err, check.IsNil)
	c.Assert(uu, check.NotNil)

	err = uu.ResetPassword("Test@test100", false)
	c.Assert(err, check.IsNil)

	result := map[string]interface{}{}

	err, _ = axopsExternalClient.PostWithTimeRetry("auth/login",
		nil,
		map[string]string{
			"username": "admin@internal",
			"password": "Test@test100",
		},
		&result,
		retryConfig,
	)
	c.Assert(err, check.IsNil)
	session := result["session"].(string)

	params := map[string]interface{}{
		"session": session,
	}

	u := &user.User{
		Username: "admin" + test.RandStr() + "@company.com",
		Password: "Test@test100",
		Groups:   []string{user.GroupDeveloper},
		State:    user.UserStateActive,
	}

	// C
	uu = &user.User{}
	err, _ = axopsExternalClient.PostWithTimeRetry("users", params, u, uu, retryConfig)
	c.Assert(err, check.IsNil)
	c.Assert(uu.ID, check.Not(check.Equals), "")
	c.Assert(uu.State, check.Equals, user.UserStateActive)
	c.Assert(uu.AuthSchemes[0], check.Equals, "native")
	c.Assert(uu.Groups[0], check.Equals, user.GroupDeveloper)
	c.Assert(uu.Password, check.Equals, "")

	// R
	copy := &user.User{}
	err, _ = axopsClient.GetWithTimeRetry("users/"+uu.Username, params, copy, retryConfig)
	c.Assert(err, check.IsNil)
	c.Assert(copy.ID, check.Not(check.Equals), "")
	c.Assert(copy.State, check.Equals, user.UserStateActive)
	c.Assert(copy.AuthSchemes[0], check.Equals, "native")
	c.Assert(copy.Groups[0], check.Equals, user.GroupDeveloper)
	c.Assert(copy.Password, check.Equals, "")

	// U
	u = &user.User{}
	uu.State = user.UserStateBanned
	uu.Groups = []string{user.GroupAdmin}
	uu.FirstName = "Docker"
	uu.LastName = "Mac"
	err, _ = axopsExternalClient.PutWithTimeRetry("users/"+uu.Username, params, uu, u, retryConfig)
	c.Assert(err, check.IsNil)
	c.Assert(u.State, check.Equals, user.UserStateBanned)
	c.Assert(u.AuthSchemes[0], check.Equals, "native")
	c.Assert(u.Groups[0], check.Equals, user.GroupAdmin)
	c.Assert(u.Password, check.Equals, "")
	c.Assert(u.FirstName, check.Equals, uu.FirstName)
	c.Assert(u.LastName, check.Equals, uu.LastName)

	// D
	err, _ = axopsClient.DeleteWithTimeRetry("users/"+uu.Username, nil, nil, nil, retryConfig)
	c.Assert(err, check.IsNil)
	err, _ = axopsClient.GetWithTimeRetry("users/"+uu.Username, params, copy, retryConfig)
	c.Assert(err, check.NotNil)
	c.Assert(err.Code, check.Equals, axerror.ERR_API_RESOURCE_NOT_FOUND.Code)
}
