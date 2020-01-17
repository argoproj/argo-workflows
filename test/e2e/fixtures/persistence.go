package fixtures

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/workflow/config"
)

type Persistence struct {
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func newPersistence(kubeClient kubernetes.Interface) *Persistence {
	cm, err := kubeClient.CoreV1().ConfigMaps(Namespace).Get("workflow-controller-configmap", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	wcConfig := &config.WorkflowControllerConfig{}
	err = yaml.Unmarshal([]byte(cm.Data["config"]), wcConfig)
	if err != nil {
		panic(err)
	}
	persistence := wcConfig.Persistence
	if persistence.PostgreSQL != nil {
		persistence.PostgreSQL.Host = "localhost"
	}
	if persistence.MySQL != nil {
		persistence.MySQL.Host = "localhost"
	}
	session, tableName, err := sqldb.CreateDBSession(kubeClient, Namespace, persistence)
	if err != nil {
		panic(err)
	}
	offloadNodeStatusRepo := sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
	return &Persistence{session, offloadNodeStatusRepo}
}

func (s *Persistence) OffloadedCount() int {
	count, err := s.session.Collection("argo_workflows").Find().Count()
	if err != nil {
		panic(err)
	}
	return int(count)
}

func (s *Persistence) Close() {
	err := s.session.Close()
	if err != nil {
		panic(err)
	}
}

func (s *Persistence) DeleteEverything() {
	_, err := s.session.DeleteFrom("argo_workflows").Exec()
	if err != nil {
		panic(err)
	}
	_, err = s.session.DeleteFrom("argo_archived_workflows").Exec()
	if err != nil {
		panic(err)
	}
}
