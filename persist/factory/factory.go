package factory

import (
	"context"
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist"
	"github.com/argoproj/argo/persist/s3"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/util/env"
	"github.com/argoproj/argo/util/instanceid"
)

var (
	// time to live - at what offloadTTL an offload becomes old
	// this environment variable allows you to make Argo Workflows delete offloaded data more or less aggressively,
	// useful for testing
	offloadTTL               = env.LookupEnvDurationOr("OFFLOAD_NODE_STATUS_TTL", 5*time.Minute)
	archivedWorkflowGCPeriod = env.LookupEnvDurationOr("ARCHIVED_WORKFLOW_GC_PERIOD", 24*time.Hour)
)

type Persist struct {
	session               sqlbuilder.Database
	OffloadNodeStatusRepo persist.OffloadNodeStatusRepo
	WorkflowArchive       persist.WorkflowArchive
	stopCh                chan struct{}
}

func New(kubeClient kubernetes.Interface, instanceIDService instanceid.Service, namespace string, c *config.PersistConfig, migrate bool) (*Persist, error) {
	out := &Persist{
		OffloadNodeStatusRepo: persist.ExplosiveOffloadNodeStatusRepo,
		WorkflowArchive:       persist.NullWorkflowArchive,
		stopCh:                make(chan struct{}),
	}
	if c == nil {
		return out, nil
	}

	var session sqlbuilder.Database
	var tableName string

	switch c.SQLConfig().(type) {
	case *config.MySQLConfig, *config.PostgreSQLConfig:
		var err error
		session, tableName, err = sqldb.CreateDBSession(kubeClient, namespace, c)
		if err != nil {
			return nil, err
		}
		log.Info("Database Session created successfully")
		if migrate {
			err = sqldb.NewMigrate(session, c.GetClusterName(), tableName).Exec(context.Background())
			if err != nil {
				return nil, err
			}
		}
	}

	secretInterface := kubeClient.CoreV1().Secrets(namespace)

	if c.NodeStatusOffload {
		switch storage := c.GetNodeStatusOffloadConfig().(type) {
		case *config.S3ArtifactRepository:
			x, err := s3.NewOffloadNodeStatusRepo(secretInterface, c.GetClusterName(), *storage, migrate, offloadTTL)
			if err != nil {
				return nil, err
			}
			out.OffloadNodeStatusRepo = x
		case *config.MySQLConfig, *config.PostgreSQLConfig:
			x, err := sqldb.NewOffloadNodeStatusRepo(session, c.GetClusterName(), tableName, offloadTTL)
			if err != nil {
				return nil, err
			}
			out.OffloadNodeStatusRepo = x
		default:
			return nil, fmt.Errorf("no status node offload storage configured: %v", reflect.TypeOf(storage))
		}
	}

	if c.Archive {
		ttl := time.Duration(c.ArchiveTTL)
		switch storage := c.GetArchiveConfig().(type) {
		case *config.S3ArtifactRepository:
			if ttl > 0 {
				log.Error("Archive TTL is not supported for S3 - ignoring")
			}
			x, err := s3.NewWorkflowArchive(secretInterface, c.GetClusterName(), *storage, migrate)
			if err != nil {
				return nil, err
			}
			out.WorkflowArchive = x
		case *config.MySQLConfig, *config.PostgreSQLConfig:
			out.WorkflowArchive = sqldb.NewWorkflowArchive(session, c.GetClusterName(), namespace, instanceIDService, ttl, archivedWorkflowGCPeriod)
		default:
			return nil, fmt.Errorf("no workflow archive configured: %v", reflect.TypeOf(storage))
		}
		go out.WorkflowArchive.Run(out.stopCh)
	}

	log.WithFields(log.Fields{
		"nodeOffloadStatusEnabled": out.OffloadNodeStatusRepo.IsEnabled(),
		"workflowArchiveEnabled":   out.WorkflowArchive.IsEnabled(),
	}).Info()

	return out, nil

}

func (p *Persist) Close() error {
	close(p.stopCh)
	if p.session != nil {
		err := p.session.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
