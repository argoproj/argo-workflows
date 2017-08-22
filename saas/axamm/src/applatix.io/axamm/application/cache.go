package application

import (
	"time"
)

var eTag string = "apps-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "apps-" + time.Now().String()
}
