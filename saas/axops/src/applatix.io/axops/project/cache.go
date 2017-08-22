package project

import "time"

var eTag string = "projects-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "projects-" + time.Now().String()
}
