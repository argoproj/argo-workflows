// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package user

import (
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/base64"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
	"os"
	"regexp"
	"time"
)

const (
	INTERNAL_ADMIN_USER = "admin@internal"
)

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func verifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == nil {
		return true
	} else {
		return false
	}
}

func newSalt() string {
	return base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
}

func checkPasswordStrength(password string) *axerror.AXError {
	if len(password) < 8 {
		return axerror.ERR_API_WEAK_PASSWORD.NewWithMessage("Password must contain at least 8 characters.")
	}

	re := regexp.MustCompile(`([A-Z]+)`)
	if !re.MatchString(password) {
		return axerror.ERR_API_WEAK_PASSWORD.NewWithMessage("Password must contain at least an upper case.")
	}

	re = regexp.MustCompile(`([a-z]+)`)
	if !re.MatchString(password) {
		return axerror.ERR_API_WEAK_PASSWORD.NewWithMessage("Password must contain at least an lower case.")
	}

	re = regexp.MustCompile(`([0-9]+)`)
	if !re.MatchString(password) {
		return axerror.ERR_API_WEAK_PASSWORD.NewWithMessage("Password must contain at least a number.")
	}

	re = regexp.MustCompile(`([!|"|#|$|%|&|'|(|)|*|+|,|\-|.|\/|:|;|<|=|>|?|@|[|\\|\]|\^|_|{|}|~|]+)`)
	if !re.MatchString(password) {
		return axerror.ERR_API_WEAK_PASSWORD.NewWithMessage("Password must contain at least a special character.")
	}

	return nil
}

func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^([^@\s]+)@((?:[-a-z0-9]+\.)+[a-z]{2,})$`)
	return re.MatchString(email)
}

func InitAdminInternalUser() {

	InitGroups()

	// Wait for table is created
	count := 0
	for {
		count++
		_, dbErr := GetUsers(nil)
		if dbErr == nil {
			break
		} else {
			utils.InfoLog.Printf("waiting for user table to be ready ...... count %v error %v", count, dbErr)
			if count > 300 {
				// Give up, marathon would restart this container
				utils.ErrorLog.Printf("user table is not available, exited")
				os.Exit(1)
			}
		}
		time.Sleep(1 * time.Second)
	}

	var axErr *axerror.AXError

	admin, axErr := GetUserByName(INTERNAL_ADMIN_USER)
	if axErr != nil {
		utils.ErrorLog.Printf("Unable to query user table due to:%v\n", axErr)
		os.Exit(1)
	}

	if admin == nil {
		admin = &User{
			ID:          "00000000-0000-0000-0000-000000000001",
			Username:    INTERNAL_ADMIN_USER,
			FirstName:   "Admin",
			LastName:    " ",
			AuthSchemes: []string{"native"},
			State:       UserStateActive,
			Groups:      []string{GroupSuperAdmin},
			Ctime:       time.Now().Unix(),
			Mtime:       time.Now().Unix(),
		}
		password := utils.GenerateRandomPassword()
		utils.DebugLog.Printf("creating admin internal user:%v\n", axErr)
		axErr = admin.ResetPassword(password, false)
		if axErr != nil {
			utils.ErrorLog.Printf("Failed to reset password for admin internal user:%v\n", axErr)
			os.Exit(1)
		}

	}
}

func ResetAdminInternalPassword() (string, *axerror.AXError) {

	admin, axErr := GetUserByName(INTERNAL_ADMIN_USER)

	if axErr != nil {
		utils.ErrorLog.Printf("Unable to query user table due to:%v\n", axErr)
		return "", axErr
	}

	if admin == nil {
		utils.ErrorLog.Printf("admin internal user not created yet\n")
		return "", axerror.ERR_API_RESOURCE_NOT_FOUND.NewWithMessage("admin internal user not found")
	}

	password := utils.GenerateRandomPassword()
	axErr = admin.ResetPassword(password, false)
	if axErr != nil {
		utils.ErrorLog.Printf("Failed to reset password for admin internal user due to:%v\n", axErr)
		return "", axErr
	}
	return password, nil
}

func InitGroups() {

	// Wait for table is created
	count := 0
	for {
		count++
		_, dbErr := GetGroups(nil)
		if dbErr == nil {
			break
		} else {
			utils.InfoLog.Printf("waiting for user group table to be ready ...... count %v error %v", count, dbErr)
			if count > 300 {
				// Give up, marathon would restart this container
				utils.ErrorLog.Printf("user group table is not available, exited")
				os.Exit(1)
			}
		}
		time.Sleep(1 * time.Second)
	}

	superAdminGroup := Group{
		ID:   "00000000-0000-0000-0000-000000000000",
		Name: GroupSuperAdmin,
	}
	axErr := superAdminGroup.Update()
	if axErr != nil {
		utils.ErrorLog.Println("Faild to create the super admin group:", axErr)
		os.Exit(1)
	} else {
		utils.InfoLog.Println("Loaded the default super admin group")
	}

	adminGroup := Group{
		ID:   "00000000-0000-0000-0000-000000000001",
		Name: GroupAdmin,
	}
	axErr = adminGroup.Update()
	if axErr != nil {
		utils.ErrorLog.Println("Faild to create the admin group:", axErr)
		os.Exit(1)
	} else {
		utils.InfoLog.Println("Loaded the default admin group")
	}

	developerGroup := Group{
		ID:   "00000000-0000-0000-0000-000000000002",
		Name: GroupDeveloper,
	}
	axErr = developerGroup.Update()
	if axErr != nil {
		utils.ErrorLog.Println("Faild to create the developer group:", axErr)
		os.Exit(1)
	} else {
		utils.InfoLog.Println("Loaded the default developer group")
	}
}
