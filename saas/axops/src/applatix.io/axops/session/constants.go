// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package session

import "time"

const SESSION_RETENTION_NS = 72 * time.Hour
const SESSION_EXTEND_DDL_NS = 24 * time.Hour
const SESSION_RETENTION_SEC = SESSION_RETENTION_NS / time.Second
const COOKIE_SESSION_TOKEN = "session_token"
const (
	RestSession = "session"
	RestUserId  = "user_id"
)
