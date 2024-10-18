package fixtures

import (
	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

type Persistence struct {
	WorkflowArchive       sqldb.WorkflowArchive
	session               db.Session
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func newPersistence(kubeClient kubernetes.Interface, wcConfig *config.Config) *Persistence {
	persistence := wcConfig.Persistence
	if persistence != nil {
		if persistence.PostgreSQL != nil {
			persistence.PostgreSQL.Host = "localhost"
		}
		if persistence.MySQL != nil {
			persistence.MySQL.Host = "localhost"
		}
		session, err := sqldb.CreateDBSession(kubeClient, Namespace, persistence)
		if err != nil {
			panic(err)
		}
		tableName, err := sqldb.GetTableName(persistence)
		if err != nil {
			panic(err)
		}
		offloadNodeStatusRepo, err := sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
		if err != nil {
			panic(err)
		}
		instanceIDService := instanceid.NewService(wcConfig.InstanceID)
		workflowArchive := sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), Namespace, instanceIDService)
		return &Persistence{workflowArchive, session, offloadNodeStatusRepo}
	} else {
		return &Persistence{offloadNodeStatusRepo: sqldb.ExplosiveOffloadNodeStatusRepo, WorkflowArchive: sqldb.NullWorkflowArchive}
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
