package fixtures

import (
	"context"

	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	persist "github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

type Persistence struct {
	WorkflowArchive       persist.WorkflowArchive
	session               db.Session
	OffloadNodeStatusRepo persist.OffloadNodeStatusRepo
}

func NewPersistence(ctx context.Context, kubeClient kubernetes.Interface, wcConfig *config.Config) *Persistence {
	persistence := wcConfig.Persistence
	if persistence != nil {
		if persistence.PostgreSQL != nil {
			persistence.PostgreSQL.Host = "localhost"
		}
		if persistence.MySQL != nil {
			persistence.MySQL.Host = "localhost"
		}
		session, err := sqldb.CreateDBSession(ctx, kubeClient, Namespace, persistence.DBConfig)
		if err != nil {
			panic(err)
		}
		tableName, err := persist.GetTableName(persistence)
		if err != nil {
			panic(err)
		}
		log := logging.RequireLoggerFromContext(ctx)
		offloadNodeStatusRepo, err := persist.NewOffloadNodeStatusRepo(ctx, log, session, persistence.GetClusterName(), tableName)
		if err != nil {
			panic(err)
		}
		instanceIDService := instanceid.NewService(wcConfig.InstanceID)
		workflowArchive := persist.NewWorkflowArchive(session, persistence.GetClusterName(), Namespace, instanceIDService)
		return &Persistence{workflowArchive, session, offloadNodeStatusRepo}
	} else {
		return &Persistence{OffloadNodeStatusRepo: persist.ExplosiveOffloadNodeStatusRepo, WorkflowArchive: persist.NullWorkflowArchive}
	}
}

func (s *Persistence) IsEnabled() bool {
	return s.OffloadNodeStatusRepo.IsEnabled()
}

func (s *Persistence) Close() {
	if s.IsEnabled() {
		err := s.session.Close()
		if err != nil {
			panic(err)
		}
	}
}
