module github.com/argoproj/argo

go 1.13

require (
	bou.ke/staticfiles v0.0.0-20190225145250-827d7f6389cd
	cloud.google.com/go v0.55.0 // indirect
	cloud.google.com/go/storage v1.6.0
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/aliyun/aliyun-oss-go-sdk v2.0.6+incompatible
	github.com/antonmedv/expr v1.8.2
	github.com/argoproj/pkg v0.0.0-20200424003221-9b858eff18a1
	github.com/aws/aws-sdk-go v1.33.0 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20180725130230-947c36da3153 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fatih/structs v1.1.0 // indirect
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.8
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-swagger/go-swagger v0.23.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/gophercloud/gophercloud v0.7.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.2
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/klauspost/compress v1.9.7 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/mattn/goreman v0.3.5
	github.com/minio/minio-go v6.0.14+incompatible // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.3.5
	github.com/valyala/fasttemplate v1.1.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200630154851-b2d8b0336632
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.28.0
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/api v0.17.5
	k8s.io/apimachinery v0.17.5
	k8s.io/client-go v0.17.5
	k8s.io/code-generator v0.17.5
	k8s.io/kube-openapi v0.0.0-20200316234421-82d701f24f9d
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/controller-tools v0.0.0-00010101000000-000000000000
	sigs.k8s.io/yaml v1.2.0
	upper.io/db.v3 v3.6.3+incompatible
)

replace sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
