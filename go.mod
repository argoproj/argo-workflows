module github.com/argoproj/argo

go 1.13

require (
	bou.ke/staticfiles v0.0.0-20210106104248-dd04075d4104
	cloud.google.com/go v0.55.0 // indirect
	cloud.google.com/go/storage v1.6.0
	github.com/Azure/go-autorest/autorest v0.11.1 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.5 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/aliyun/aliyun-oss-go-sdk v2.1.5+incompatible
	github.com/antonmedv/expr v1.8.8
	github.com/argoproj/argo-events v0.0.0-00010101000000-000000000000
	github.com/argoproj/pkg v0.3.0
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/blushft/go-diagrams v0.0.0-20201006005127-c78c821223d9
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/fatih/structs v1.1.0 // indirect
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/go-openapi/jsonreference v0.19.5
	github.com/go-openapi/spec v0.20.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-swagger/go-swagger v0.25.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/mattn/goreman v0.3.5
	github.com/minio/minio-go/v7 v7.0.2
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.0
	github.com/valyala/fasttemplate v1.1.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201216054612-986b41b23924
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/tools v0.0.0-20200717024301-6ddee64345a6
	google.golang.org/api v0.20.0
	google.golang.org/genproto v0.0.0-20200806141610-86f49bd18e98
	google.golang.org/grpc v1.33.1
	google.golang.org/grpc/examples v0.0.0-20201226181154-53788aa5dcb4 // indirect
	gopkg.in/go-playground/webhooks.v5 v5.15.0
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v0.19.6
	k8s.io/code-generator v0.19.6
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027 // indirect
	k8s.io/klog/v2 v2.4.0
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0
	upper.io/db.v3 v3.6.3+incompatible
)

replace github.com/argoproj/argo-events => github.com/argoproj/argo-events v1.2.0
