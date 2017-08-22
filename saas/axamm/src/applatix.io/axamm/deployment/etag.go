package deployment

import "time"

var eTag string = "deployments-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "deployments-" + time.Now().String()
}
