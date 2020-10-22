package fixtures

import (
	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/util/instanceid"
)

type Persistence struct {
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	workflowArchive       sqldb.WorkflowArchive
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
		session, tableName, err := sqldb.CreateDBSession(kubeClient, Namespace, persistence)
		if err != nil {
			panic(err)
		}
		offloadNodeStatusRepo, err := sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
		if err != nil {
			panic(err)
		}
		instanceIDService := instanceid.NewService(wcConfig.InstanceID)
		workflowArchive := sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), Namespace, instanceIDService)
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
