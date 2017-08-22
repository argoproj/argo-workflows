// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package utils

import (
	"applatix.io/axdb/axdbcl"
	axopsutils "applatix.io/axops/utils"
	"applatix.io/common"
	"applatix.io/rediscl"
	"applatix.io/restcl"
	"log"
)

var DbCl *axdbcl.AXDBClient
var AxmonCl *restcl.RestClient
var AmmCl *restcl.RestClient
var FixMgrCl *restcl.RestClient
var AdcCl *restcl.RestClient
var AxopsCl *restcl.RestClient
var RedisAdcCl *rediscl.RedisClient
var RedisSaaSCl *rediscl.RedisClient
var AxNotifierCl *restcl.RestClient

var DebugLog *log.Logger
var InfoLog *log.Logger
var ErrorLog *log.Logger

const (
	RedisAdcDatabase  = 2
	RedisSaaSDatabase = 10
)

// Init the loggers.
func InitLoggers(prefix string) {
	common.InitLoggers(prefix)

	DebugLog = common.DebugLog
	InfoLog = common.InfoLog
	ErrorLog = common.ErrorLog

	axopsutils.DebugLog = common.DebugLog
	axopsutils.InfoLog = common.InfoLog
	axopsutils.ErrorLog = common.ErrorLog
}

func InitAdcRedis() {
	RedisAdcCl = rediscl.NewRedisClient("redis.axsys:6379", "", RedisAdcDatabase)
}

func InitSaaSRedis() {
	RedisSaaSCl = rediscl.NewRedisClient("redis.axsys:6379", "", RedisSaaSDatabase)
}

func NewTrue() *bool {
	b := true
	return &b
}

func NewFalse() *bool {
	b := false
	return &b
}

var APPLICATION_NAME string

func GetStatusDetail(code, message, detail string) map[string]interface{} {
	return map[string]interface{}{
		"code":    code,
		"message": message,
		"detail":  detail,
	}
}

type Event struct {
	Id           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	StatusDetail map[string]interface{} `json:"status_detail"`
}
