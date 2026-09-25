package main

import (
	"fmt"
	"os"
	"strings"

	persistsqldb "github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

var dbTypes = []sqldb.DBType{sqldb.MySQL, sqldb.Postgres}

type migrationSection struct {
	title   string
	changes func(sqldb.DBType) []sqldb.Change
}

var migrationSections = []migrationSection{
	{
		title: "Archive Database",
		changes: func(dbType sqldb.DBType) []sqldb.Change {
			return persistsqldb.MigrateChanges("<cluster-name>", "argo_workflows", dbType)
		},
	},
	{
		title: "Sync Database",
		changes: func(_ sqldb.DBType) []sqldb.Change {
			return syncdb.MigrateChanges(&syncdb.Config{
				LimitTable:      "sync_limit",
				StateTable:      "sync_state",
				ControllerTable: "sync_controller",
				LockTable:       "sync_lock",
			})
		},
	},
}

const docHeader = `# Database Migrations

This page lists the SQL migrations that Argo Workflows applies automatically at startup.
Migrations are applied incrementally and tracked by a version table.
Each migration is only run once. Do not re-order or remove entries.

The table names and cluster name shown here are defaults.
Your deployment may use different values depending on your configuration.

See [Workflow Archive](workflow-archive.md) and [Synchronization](synchronization.md) for configuration details.

## Steps

Each migration is numbered as a ` + "`Step`" + `. When Argo Workflows runs the automatic migration at controller startup, it records the highest applied step number in a version table (` + "`schema_history`" + ` for the archive database,` + "`sync_schema_history`" + ` for the sync database) and only runs steps with a higher number on subsequent starts.
Steps may be missing where the step does nothing for the database type.
This means steps must never be re-ordered or removed — a new schema change is always appended as a new step.

If you run these statements yourself (for example, because you set ` + "`skipMigration: true`" + ` or want to manually create the schema), you are responsible for tracking which steps your database is already at.
Argo Workflows will not detect partial manual application — it only reads the version table.
If the version table is out of sync with the actual schema, the controller may try to re-apply steps and fail, or skip steps that you have not run.

Programmatic migrations are not described here and the code needs to be consulted for those steps.
`

func main() {
	var sb strings.Builder

	sb.WriteString(docHeader)

	for _, section := range migrationSections {
		fmt.Fprintf(&sb, "## %s\n\n", section.title)
		for _, dt := range dbTypes {
			fmt.Fprintf(&sb, "### %s\n\n", dbTypeTitle(dt))
			sb.WriteString("```sql\n")
			changes := section.changes(dt)
			for i, change := range changes {
				writeChange(&sb, i, dt, change)
			}
			sb.WriteString("```\n\n")
		}
	}

	filename := "docs/database-migrations.md"
	if err := os.WriteFile(filename, []byte(sb.String()), 0o666); err != nil {
		panic(err)
	}
	fmt.Printf("Wrote %s\n", filename)
}

func dbTypeTitle(dt sqldb.DBType) string {
	switch dt {
	case sqldb.MySQL:
		return "MySQL"
	case sqldb.Postgres:
		return "PostgreSQL"
	default:
		return string(dt)
	}
}

func writeChange(sb *strings.Builder, index int, dbType sqldb.DBType, change sqldb.Change) {
	switch c := change.(type) {
	case sqldb.AnsiSQLChange:
		writeSQLBlock(sb, index, string(c))
	case sqldb.TypedChange:
		variant, ok := c.Changes[dbType]
		if !ok {
			return
		}
		if sql, ok := variant.(sqldb.AnsiSQLChange); ok {
			writeSQLBlock(sb, index, string(sql))
		} else {
			writeProgrammaticBlock(sb, index, variant)
		}
	default:
		writeProgrammaticBlock(sb, index, change)
	}
}

func writeStep(sb *strings.Builder, index int) {
	fmt.Fprintf(sb, "-- Step %d\n", index)
}

func writeSQLBlock(sb *strings.Builder, index int, sql string) {
	writeStep(sb, index)
	sb.WriteString(sql)
	sb.WriteString(";\n\n")
}

func writeProgrammaticBlock(sb *strings.Builder, index int, migration sqldb.Change) {
	writeStep(sb, index)
	fmt.Fprintf(sb, "— *Programmatic migration: %s*\n\n", migration)
}
