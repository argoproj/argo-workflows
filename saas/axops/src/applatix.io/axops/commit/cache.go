package commit

import (
	"applatix.io/axops/utils"
	"time"
)

var eTag string = "commits-" + time.Now().String()
var reposUpdated string

func GetETag() string {

	if msg, axErr := utils.RedisCacheCl.GetString("gateway:repos_updated"); axErr == nil {
		if msg == reposUpdated {
			return eTag
		} else {
			reposUpdated = msg
			UpdateETag()
		}
	} else {
		UpdateETag()
	}

	return eTag
}

func UpdateETag() {
	eTag = "commits-" + time.Now().String()
}
