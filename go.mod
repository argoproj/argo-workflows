module github.com/argoproj/argo-server/v3

go 1.13

require (
	bou.ke/staticfiles v0.0.0-20190225145250-827d7f6389cd
	github.com/antonmedv/expr v1.8.2
	github.com/argoproj/argo v0.0.0-20201215170742-300db5e628be
	github.com/argoproj/pkg v0.3.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.8
	github.com/go-swagger/go-swagger v0.23.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/mattn/goreman v0.3.5
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.6.0
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200630154851-b2d8b0336632
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.29.1
	gopkg.in/go-playground/webhooks.v5 v5.15.0
	gopkg.in/square/go-jose.v2 v2.4.1
	k8s.io/api v0.17.8
	k8s.io/apimachinery v0.17.8
	k8s.io/client-go v0.17.8
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/grpc-ecosystem/grpc-gateway => github.com/grpc-ecosystem/grpc-gateway v1.12.2
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
)
