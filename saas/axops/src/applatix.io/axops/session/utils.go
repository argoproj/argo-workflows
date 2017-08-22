// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package session

import (
	"encoding/base64"
	"github.com/gorilla/securecookie"
)

func GenerateSessionID() string {
	return base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
}
