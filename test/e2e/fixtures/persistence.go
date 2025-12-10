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
	offloadNodeStatusRepo persist.OffloadNodeStatusRepo
}

// newPersistence creates a Persistence configured from the provided Argo Workflows
// config and Kubernetes client.
//
// When wcConfig.Persistence is non-nil it configures DB hosts for local testing,
// creates a DB session and a session proxy, constructs an instance ID service,
// an offload node status repository, and a workflow archive, and returns a
// Persistence containing those components and the DB session. If any setup step
// fails while persistence is configured the function panics. If wcConfig.Persistence
// is nil it returns a Persistence with persist.ExplosiveOffloadNodeStatusRepo and
// persist.NullWorkflowArchive.
func newPersistence(ctx context.Context, kubeClient kubernetes.Interface, wcConfig *config.Config) *Persistence {
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
		return &Persistence{workflowArchive, session, offloadNodeStatusRepo}
	} else {
		return &Persistence{offloadNodeStatusRepo: persist.ExplosiveOffloadNodeStatusRepo, WorkflowArchive: persist.NullWorkflowArchive}
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