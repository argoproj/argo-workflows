package fixtures

import (
	"context"

	"github.com/upper/db/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/config"
	persist "github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
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
		tableName, err := persist.GetTableName(persistence)
		if err != nil {
			panic(err)
		}
		log := logging.RequireLoggerFromContext(ctx)
		instanceIDService := instanceid.NewService(wcConfig.InstanceID)
		sessionProxy, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
			KubectlConfig: kubeClient,
			Namespace:     Namespace,
			DBConfig:      persistence.DBConfig,
		})
		if err != nil {
			panic(err)
		}
		offloadNodeStatusRepo, err := persist.NewOffloadNodeStatusRepo(ctx, log, sessionProxy, persistence.GetClusterName(), tableName)
		if err != nil {
			panic(err)
		}
		workflowArchive := persist.NewWorkflowArchive(sessionProxy, persistence.GetClusterName(), Namespace, instanceIDService)
		return &Persistence{workflowArchive, sessionProxy.Session(), offloadNodeStatusRepo}
	}
	return &Persistence{OffloadNodeStatusRepo: persist.ExplosiveOffloadNodeStatusRepo, WorkflowArchive: persist.NullWorkflowArchive}
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
