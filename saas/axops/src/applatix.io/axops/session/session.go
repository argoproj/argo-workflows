// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package session

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"encoding/json"
	"fmt"
	"time"
)

const RedisSessionKey = "session-%v"

type Session struct {
	ID       string `json:"id,omitempty"`
	UserID   string `json:"userid,omitempty"`
	Username string `json:"username,omitempty"`
	State    int    `json:"state,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Ctime    int64  `json:"ctime,omitempty"`
	Expiry   int64  `json:"expiry,omitempty"`
}

func (s *Session) Create() (*Session, *axerror.AXError) {
	if s.UserID == "" || s.Username == "" {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Mising user information during session creation.")
	}

	if s.Scheme == "" {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Mising auth scheme information during session creation.")

	}

	s.ID = GenerateSessionID()
	s.Ctime = time.Now().Unix()
	s.Expiry = time.Now().Add(SESSION_RETENTION_NS).Unix()

	if axErr := s.Save(); axErr != nil {
		return nil, axErr
	}

	return s, nil
}

func (s *Session) Reload() (*Session, *axerror.AXError) {
	if s.ID == "" {
		return nil, axerror.ERR_API_AUTH_FAILED.NewWithMessage("Missing session ID.")
	}

	session, axErr := GetSessionById(s.ID)
	if axErr != nil {
		return nil, axErr
	}

	if session == nil {
		return nil, axerror.ERR_API_AUTH_FAILED.NewWithMessagef("Cannot find session with ID: %v", s.ID)
	}

	return session, nil
}

func (s *Session) Delete() *axerror.AXError {
	_, axErr := utils.Dbcl.Delete(axdb.AXDBAppAXOPS, SessionTableName, []*Session{s})
	if axErr != nil {
		utils.ErrorLog.Printf("Delete session failed:%v\n", axErr)
	}
	utils.RedisCacheCl.Del(fmt.Sprintf(RedisSessionKey, s.ID))
	return nil
}

func (s *Session) Validate() *axerror.AXError {
	if s.Expiry > time.Now().Unix() {
		return nil
	} else {
		s.Delete()
		return axerror.ERR_API_EXPIRED_SESSION.New()
	}
}

func (s *Session) Extend() *axerror.AXError {
	//if s.Expiry < time.Now().Add(SESSION_EXTEND_DDL_NS).Unix() && s.Scheme == "native" {
	//	// Update Expiry when going to expire within 24 hours
	//	s.Expiry = time.Now().Add(SESSION_RETENTION_NS).Unix()
	//	return s.Save()
	//}
	return nil
}

func (s *Session) Save() *axerror.AXError {
	if _, axErr := utils.Dbcl.Put(axdb.AXDBAppAXOPS, SessionTableName, s); axErr != nil {
		return axErr
	}
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisSessionKey, s.ID), s, time.Hour)
	return nil
}

func (s *Session) String() string {
	sBytes, _ := json.Marshal(s)
	return string(sBytes)
}

func (s *Session) FromString(sStr string) (*Session, *axerror.AXError) {
	err := json.Unmarshal([]byte(sStr), s)
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}
	return s, nil
}

func GetSessionById(id string) (*Session, *axerror.AXError) {

	s := &Session{}
	if axErr := utils.RedisCacheCl.GetObj(fmt.Sprintf(RedisSessionKey, id), s); axErr == nil {
		utils.DebugLog.Printf("[Cache] cache hit for session with id %v\n", id)
		return s, nil
	}

	sessions, axErr := GetSessions(map[string]interface{}{
		SessionID: id,
	})

	if axErr != nil {
		return nil, axErr
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	s = &sessions[0]
	utils.RedisCacheCl.SetObjWithTTL(fmt.Sprintf(RedisSessionKey, s.ID), s, time.Hour)
	return s, nil
}

func GetSessionByUserID(uid string) ([]Session, *axerror.AXError) {
	sessions, axErr := GetSessions(map[string]interface{}{
		SessionUserId: uid,
	})

	if axErr != nil {
		return nil, axErr
	}

	return sessions, nil

}

func GetSessionByUsername(uname string) ([]Session, *axerror.AXError) {
	sessions, axErr := GetSessions(map[string]interface{}{
		SessionUserName: uname,
	})

	if axErr != nil {
		return nil, axErr
	}

	return sessions, nil
}

func GetSessions(params map[string]interface{}) ([]Session, *axerror.AXError) {
	sessions := []Session{}
	axErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, SessionTableName, params, &sessions)
	if axErr != nil {
		return nil, axErr
	}
	return sessions, nil
}

func DeleteSessionsByUsername(uname string) *axerror.AXError {
	sessions, axErr := GetSessionByUsername(uname)
	if axErr != nil {
		utils.ErrorLog.Printf("Delete session failed:%v", axErr)
		return nil
	}

	for _, s := range sessions {
		axErr = s.Delete()
		if axErr != nil {
			utils.ErrorLog.Printf("Delete session failed:%v", axErr)
		}
	}
	return nil
}
