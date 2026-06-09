package db

import (
	"context"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

func migrate(ctx context.Context, session db.Session, dbType sqldb.DBType, tableName string) error {
	if err := validateTableName(tableName); err != nil {
		return err
	}
	return sqldb.Migrate(ctx, session, dbType, memoizationVersionTableName(tableName), []sqldb.Change{
		// MySQL: use LONGTEXT for outputs (TEXT is 64KB).
		// Postgres: use text for outputs (no size limit).
		// Varchar sizes chosen to keep composite PK within InnoDB's 3072-byte limit with utf8mb4:
		//   (64 + 128 + 256) * 4 = 1792 bytes.
		sqldb.ByType(dbType, sqldb.TypedChanges{
			sqldb.Postgres: sqldb.AnsiSQLChange(`create table if not exists ` + tableName + ` (
    namespace        varchar(64)  not null,
    cache_name       varchar(128) not null,
    cache_key        varchar(256) not null,
    node_id          text         not null,
    outputs          text         not null,
    created_at       timestamp    not null,
    expires_at       timestamp    not null,
    primary key (namespace, cache_name, cache_key)
)`),
			sqldb.MySQL: sqldb.AnsiSQLChange("create table if not exists " + tableName + " (" +
				"namespace        varchar(64)  not null, " +
				"cache_name       varchar(128) not null, " +
				"cache_key        varchar(256) not null, " +
				"node_id          text         not null, " +
				"outputs          longtext     not null, " +
				"created_at       timestamp    not null, " +
				"expires_at       timestamp    not null, " +
				"primary key (namespace, cache_name, cache_key))"),
		}),
		sqldb.AnsiSQLChange(`create index ` + memoizationExpiresAtIndexName(tableName) + ` on ` + tableName + ` (expires_at)`),
	})
}
