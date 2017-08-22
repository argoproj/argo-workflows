package policy

import "time"

var eTag string = "policies-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "policies-" + time.Now().String()
}
