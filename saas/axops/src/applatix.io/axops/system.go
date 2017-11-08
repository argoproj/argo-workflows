// Copyright 2015-2016 Applatix, Inc. All rights reserved.
// @SubApi System API [/system]
package axops

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"applatix.io/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/url"
)

func SystemStatusHandler(c *gin.Context) {
	resultMap := map[string]int64{RestBuildWait: int64(rand.Float32() * 0), RestTestWait: int64(rand.Float32() * 0)}
	c.JSON(axdb.RestStatusOK, resultMap)
}

type DnsName struct {
	DnsName string `json:"dnsname"`
}

// @Title GetSystemDnsName
// @Description Get system host name
// @Accept  json
// @Success 200 {object} DnsName
// @Resource /system
// @Router /system/settings/dnsname [GET]
func GetDnsName() gin.HandlerFunc {
	return func(c *gin.Context) {
		dnsname := DnsName{
			DnsName: common.GetPublicDNS(),
		}
		c.JSON(axerror.REST_STATUS_OK, dnsname)
		return
	}
}

// @Title SetSystemDnsName
// @Description Set system host name
// @Accept  json
// @Param   hostname     body    DnsName   true         "DNS name object."
// @Success 200 {object} DnsName
// @Failure 400 {object} axerror.AXError "Bad request body"
// @Resource /system
// @Router /system/settings/dnsname [PUT]
func SetDnsName() gin.HandlerFunc {
	return func(c *gin.Context) {
		dnsname := DnsName{}
		err := utils.GetUnmarshalledBody(c, &dnsname)
		if err != nil {
			c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_REQ.New())
			return
		}

		_, axErr := utils.AxmonCl.Put("dnsname", dnsname)
		if axErr != nil {
			c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.New())
			return
		}

		c.JSON(axerror.REST_STATUS_OK, dnsname)
		return
	}
}

type SpotInstanceConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func AxmonSpotInstanceConfigProxy() gin.HandlerFunc {
	axmonURL := utils.AxmonCl.GetRootUrl() + "/cluster/spot_instance_config"
	url, err := url.Parse(axmonURL)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the axmon url: %v", axmonURL))
	}
	fmt.Println(axmonURL)
	return gin.WrapH(NewSingleHostReverseProxy(url))
}

// @Title GetSystemSpotInstanceConfig
// @Description Get system spot instance configuration
// @Accept  json
// @Success 200 {object} SpotInstanceConfig
// @Resource /system
// @Router /system/settings/spot_instance_config [GET]
func GetSpotInstanceConfig() gin.HandlerFunc {
	return AxmonSpotInstanceConfigProxy()
}

// @Title SetSystemSpotInstanceConfig
// @Description Set system spot instance configuration
// @Accept  json
// @Param   config       body    SpotInstanceConfig   true         "Spot instance configuration object."
// @Success 200 {object} SpotInstanceConfig
// @Resource /system
// @Router /system/settings/spot_instance_config [PUT]
func SetSpotInstanceConfig() gin.HandlerFunc {
	return AxmonSpotInstanceConfigProxy()
}

func AxmonSecurityGroupsConfigProxy() gin.HandlerFunc {
	axmonURL := utils.AxmonCl.GetRootUrl() + "/cluster/security_groups"
	url, err := url.Parse(axmonURL)
	if err != nil {
		panic(fmt.Sprintf("Can not parse the axmon url: %v", axmonURL))
	}
	fmt.Println(axmonURL)
	return gin.WrapH(NewSingleHostReverseProxy(url))
}

type SecurityGroupsConfig struct {
	TrustedCidrs []string `json:"trusted_cidrs,omitempty"`
}

// @Title GetSecurityGroupsConfig
// @Description Get security groups configuration
// @Accept  json
// @Success 200 {object} SecurityGroupsConfig
// @Resource /system
// @Router /system/settings/security_groups_config [GET]
func GetSecurityGroupsConfig() gin.HandlerFunc {
	return AxmonSecurityGroupsConfigProxy()
}

// @Title SetSecurityGroupsConfig
// @Description Set security groups configuration
// @Accept  json
// @Param   config       body    SecurityGroupsConfig   true         "Security groups configuration object."
// @Success 200 {object} SecurityGroupsConfig
// @Resource /system
// @Router /system/settings/security_groups_config [PUT]
func SetSecurityGroupsConfig() gin.HandlerFunc {
	return AxmonSecurityGroupsConfigProxy()
}

type SystemVersion struct {
	Namespace   string `json:"namespace"`
	Version     string `json:"version"`
	ClusterID   string `json:"cluster_id"`
	FeaturesSet string `json:"features_set"`
}

// @Title GetSystemVersion
// @Description Get system version
// @Accept  json
// @Success 200 {object} SystemVersion
// @Resource /system
// @Router /system/version [GET]
func GetSystemVersion() gin.HandlerFunc {
	systemVersion := SystemVersion{
		Namespace:   common.GetAxNameSpace(),
		Version:     utils.Version,
		ClusterID:   utils.GetClusterId(),
		FeaturesSet: utils.GetFeaturesSet(),
	}
	return func(c *gin.Context) {
		c.JSON(axerror.REST_STATUS_OK, systemVersion)
		return
	}
}
