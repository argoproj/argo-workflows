package common

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"fmt"
	"regexp"

	"applatix.io/axdb"
	"applatix.io/axerror"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gzip"
	"github.com/gocql/gocql"
	"github.com/nu7hatch/gouuid"
)

func GetGZipHandler() func(c *gin.Context) {
	gzip := gzip.Gzip(gzip.BestCompression)
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.RequestURI, "/v1/service/events") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/application/events") ||
			strings.HasPrefix(c.Request.RequestURI, "/v1/deployment/events") {

			c.Next()

		} else {
			gzip(c)
		}
	}
}

func NoCache(c *gin.Context) {
	c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	c.Next()
}

func ValidateCache(c *gin.Context) {
	c.Header("Cache-Control", "max-age=0, must-revalidate")
	c.Next()
}

func GetAxNameSpace() string {
	axNameSpace, _ := os.LookupEnv("AX_NAMESPACE")
	return axNameSpace
}

func GetAxVersion() string {
	axVersion, _ := os.LookupEnv("AX_VERSION")
	return axVersion
}

const (
	ENV_CPU_MULT = "CPU_MULT"
	ENV_MEM_MULT = "MEM_MULT"
)

func GetEnvFloat64(env string, defVal float64) float64 {
	envStr, ok := os.LookupEnv(env)
	if !ok {
		return defVal
	}

	if val, err := strconv.ParseFloat(envStr, 64); err == nil {
		return val
	} else {
		return defVal
	}
}

var DefaultMaxPendingJobs float64 = 200.0

func GetMaxPendingJobs() int64 {
	maxPendingJob, _ := os.LookupEnv("AX_MAX_PENDING_JOBS")
	if maxPendingJob == "" {
		return int64(DefaultMaxPendingJobs * GetEnvFloat64(ENV_MEM_MULT, 1.0))
	}

	num, err := strconv.ParseInt(maxPendingJob, 10, 64)
	if err != nil {
		fmt.Printf("Error parse env %v: %v./n", maxPendingJob, err)
		return int64(DefaultMaxPendingJobs * GetEnvFloat64(ENV_MEM_MULT, 1.0))
	}

	return num
}

var DefaultMaxPendingContainers float64 = 300.0

func GetMaxPendingContainers() int64 {
	maxPendingJob, _ := os.LookupEnv("AX_MAX_PENDING_CONTAINERS")
	if maxPendingJob == "" {
		return int64(DefaultMaxPendingContainers * GetEnvFloat64(ENV_MEM_MULT, 1.0))
	}

	num, err := strconv.ParseInt(maxPendingJob, 10, 64)
	if err != nil {
		fmt.Printf("Error parse env %v: %v./n", maxPendingJob, err)
		return int64(DefaultMaxPendingContainers * GetEnvFloat64(ENV_MEM_MULT, 1.0))
	}

	return num
}

func GetApplicationName() string {
	applicationName, _ := os.LookupEnv("APPLICATION_NAME")
	return applicationName
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

func QueryParameter(c *gin.Context, name string) string {
	valueArray := c.Request.URL.Query()[name]
	if valueArray == nil {
		return ""
	}
	return valueArray[0]
}

// returns 0 on error. 0 is not a valid parameter. 0 also indicates that parameter is not set
func QueryParameterInt(c *gin.Context, name string) int64 {
	valueArray := c.Request.URL.Query()[name]
	if valueArray == nil {
		return 0
	}

	value, err := strconv.ParseInt(valueArray[0], 10, 64)
	if err != nil {
		c.JSON(axdb.RestStatusInvalid, map[string]string{})
		return 0
	}
	return value
}

type RepoBranch struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

func GetContextParams(c *gin.Context, sFields []string, bFields []string, iFields []string, mField []string) (map[string]interface{}, *axerror.AXError) {

	params := make(map[string]interface{})
	luceneSearch := axdb.NewLuceneSearch()

	fields := c.Request.URL.Query().Get("fields")
	if fields != "" {
		params[axdb.AXDBSelectColumns] = strings.Split(fields, ",")
	}

	sorts := c.Request.URL.Query().Get("sort")
	if sorts != "" {
		for _, sort := range strings.Split(sorts, ",") {
			//utils.InfoLog.Printf("[ARTIFACT]: sort = %s", sort)
			reverse := false
			if strings.HasPrefix(sort, "-") {
				reverse = true
				sort = strings.TrimPrefix(sort, "-")
			}
			//utils.InfoLog.Printf("[ARTIFACT]: reverse = %t", reverse)
			luceneSearch.AddSorter(axdb.NewLuceneSorterBase(sort, reverse))

			// Make sure sort field is included in the SELECT fields
			if params[axdb.AXDBSelectColumns] != nil {
				fieldStrs := params[axdb.AXDBSelectColumns].([]string)
				if !StringInSlice(sort, fieldStrs) {
					fieldStrs = append(fieldStrs, sort)
				}
				params[axdb.AXDBSelectColumns] = fieldStrs
			}
		}
	}

	search := c.Request.URL.Query().Get("search")
	if search != "" {
		searchStr := "*" + strings.TrimPrefix(search, "~") + "*"
		searchFieldStrs := c.Request.URL.Query().Get("search_fields")
		var searchFields []string
		if searchFieldStrs != "" {
			searchFields = strings.Split(searchFieldStrs, ",")
		}
		// if search_fields isn't specified from UI, just use the hard-coded sFields;
		// otherwise, just use the search_fields
		if len(searchFields) == 0 {
			searchFields = sFields
		}
		for _, field := range searchFields {
			searchStr = strings.ToLower(searchStr)
			luceneSearch.AddQueryShould(axdb.NewLuceneWildCardFilterBase(field, searchStr))
		}
	}

	for _, fieldName := range sFields {
		field := c.Request.URL.Query().Get(fieldName)
		if field != "" {
			if strings.HasPrefix(field, "~") {
				field = strings.ToLower(field)
				luceneSearch.AddQueryMust(axdb.NewLuceneWildCardFilterBase(fieldName, "*"+strings.TrimPrefix(field, "~")+"*"))
			} else if strings.HasPrefix(field, "[{") {
				continue
			} else if strings.Contains(field, ",") {
				valString := strings.ToLower(strings.Replace(field, ",", "|", -1))
				luceneSearch.AddQueryMust(axdb.NewLuceneRegexpFilterBase(fieldName, valString))
			} else {
				params[fieldName] = field
			}
		}
	}

	for _, fieldName := range bFields {
		field := c.Request.URL.Query().Get(fieldName)
		if field != "" {
			field = strings.ToLower(field)
			switch field {
			case "true":
				params[fieldName] = true
			case "false":
				params[fieldName] = false
			default:
				return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The field %v type is bool, %v is not valid bool representation.", fieldName, field)
			}
		}
	}

	for _, fieldName := range iFields {
		field := c.Request.URL.Query().Get(fieldName)
		if field != "" {
			if strings.Contains(field, ",") {
				valueStrs := strings.Split(field, ",")
				var values []int64
				for _, vStr := range valueStrs {
					val, err := strconv.ParseInt(vStr, 10, 64)
					if err != nil {
						return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The field %v type is int, %v is not valid int representation.", fieldName, vStr)
					}
					values = append(values, val)
				}
				luceneSearch.AddQueryMust(axdb.NewLuceneContainsFilterBase(fieldName, values))
			} else {
				val, err := strconv.ParseInt(field, 10, 64)
				if err != nil {
					return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The field %v type is int, %v is not valid int representation.", fieldName, field)
				}
				params[fieldName] = val
			}
		}
	}

	for _, fieldName := range mField {
		field := c.Request.URL.Query().Get(fieldName)
		if field != "" {
			kvs := strings.Split(field, ";")
			for _, kv := range kvs {
				if strings.Count(kv, ":") != 1 {
					return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("The query string is invalid for field %v : Expecting one ':' as seperator of key and value in %v.", fieldName, kv)
				}

				kvl := strings.Split(kv, ":")
				key := kvl[0]
				valStr := strings.Replace(kvl[1], ",", "|", -1)
				luceneSearch.AddQueryMust(axdb.NewLuceneRegexpFilterBase(fieldName+"$"+key, valStr))
			}
		}
	}

	// special handling of repo_branch
	repoBranch := c.Request.URL.Query().Get("repo_branch")
	if repoBranch != "" {
		if strings.HasPrefix(repoBranch, "[{") {
			branches := []RepoBranch{}
			err := json.Unmarshal([]byte(repoBranch), &branches)
			if err != nil {
				return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Can not unmarshal repo_branch field: %v", err)
			}

			var buffer bytes.Buffer
			for i, _ := range branches {
				buffer.WriteString(strings.ToLower(regexp.QuoteMeta(branches[i].Repo + "_" + branches[i].Branch)))
				if i != len(branches)-1 {
					buffer.WriteString("|")
				}
			}
			delete(params, "repo_branch")
			luceneSearch.AddQueryMust(axdb.NewLuceneRegexpFilterBase("repo_branch", buffer.String()))

			// Make sure repo_branch is included in the SELECT fields
			if params[axdb.AXDBSelectColumns] != nil {
				fieldStrs := params[axdb.AXDBSelectColumns].([]string)
				fieldStrs = append(fieldStrs, "repo_branch")
				fieldStrs = DedupFields(fieldStrs)
				params[axdb.AXDBSelectColumns] = fieldStrs
			}

		}
	}

	if luceneSearch.IsValid() {
		params[axdb.AXDBQuerySearch] = luceneSearch
	}

	limit := QueryParameterInt(c, QueryLimit)
	if limit != 0 {
		params[axdb.AXDBQueryMaxEntries] = limit
	}

	offset := QueryParameterInt(c, QueryOffset)
	if offset != 0 {
		params[axdb.AXDBQueryOffsetEntries] = offset
	}

	return params, nil
}

func GetContextTimeParams(c *gin.Context, params map[string]interface{}) (map[string]interface{}, *axerror.AXError) {

	minTime := QueryParameterInt(c, QueryMinTime)
	maxTime := QueryParameterInt(c, QueryMaxTime)

	if minTime == 0 && maxTime == 0 {
		return params, nil
	}

	search, ok := params[axdb.AXDBQuerySearch]

	if ok {
		luceneSearch := search.(*axdb.LuceneSearch)
		if minTime != 0 || maxTime != 0 {
			luceneSearch.AddQueryMust(axdb.NewLuceneRangeFilterBase(axdb.AXDBTimeColumnName, minTime*1e6, maxTime*1e6))

			// Make sure ax_time is included in the SELECT fields
			if params[axdb.AXDBSelectColumns] != nil {
				fieldStrs := params[axdb.AXDBSelectColumns].([]string)
				fieldStrs = append(fieldStrs, axdb.AXDBTimeColumnName)
				fieldStrs = DedupFields(fieldStrs)
				params[axdb.AXDBSelectColumns] = fieldStrs
			}
		}
		params[axdb.AXDBQuerySearch] = luceneSearch

	} else {
		if minTime != 0 {
			params[axdb.AXDBQueryMinTime] = minTime * 1e6
		}

		if maxTime != 0 {
			params[axdb.AXDBQueryMaxTime] = maxTime * 1e6
		}
	}

	return params, nil
}

func DedupFields(old []string) []string {
	m := make(map[string]bool)

	for _, str := range old {
		str = strings.TrimSpace(str)
		m[str] = true
	}

	new := []string{}

	for k, _ := range m {
		if k != "id" {
			new = append(new, k)
		}
	}

	return new
}

func GetBodyString(c *gin.Context) ([]byte, error) {
	buffer := new(bytes.Buffer)
	_, err := buffer.ReadFrom(c.Request.Body)
	if err != nil {
		return nil, err
	}
	body := buffer.Bytes()
	return body, nil
}

func GetUnmarshalledBody(c *gin.Context, obj interface{}) error {
	body, err := GetBodyString(c)
	if err != nil {
		return err
	}

	jsonErr := json.Unmarshal(body, obj)
	if jsonErr != nil {
		return jsonErr
	}

	return nil
}

func ValidateKubeObjName(name string) bool {
	re := regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?)$`)
	return re.MatchString(name)
}

func ValidateFDQN(name string) bool {
	re := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9])).([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}.[a-zA-Z]{2,3})$`)
	return re.MatchString(name)
}

func ValidateCIDR(name string) bool {
	re := regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}(\/([0-9]|[1-2][0-9]|3[0-2]))?$`)
	return re.MatchString(name)
}

var axPublicDNS, _ = os.LookupEnv("AXOPS_EXT_DNS")

func GetPublicDNS() string {
	return axPublicDNS
}

func Min(a, b int) int {
	if a <= b {
		return a
	} else {
		return b
	}
}

func Max(a, b int) int {
	if a >= b {
		return a
	} else {
		return b
	}
}

func StringInSlice(s string, array []string) bool {
	for _, v := range array {
		if v == s {
			return true
		}
	}
	return false
}
