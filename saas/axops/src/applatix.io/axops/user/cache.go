package user

import "time"

var eTag string = "users-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "users-" + time.Now().String()
}
