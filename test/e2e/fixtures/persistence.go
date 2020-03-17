package fixtures

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/config"
)

type Persistence struct {
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	workflowArchive       sqldb.WorkflowArchive
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
	if persistence != nil {
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
		offloadNodeStatusRepo, err := sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
		if err != nil {
			panic(err)
		}
		workflowArchive := sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), wcConfig.InstanceID)
		return &Persistence{session, offloadNodeStatusRepo, workflowArchive}
	} else {
		return &Persistence{offloadNodeStatusRepo: sqldb.ExplosiveOffloadNodeStatusRepo, workflowArchive: sqldb.NullWorkflowArchive}
	}
}

func (s *Persistence) IsEnabled() bool {
	return s.offloadNodeStatusRepo.IsEnabled()
}

func (s *Persistence) Close() {
	if s.IsEnabled() {
		err := s.session.Close()
		if err != nil {
			panic(err)
		}
	}
}
