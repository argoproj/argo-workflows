package service

import "time"

var serviceETag string = "services-" + time.Now().String()

func GetServiceETag() string {
	return serviceETag
}

func UpdateServiceETag() {
	serviceETag = "services-" + time.Now().String()
}

var templateETag string = "templates-" + time.Now().String()

func GetTemplateETag() string {
	return templateETag
}

func UpdateTemplateETag() {
	templateETag = "services-" + time.Now().String()
}
