package fixture

import "time"

var eTag string = "fixtures-" + time.Now().String()

func GetETag() string {
	return eTag
}

func UpdateETag() {
	eTag = "fixtures-" + time.Now().String()
}
