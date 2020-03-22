package sqldb

type tables struct {
	archivedWorkflows       string // argo_archived_workflows
	archivedWorkflowsLabels string // argo_workflow_history_labels
	schemaHistory           string // schema_history
	workflows               string // tableName
	workflowsHistory        string // argo_workflow_history
}

type indexes struct {
	idxName string // idx_name
}

type dbModel struct {
	schema    string
	tableName string
	tables    tables
	indexes   indexes
}

func NewDBModel(schema string, tableName string) dbModel {
	t := tables{
		archivedWorkflows:       "argo_archived_workflows",
		archivedWorkflowsLabels: "argo_archived_workflows_labels",
		schemaHistory:           "schema_history",
		workflows:               tableName,
		workflowsHistory:        "argo_workflow_history",
	}

	i := indexes{idxName: "idx_name"}

	dbm := dbModel{schema: schema, tableName: tableName, tables: t, indexes: i}
	dbm.parseSchema(schema)
	return dbm
}

func (dbm *dbModel) parseSchema(schemaName string) {
	if schemaName != "public" && schemaName != "" {
		dbm.tables.archivedWorkflows = schemaName + "." + dbm.tables.archivedWorkflows
		dbm.tables.archivedWorkflowsLabels = schemaName + "." + dbm.tables.archivedWorkflowsLabels
		dbm.tables.schemaHistory = schemaName + "." + dbm.tables.schemaHistory
		dbm.tables.workflows = schemaName + "." + dbm.tables.workflows
		dbm.tables.workflowsHistory = schemaName + "." + dbm.tables.workflowsHistory
		dbm.indexes.idxName = schemaName + "." + dbm.indexes.idxName
	}
}
