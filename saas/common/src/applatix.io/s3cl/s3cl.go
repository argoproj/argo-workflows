package s3cl

import (
	"applatix.io/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"net/http"
	"os"
)

var s3Region, _ = os.LookupEnv("AX_REGION")
var s3EndPoint, _ = os.LookupEnv("ARGO_S3_ENDPOINT")
var s3AccessKeyId, _ = os.LookupEnv("ARGO_S3_ACCESS_KEY_ID")
var s3AccessKeySecret, _ = os.LookupEnv("ARGO_S3_ACCESS_KEY_SECRET")

func getS3(bucket string) (*s3.S3, error) {

	if len(s3EndPoint) == 0 {
		common.DebugLog.Println("creating aws s3 session")
		return getAwsS3(bucket)
	} else {
		var forcePath = true
		var cred *credentials.Credentials
		if len(s3AccessKeyId) == 0{
			cred = credentials.AnonymousCredentials
		}else{
			cred = credentials.NewStaticCredentials(s3AccessKeyId, s3AccessKeySecret, "")
		}
		sess := session.New(&aws.Config{
			Endpoint:         &s3EndPoint,
			Region:           &s3Region,
			Credentials:      cred,
			S3ForcePathStyle: &forcePath,
		})
		return s3.New(sess), nil
	}
}

func getAwsS3(bucket string) (*s3.S3, error) {
	myRegion := aws.String(s3Region)
	tr := &http.Transport{
		DisableCompression: true,
	}

	client := &http.Client{
		Transport: tr,
	}

	svc := s3.New(session.New(&aws.Config{
		Region:     myRegion,
		HTTPClient: client,
	}))

	resp, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
	if err != nil {
		common.InfoLog.Printf("Unable to get s3 bucket location due to error:%v\n", err)
		return nil, err
	}

	var bucketRegion string
	if resp.LocationConstraint == nil {
		bucketRegion = "us-east-1"
	} else {
		bucketRegion = *resp.LocationConstraint
	}
	common.InfoLog.Printf("bucket region: %s\n", bucketRegion)
	if bucketRegion != *myRegion {
		svc = s3.New(session.New(&aws.Config{
			Region:     &bucketRegion,
			HTTPClient: client,
		}))
	}

	return svc, nil
}

func GetObjectFromS3(bucket *string, key *string) (*s3.GetObjectOutput, error) {
	svc, err := getS3(*bucket)
	if err != nil {
		return nil, err
	}
	return svc.GetObject(&s3.GetObjectInput{
		Bucket: bucket,
		Key:    key,
	})
}
