// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"applatix.io/axdb"
	"applatix.io/axdb/axdbcl"
	"applatix.io/axerror"
	"applatix.io/common"
	"applatix.io/rediscl"
	"applatix.io/restcl"
	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/nu7hatch/gouuid"
)

var Dbcl *axdbcl.AXDBClient
var AxmonCl *restcl.RestClient
var AxNotifierCl *restcl.RestClient
var FixMgrCl *restcl.RestClient
var DevopsCl *restcl.RestClient
var WorkflowAdcCl *restcl.RestClient
var SchedulerCl *restcl.RestClient
var ArtifactCl *restcl.RestClient
var RedisCacheCl *rediscl.RedisClient
var AxammCl *restcl.RestClient

const (
	RedisCachingDatabase = 10
)

var DebugLog *log.Logger
var InfoLog *log.Logger
var ErrorLog *log.Logger

// Version information
var (
	Version = "unknown"
)

type MapType map[string]string

// Init the loggers.
func InitLoggers() {
	// Log to stdout during development. Later switch to log to syslog.
	DebugLog = log.New(os.Stdout, "[AXOPS-debug] ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLog = log.New(os.Stdout, "[AXOPS--info] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(os.Stdout, "[AXOPS-error] ", log.Ldate|log.Ltime|log.Lshortfile)
}

const (
	RestData = "data"
)

func getBodyString(c *gin.Context) ([]byte, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(c.Request.Body)
	if err != nil {
		return nil, err
	}
	body := buffer.Bytes()
	return body, nil
}

func GetUnmarshalledBody(c *gin.Context, obj interface{}) error {
	body, err := getBodyString(c)
	if err != nil {
		return err
	}

	jsonErr := json.Unmarshal(body, obj)
	if jsonErr != nil {
		return jsonErr
	}

	return nil
}

func GenerateUUIDv1() string {
	return gocql.TimeUUID().String()
}

func GenerateUUIDv5(name string) string {
	ns := uuid.NamespaceOID
	u, err := uuid.NewV5(ns, []byte(name))
	if err != nil {
		panic("Can not create UUID v5 with string" + name)
	}
	return u.String()
}

var rxUUID = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// IsUUID check if the string is a UUID (version 3, 4 or 5).
func IsUUID(str string) bool {
	return rxUUID.MatchString(str)
}

var NullMap = axdb.AXMap{}
var NullMapArray = []axdb.AXMap{}

var axClusterId, _ = os.LookupEnv("AX_CLUSTER")
var axRegion, _ = os.LookupEnv("AX_REGION")
var axFeaturesSet, _ = os.LookupEnv("ARGO_FEATURES_SET")

//var axPublicIP, _ = os.LookupEnv("AX_CLUSTER_PUBLIC_IP")

func GetClusterId() string {
	return axClusterId
}

func GetRegion() string {
	return axRegion
}

func GetFeaturesSet() string {
	if len(axFeaturesSet) == 0{
		return "full"
	}
	return axFeaturesSet
}

func GetEntityID() string {
	return "https://" + common.GetPublicDNS()
}

func GetSSOURL() string {
	return "https://" + common.GetPublicDNS() + "/v1/auth/saml/consume"
}

func GetPublicCertPath() string {
	return "/axops/cert.pem"
}

func GetPrivateKeyPath() string {
	return "/axops/key.pem"
}

func WriteToFile(content, path string) *axerror.AXError {
	err := ioutil.WriteFile(path, []byte(content), 0777)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to write the file:%v", err)
	}
	return nil
}

func ReadFromFile(path string) (string, *axerror.AXError) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to read the file:%v", err)
	}
	return string(content), nil
}

func PackMessage(key string, op string, payload interface{}) interface{} {
	data := map[string]interface{}{
		"key": key, "value": map[string]interface{}{
			"Op": op, "Payload": payload,
		},
	}

	records := map[string]interface{}{
		"records": []map[string]interface{}{data},
	}

	return records
}

func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^([^@\s]+)@((?:[-a-z0-9]+\.)+[a-z]{2,})$`)
	return re.MatchString(email)
}

func GetUserEmail(author string) string {
	author = strings.ToLower(strings.TrimSpace(author))

	if author == "" {
		return ""
	}

	re := regexp.MustCompile(".*<(.*@.*)>$")
	matches := re.FindStringSubmatch(author)
	if len(matches) != 2 {
		if ValidateEmail(author) {
			return author
		}
	} else {
		if ValidateEmail(matches[1]) {
			return matches[1]
		}
	}

	return ""
}

func NewTrue() *bool {
	b := true
	return &b
}

func NewFalse() *bool {
	b := false
	return &b
}

func NewString(s string) *string {
	return &s
}

func ParseRepoURL(repo string) (owner, name string) {
	if repo == "" {
		return "", ""
	}

	re := regexp.MustCompile(".*[/|:](.+)/(.+).git$")
	matches := re.FindStringSubmatch(repo)
	if len(matches) != 3 {
		return repo, ""
	} else {
		return matches[1], matches[2]
	}
}

func DedupStringList(old []string) []string {
	m := make(map[string]bool)

	for _, str := range old {
		str = strings.TrimSpace(str)
		m[str] = true
	}

	new := []string{}

	for k, _ := range m {
		new = append(new, k)
	}

	return new
}

func GenerateHashFromDNS(dns string) int64 {
	h := fnv.New32a()
	h.Write([]byte(dns))
	return int64(h.Sum32())
}

func GenerateSelfSignedCert() (crt, key string) {
	// ok, lets populate the certificate with some data
	// not all fields in Certificate will be populated
	// see Certificate structure at
	// http://golang.org/pkg/crypto/x509/#Certificate
	template := &x509.Certificate{
		IsCA: true,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{1, 2, 3},
		//SerialNumber:          big.NewInt(1234),
		SerialNumber: big.NewInt(GenerateHashFromDNS(common.GetPublicDNS())),
		Subject: pkix.Name{
			Country:      []string{"United States"},
			Organization: []string{"Applatix Inc."},
		},
		NotBefore: time.Now(),
		// Make the certificate to be valid for the following 4 years
		NotAfter: time.Now().AddDate(4, 0, 0),
		// see http://golang.org/pkg/crypto/x509/#KeyUsage
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		SignatureAlgorithm: x509.SHA512WithRSA,
	}

	// generate private key
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		fmt.Println(err)
	}

	publickey := &privatekey.PublicKey

	cert, err := x509.CreateCertificate(rand.Reader, template, template, publickey, privatekey)

	if err != nil {
		fmt.Println(err)
	}

	keyB := &bytes.Buffer{}
	var pemkey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privatekey)}
	pem.Encode(keyB, pemkey)
	key = keyB.String()

	crtB := &bytes.Buffer{}
	var pemCrt = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert}
	pem.Encode(crtB, pemCrt)
	crt = crtB.String()

	return
}

func GetErrorURL(code int, err *axerror.AXError) string {
	return fmt.Sprintf("https://%v/error/%v/type/%v;msg=%v", common.GetPublicDNS(), code, err.Code, url.QueryEscape(err.Message))
}

func GetParamsFromString(str string, sep string) []string {
	params := []string{}
	str = strings.TrimSpace(str)
	parts := strings.Split(str, sep)
	start := 1
	for i := start; i < len(parts)-1; i = i + 2 {
		params = append(params, parts[i])
	}

	return params
}

func CopyMap(data map[string]interface{}) map[string]interface{} {
	newData := make(map[string]interface{})
	if data != nil {
		for k, v := range data {
			newData[k] = v
		}
	}
	return newData
}

func GenerateRandomPassword() string {
	return uniuri.NewLen(24)
}
