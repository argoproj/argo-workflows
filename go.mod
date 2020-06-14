module github.com/argoproj/argo

go 1.13

require (
	cloud.google.com/go v0.55.0 // indirect
	cloud.google.com/go/storage v1.6.0
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/aliyun/aliyun-oss-go-sdk v2.0.6+incompatible
	github.com/antonmedv/expr v1.8.2
	github.com/argoproj/argo-events v0.15.0
	github.com/argoproj/pkg v0.0.0-20200424003221-9b858eff18a1
	github.com/aws/aws-sdk-go v1.27.1 // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180805212432-9746310a4d31
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.2.0+incompatible
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.7
	github.com/go-openapi/swag v0.19.8 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.0
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
	github.com/klauspost/compress v1.9.7 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/minio/minio-go v6.0.14+incompatible // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b
	github.com/pkg/errors v0.9.1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/common v0.7.0
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
	github.com/stretchr/testify v1.5.1
	github.com/tidwall/gjson v1.3.5
	github.com/valyala/fasttemplate v1.1.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20200422194213-44a606286825
	golang.org/x/net v0.0.0-20200421231249-e086a090c8fd
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200428211428-0c9eba77bc32 // indirect
	google.golang.org/api v0.20.0
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20200317114155-1f3552e48f24
	google.golang.org/grpc v1.28.0
	gopkg.in/ini.v1 v1.54.0 // indirect
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/square/go-jose.v2 v2.5.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.5
	k8s.io/apimachinery v0.17.5
	k8s.io/client-go v0.0.0-20191225075139-73fd2ddc9180
	k8s.io/kube-openapi v0.0.0-20200316234421-82d701f24f9d
	k8s.io/utils v0.0.0-20191218082557-f07c713de883
	sigs.k8s.io/structured-merge-diff v0.0.0-20190525122527-15d366b2352e // indirect
	sigs.k8s.io/yaml v1.1.0
	upper.io/db.v3 v3.6.3+incompatible
)

replace k8s.io/api => k8s.io/api v0.17.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.17.6-beta.0

replace k8s.io/apiserver => k8s.io/apiserver v0.17.5

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.5

replace k8s.io/client-go => k8s.io/client-go v0.17.5

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.5

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.5

replace k8s.io/component-base => k8s.io/component-base v0.17.5

replace k8s.io/cri-api => k8s.io/cri-api v0.17.6-beta.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.5

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.5

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.5

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.5

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.5

replace k8s.io/kubectl => k8s.io/kubectl v0.17.5

replace k8s.io/kubelet => k8s.io/kubelet v0.17.5

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.5

replace k8s.io/metrics => k8s.io/metrics v0.17.5

replace k8s.io/node-api => k8s.io/node-api v0.17.5

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.5

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.5

replace k8s.io/sample-controller => k8s.io/sample-controller v0.17.5

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible

replace k8s.io/code-generator => k8s.io/code-generator v0.17.5
