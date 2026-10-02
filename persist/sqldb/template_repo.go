package sqldb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/upper/db/v4"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// TemplateOffloadTableName is the dedicated table for offloaded workflow templates.
const TemplateOffloadTableName = "argo_offloaded_workflow_templates"

// TemplateRepo interface for template storage
type TemplateRepo interface {
	SaveTemplates(ctx context.Context, uid, namespace string, templates []wfv1.Template) error
	GetTemplates(ctx context.Context, uid string) ([]wfv1.Template, error)
	DeleteTemplates(ctx context.Context, uid string) error
	IsEnabled() bool
	// ListOldOffloads returns UIDs with template offloads older than the given age.
	// This is used by the template GC to clean up orphaned template rows.
	ListOldOffloads(ctx context.Context, age time.Duration) ([]string, error)
}

// templateRecord represents a row in the argo_offloaded_workflow_templates table
type templateRecord struct {
	ClusterName  string    `db:"clustername"`
	UID          string    `db:"uid"`
	Namespace    string    `db:"namespace"`
	TemplateName string    `db:"template_name"`
	Template     string    `db:"template"`
	CreatedAt    time.Time `db:"createdat"`
}

// templateRepo implements TemplateRepo interface
type templateRepo struct {
	sessionProxy *sqldb.SessionProxy
	clusterName  string
	tableName    string
	enabled      bool
	// per-operation timeout for database operations, bounded via context.WithTimeout
	opTimeout time.Duration
	log       logging.Logger
}

// NewTemplateRepo creates a new template repository
func NewTemplateRepo(ctx context.Context, log logging.Logger, sessionProxy *sqldb.SessionProxy, clusterName, tableName string, opTimeout time.Duration) TemplateRepo {
	return &templateRepo{
		sessionProxy: sessionProxy,
		clusterName:  clusterName,
		tableName:    tableName,
		enabled:      true,
		opTimeout:    opTimeout,
		log:          log,
	}
}

// IsEnabled returns true if template offloading is enabled
func (r *templateRepo) IsEnabled() bool {
	return r.enabled
}

// SaveTemplates saves templates to database
func (r *templateRepo) SaveTemplates(ctx context.Context, uid, namespace string, templates []wfv1.Template) error {
	if !r.enabled {
		return fmt.Errorf("template offloading is disabled")
	}

	if uid == "" {
		return fmt.Errorf("uid is required")
	}

	logCtx := r.log.WithFields(logging.Fields{"uid": uid, "templateCount": len(templates)})
	logCtx.Debug(ctx, "Saving templates to database")

	// Bound the operation by operationTimeoutSeconds so a slow or locked Postgres
	// can't block reconciliation; transaction-abort retries are handled by the layers below.
	opCtx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	return r.sessionProxy.TxWith(opCtx, func(s *sqldb.SessionProxy) error {
		sess := s.Session(ctx)

		// Delete existing templates for this workflow
		_, err := sess.SQL().
			DeleteFrom(r.tableName).
			Where(db.Cond{"clustername": r.clusterName}).
			And(db.Cond{"uid": uid}).
			Exec()
		if err != nil {
			return fmt.Errorf("failed to delete existing templates: %w", err)
		}

		// Batch INSERT in chunks; the driver caps a query at 65535 parameters
		// (PostgreSQL) or placeholders (MySQL), so a 10000-row/chunk batch stays under it.
		const maxBatchSize = 10000
		for chunkStart := 0; chunkStart < len(templates); chunkStart += maxBatchSize {
			chunkEnd := min(chunkStart+maxBatchSize, len(templates))
			chunk := templates[chunkStart:chunkEnd]
			batch := sess.SQL().
				InsertInto(r.tableName).
				Columns("clustername", "uid", "namespace", "template_name", "template", "createdat").
				Batch(len(chunk))

			for _, tmpl := range chunk {
				tmplJSON, err := json.Marshal(tmpl)
				if err != nil {
					batch.Done()
					return fmt.Errorf("failed to marshal template %s: %w", tmpl.Name, err)
				}
				batch.Values(r.clusterName, uid, namespace, tmpl.Name, string(tmplJSON), time.Now())
			}
			batch.Done()
			if err := batch.Wait(); err != nil {
				return fmt.Errorf("failed to batch insert templates (chunk %d-%d): %w", chunkStart, chunkEnd, err)
			}
			logCtx.WithFields(logging.Fields{"chunkStart": chunkStart, "chunkEnd": chunkEnd}).Debug(ctx, "Inserted template chunk")
		}

		logCtx.Debug(ctx, "Templates saved successfully")
		return nil
	}, nil)
}

// DeleteTemplates deletes all offloaded templates for a given workflow UID
func (r *templateRepo) DeleteTemplates(ctx context.Context, uid string) error {
	if !r.enabled {
		return fmt.Errorf("template offloading is disabled")
	}
	if uid == "" {
		return fmt.Errorf("uid is required")
	}
	logCtx := r.log.WithField("uid", uid)
	logCtx.Debug(ctx, "Deleting templates from database")

	opCtx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	return r.sessionProxy.TxWith(opCtx, func(s *sqldb.SessionProxy) error {
		sess := s.Session(ctx)
		_, err := sess.SQL().
			DeleteFrom(r.tableName).
			Where(db.Cond{"clustername": r.clusterName}).
			And(db.Cond{"uid": uid}).
			Exec()
		if err != nil {
			return fmt.Errorf("failed to delete templates: %w", err)
		}
		logCtx.Debug(ctx, "Templates deleted successfully")
		return nil
	}, nil)
}

// ListOldOffloads returns UIDs with template offloads older than the given age.
func (r *templateRepo) ListOldOffloads(ctx context.Context, age time.Duration) ([]string, error) {
	if !r.enabled {
		return nil, fmt.Errorf("template offloading is disabled")
	}

	logCtx := r.log.WithField("age", age)
	logCtx.Debug(ctx, "Listing old template offloads")

	if r.sessionProxy == nil {
		logCtx.Error(ctx, "Session proxy is nil")
		return nil, fmt.Errorf("session proxy is nil")
	}

	var uids []string
	opCtx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	err := r.sessionProxy.With(opCtx, func(s db.Session) error {
		var records []struct {
			UID string `db:"uid"`
		}
		err := s.SQL().
			Select("uid").
			From(r.tableName).
			Where(db.Cond{"clustername": r.clusterName}).
			And(fmt.Sprintf("createdat < current_timestamp - interval '%d' second", int(age.Seconds()))).
			All(&records)
		if err != nil {
			return fmt.Errorf("failed to query old template offloads: %w", err)
		}
		for _, rec := range records {
			uids = append(uids, rec.UID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	logCtx.WithField("count", len(uids)).Debug(ctx, "Found old template offloads")
	return uids, nil
}

// GetTemplates retrieves templates from database
func (r *templateRepo) GetTemplates(ctx context.Context, uid string) ([]wfv1.Template, error) {
	if !r.enabled {
		return nil, fmt.Errorf("template offloading is disabled")
	}

	logCtx := r.log.WithField("uid", uid)
	logCtx.WithFields(logging.Fields{"tableName": r.tableName, "clusterName": r.clusterName}).Debug(ctx, "Getting templates from database")

	// Check session state before querying
	if r.sessionProxy == nil {
		logCtx.Error(ctx, "Session proxy is nil")
		return nil, fmt.Errorf("session proxy is nil")
	}

	var templates []wfv1.Template

	opCtx, cancel := context.WithTimeout(ctx, r.opTimeout)
	defer cancel()
	err := r.sessionProxy.With(opCtx, func(s db.Session) error {
		var records []templateRecord

		err := s.SQL().
			Select("template_name", "template").
			From(r.tableName).
			Where("clustername = ?", r.clusterName).
			And("uid = ?", uid).
			// Order by template name so hydration is deterministic: uid is pinned to one
			// value by the WHERE clause, so template_name is the only varying column (and
			// the PK suffix, riding the index with no sort). The returned slice matches the
			// canonical name-sorted order ComputeTemplateVersion hashes.
			OrderBy("template_name").
			All(&records)
		if err != nil {
			logCtx.WithError(err).Error(ctx, "Query failed")
			return fmt.Errorf("failed to query templates: %w", err)
		}

		logCtx.WithField("recordCount", len(records)).Debug(ctx, "Query returned records")
		for _, record := range records {
			var tmpl wfv1.Template
			if err := json.Unmarshal([]byte(record.Template), &tmpl); err != nil {
				return fmt.Errorf("failed to unmarshal template %s: %w", record.TemplateName, err)
			}
			templates = append(templates, tmpl)
		}

		logCtx.WithField("templateCount", len(templates)).Debug(ctx, "Templates retrieved successfully")
		return nil
	})
	if err != nil {
		logCtx.WithError(err).Error(ctx, "sessionProxy.With failed")
		return nil, err
	}

	return templates, nil
}
