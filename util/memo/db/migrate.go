package db

import "github.com/argoproj/argo-workflows/v4/util/sqldb"

// MigrateChanges describes the schema for a memoization cache table.
// Migrate validates tableName before applying these changes.
func MigrateChanges(tableName string, dbType sqldb.DBType) []sqldb.Change {
	return []sqldb.Change{
		// The composite key uses at most (64 + 253 + 256) * 4 = 2292 bytes with utf8mb4.
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.Postgres: sqldb.AnsiSQLChange(`create table if not exists "` + tableName + `" (
    namespace        varchar(64)  not null,
    cache_name       varchar(253) not null,
    cache_key        varchar(256) not null,
    node_id          text         not null,
    outputs          text         not null,
    created_at       timestamp    not null,
    expires_at       timestamp    null,
    primary key (namespace, cache_name, cache_key)
)`),
			sqldb.MySQL: sqldb.AnsiSQLChange("create table if not exists `" + tableName + "` (" +
				"namespace        varchar(64)  not null, " +
				"cache_name       varchar(253) not null, " +
				"cache_key        varchar(256) not null, " +
				"node_id          text         not null, " +
				"outputs          longtext     not null, " +
				"created_at       datetime(6)  not null, " +
				"expires_at       datetime(6)  null, " +
				"primary key (namespace, cache_name, cache_key)) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"),
		}),
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.Postgres: sqldb.AnsiSQLChange(`create index ` + tableName + `_i1 on "` + tableName + `" (expires_at)`),
			sqldb.MySQL:    sqldb.AnsiSQLChange("create index " + tableName + "_i1 on `" + tableName + "` (expires_at)"),
		}),
	}
}
