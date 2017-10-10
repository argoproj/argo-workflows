// Copyright 2015-2017 Applatix, Inc. All rights reserved.
// @SubApi S3 object download API [/s3]

package axops

import (
	"applatix.io/axops/utils"
	"applatix.io/s3cl"
	"github.com/gin-gonic/gin"
	"io"
)

// @Title GetS3Object
// @Description Get s3 object
// @Accept  json
// @Param   bucket     	 query    string     true        "bucket for  the object"
// @Param   key     	 query    string     true        "key for  the object"
// @Success 200
// @Failure 404
// @Failure 500
// @Resource /s3object
// @Router /s3object [GET]
func GetS3Object() gin.HandlerFunc {
	return func(c *gin.Context) {
		bucket := c.Query("bucket")
		key := c.Query("key")
		output, err := s3cl.GetObjectFromS3(&bucket, &key)

		if err != nil {
			utils.ErrorLog.Printf("Unable to read object with bucket:%v and key:%v from s3 due to %v", bucket, key, err)
			return
		}
		c.Header("Content-Type", *output.ContentType)
		if output.ContentDisposition != nil {
			c.Header("Content-Disposition", *output.ContentDisposition)
		} else {
			c.Header("Content-Disposition", "attachment; filename="+key)
		}
		_, err = io.Copy(c.Writer, output.Body)
		output.Body.Close()
		return
	}
}
