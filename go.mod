module github.com/argoproj/argo

go 1.13

require (
	cloud.google.com/go v0.55.0 // indirect
	cloud.google.com/go/storage v1.6.0
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/ajg/form v1.5.1 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v2.0.6+incompatible
	github.com/argoproj/pkg v0.0.0-20200318225345-d3be5f29b1a8
	github.com/aws/aws-sdk-go v1.27.1 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/fasthttp-contrib/websocket v0.0.0-20160511215533-1f3b11f56072 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.5
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.7.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.1
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/klauspost/compress v1.9.7 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/minio/minio-go v6.0.14+incompatible // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/moul/http2curl v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/common v0.7.0 // indirect
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/robfig/cron v1.2.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v0.0.4-0.20181021141114-fe5e611709b0
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.3.5
	github.com/valyala/fasthttp v0.0.0-20171207120941-e5f51c11919d // indirect
	github.com/valyala/fasttemplate v1.1.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0 // indirect
	github.com/yudai/gojsondiff v0.0.0-20170107030110-7b1b7adf999d // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200406144418-7db14c95bfa9 // indirect
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200317114155-1f3552e48f24
	google.golang.org/grpc v1.28.0
	gopkg.in/gavv/httpexpect.v2 v2.0.0
	gopkg.in/ini.v1 v1.52.0 // indirect
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/api v0.0.0-20191219150132-17cfeff5d095
	k8s.io/apimachinery v0.16.7-beta.0
	k8s.io/client-go v0.0.0-20191225075139-73fd2ddc9180
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/utils v0.0.0-20191218082557-f07c713de883
	sigs.k8s.io/yaml v1.1.0
	upper.io/db.v3 v3.6.3+incompatible
)
