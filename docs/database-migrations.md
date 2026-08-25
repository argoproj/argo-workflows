# Database Migrations

This page lists the SQL migrations that Argo Workflows applies automatically at startup.
Migrations are applied incrementally and tracked by a version table.
Each migration is only run once. Do not re-order or remove entries.

The table names and cluster name shown here are defaults.
Your deployment may use different values depending on your configuration.

See [Workflow Archive](workflow-archive.md) and [Synchronization](synchronization.md) for configuration details.

## Steps

Each migration is numbered as a `Step`. When Argo Workflows runs the automatic migration at controller startup, it records the highest applied step number in a version table (`schema_history` for the archive database,`sync_schema_history` for the sync database) and only runs steps with a higher number on subsequent starts.
Steps may be missing where the step does nothing for the database type.
This means steps must never be re-ordered or removed — a new schema change is always appended as a new step.

If you run these statements yourself (for example, because you set `skipMigration: true` or want to manually create the schema), you are responsible for tracking which steps your database is already at.
Argo Workflows will not detect partial manual application — it only reads the version table.
If the version table is out of sync with the actual schema, the controller may try to re-apply steps and fail, or skip steps that you have not run.

Programmatic migrations are not described here and the code needs to be consulted for those steps.
## Archive Database

### MySQL

```sql
-- Step 0
create table if not exists argo_workflows (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    creationtimestamp timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
);

-- Step 1
create unique index idx_name on argo_workflows (name);

-- Step 2
create table if not exists argo_workflow_history (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
);

-- Step 3
alter table argo_workflow_history rename to argo_archived_workflows;

-- Step 4
drop index idx_name on argo_workflows;

-- Step 5
create unique index idx_name on argo_workflows(name, namespace);

-- Step 6
alter table argo_workflows drop primary key;

-- Step 7
alter table argo_workflows add primary key(name,namespace);

-- Step 8
alter table argo_archived_workflows drop primary key;

-- Step 9
alter table argo_archived_workflows add primary key(id);

-- Step 10
alter table argo_archived_workflows change column id uid varchar(128);

-- Step 11
alter table argo_archived_workflows modify column uid varchar(128) not null;

-- Step 12
alter table argo_archived_workflows modify column phase varchar(25) not null;

-- Step 13
alter table argo_archived_workflows modify column namespace varchar(256) not null;

-- Step 14
alter table argo_archived_workflows modify column workflow text not null;

-- Step 15
alter table argo_archived_workflows modify column startedat timestamp not null default CURRENT_TIMESTAMP;

-- Step 16
alter table argo_archived_workflows modify column finishedat timestamp not null default CURRENT_TIMESTAMP;

-- Step 17
alter table argo_archived_workflows add clustername varchar(64);

-- Step 18
update argo_archived_workflows set clustername = '<cluster-name>' where clustername is null;

-- Step 19
alter table argo_archived_workflows modify column clustername varchar(64) not null;

-- Step 20
alter table argo_archived_workflows drop primary key;

-- Step 21
alter table argo_archived_workflows add primary key(clustername,uid);

-- Step 22
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,namespace);

-- Step 23
alter table argo_workflows drop column phase;

-- Step 24
alter table argo_workflows drop column startedat;

-- Step 25
alter table argo_workflows drop column finishedat;

-- Step 26
alter table argo_workflows change column id uid varchar(128);

-- Step 27
alter table argo_workflows modify column uid varchar(128) not null;

-- Step 28
alter table argo_workflows modify column namespace varchar(256) not null;

-- Step 29
alter table argo_workflows add column clustername varchar(64);

-- Step 30
update argo_workflows set clustername = '<cluster-name>' where clustername is null;

-- Step 31
alter table argo_workflows modify column clustername varchar(64) not null;

-- Step 32
alter table argo_workflows add column version varchar(64);

-- Step 33
alter table argo_workflows add column nodes text;

-- Step 34
— *Programmatic migration: backfillNodes{argo_workflows}*

-- Step 35
alter table argo_workflows modify column nodes text not null;

-- Step 36
alter table argo_workflows drop column workflow;

-- Step 37
alter table argo_workflows add column updatedat timestamp not null default current_timestamp;

-- Step 38
alter table argo_workflows drop primary key;

-- Step 39
drop index idx_name on argo_workflows;

-- Step 40
alter table argo_workflows drop column name;

-- Step 41
alter table argo_workflows add primary key(clustername,uid,version);

-- Step 42
create index argo_workflows_i1 on argo_workflows (clustername,namespace);

-- Step 43
alter table argo_archived_workflows modify column workflow json not null;

-- Step 44
alter table argo_archived_workflows modify column name varchar(256) not null;

-- Step 45
create index argo_workflows_i2 on argo_workflows (clustername,namespace,updatedat);

-- Step 46
create table if not exists argo_archived_workflows_labels (
	clustername varchar(64) not null,
	uid varchar(128) not null,
    name varchar(317) not null,
    value varchar(63) not null,
    primary key (clustername, uid, name),
 	foreign key (clustername, uid) references argo_archived_workflows(clustername, uid) on delete cascade
);

-- Step 47
alter table argo_workflows modify column nodes json not null;

-- Step 48
alter table argo_archived_workflows add column instanceid varchar(64);

-- Step 49
update argo_archived_workflows set instanceid = '' where instanceid is null;

-- Step 50
alter table argo_archived_workflows modify column instanceid varchar(64) not null;

-- Step 51
drop index argo_archived_workflows_i1 on argo_archived_workflows;

-- Step 52
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,instanceid,namespace);

-- Step 53
drop index argo_workflows_i1 on argo_workflows;

-- Step 54
drop index argo_workflows_i2 on argo_workflows;

-- Step 55
create index argo_workflows_i1 on argo_workflows (clustername,namespace,updatedat);

-- Step 56
create index argo_archived_workflows_i2 on argo_archived_workflows (clustername,instanceid,finishedat);

-- Step 57
create index argo_archived_workflows_i3 on argo_archived_workflows (clustername,instanceid,name);

-- Step 58
create index argo_archived_workflows_i4 on argo_archived_workflows (startedat);

-- Step 59
create index argo_archived_workflows_labels_i1 on argo_archived_workflows_labels (name,value);

-- Step 61
drop index argo_archived_workflows_i4 on argo_archived_workflows;

-- Step 62
create index argo_archived_workflows_i4 on argo_archived_workflows (clustername, startedat);

-- Step 63
alter table argo_archived_workflows add column creationtimestamp timestamp null;

-- Step 64
update argo_archived_workflows set creationtimestamp = startedat where creationtimestamp is null;

-- Step 65
alter table argo_archived_workflows modify column creationtimestamp timestamp not null default CURRENT_TIMESTAMP;

-- Step 66
create index argo_archived_workflows_i5 on argo_archived_workflows (creationtimestamp);

-- Step 67
drop index argo_archived_workflows_i1 on argo_archived_workflows;

-- Step 68
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername, instanceid, namespace, startedat DESC);

```

### PostgreSQL

```sql
-- Step 0
create table if not exists argo_workflows (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    creationtimestamp timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
);

-- Step 1
create unique index idx_name on argo_workflows (name);

-- Step 2
create table if not exists argo_workflow_history (
    id varchar(128) ,
    name varchar(256),
    phase varchar(25),
    namespace varchar(256),
    workflow text,
    startedat timestamp default CURRENT_TIMESTAMP,
    finishedat timestamp default CURRENT_TIMESTAMP,
    primary key (id, namespace)
);

-- Step 3
alter table argo_workflow_history rename to argo_archived_workflows;

-- Step 4
drop index idx_name;

-- Step 5
create unique index idx_name on argo_workflows(name, namespace);

-- Step 6
alter table argo_workflows drop constraint argo_workflows_pkey;

-- Step 7
alter table argo_workflows add primary key(name,namespace);

-- Step 8
alter table argo_archived_workflows drop constraint argo_workflow_history_pkey;

-- Step 9
alter table argo_archived_workflows add primary key(id);

-- Step 10
alter table argo_archived_workflows rename column id to uid;

-- Step 11
alter table argo_archived_workflows alter column uid set not null;

-- Step 12
alter table argo_archived_workflows alter column phase set not null;

-- Step 13
alter table argo_archived_workflows alter column namespace set not null;

-- Step 14
alter table argo_archived_workflows alter column workflow set not null;

-- Step 15
alter table argo_archived_workflows alter column startedat set not null;

-- Step 16
alter table argo_archived_workflows alter column finishedat set not null;

-- Step 17
alter table argo_archived_workflows add clustername varchar(64);

-- Step 18
update argo_archived_workflows set clustername = '<cluster-name>' where clustername is null;

-- Step 19
alter table argo_archived_workflows alter column clustername set not null;

-- Step 20
alter table argo_archived_workflows drop constraint argo_archived_workflows_pkey;

-- Step 21
alter table argo_archived_workflows add primary key(clustername,uid);

-- Step 22
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,namespace);

-- Step 23
alter table argo_workflows drop column phase;

-- Step 24
alter table argo_workflows drop column startedat;

-- Step 25
alter table argo_workflows drop column finishedat;

-- Step 26
alter table argo_workflows rename column id to uid;

-- Step 27
alter table argo_workflows alter column uid set not null;

-- Step 28
alter table argo_workflows alter column namespace set not null;

-- Step 29
alter table argo_workflows add column clustername varchar(64);

-- Step 30
update argo_workflows set clustername = '<cluster-name>' where clustername is null;

-- Step 31
alter table argo_workflows alter column clustername set not null;

-- Step 32
alter table argo_workflows add column version varchar(64);

-- Step 33
alter table argo_workflows add column nodes text;

-- Step 34
— *Programmatic migration: backfillNodes{argo_workflows}*

-- Step 35
alter table argo_workflows alter column nodes set not null;

-- Step 36
alter table argo_workflows drop column workflow;

-- Step 37
alter table argo_workflows add column updatedat timestamp not null default current_timestamp;

-- Step 38
alter table argo_workflows drop constraint argo_workflows_pkey;

-- Step 39
drop index idx_name;

-- Step 40
alter table argo_workflows drop column name;

-- Step 41
alter table argo_workflows add primary key(clustername,uid,version);

-- Step 42
create index argo_workflows_i1 on argo_workflows (clustername,namespace);

-- Step 43
alter table argo_archived_workflows alter column workflow type json using workflow::json;

-- Step 44
alter table argo_archived_workflows alter column name set not null;

-- Step 45
create index argo_workflows_i2 on argo_workflows (clustername,namespace,updatedat);

-- Step 46
create table if not exists argo_archived_workflows_labels (
	clustername varchar(64) not null,
	uid varchar(128) not null,
    name varchar(317) not null,
    value varchar(63) not null,
    primary key (clustername, uid, name),
 	foreign key (clustername, uid) references argo_archived_workflows(clustername, uid) on delete cascade
);

-- Step 47
alter table argo_workflows alter column nodes type json using nodes::json;

-- Step 48
alter table argo_archived_workflows add column instanceid varchar(64);

-- Step 49
update argo_archived_workflows set instanceid = '' where instanceid is null;

-- Step 50
alter table argo_archived_workflows alter column instanceid set not null;

-- Step 51
drop index argo_archived_workflows_i1;

-- Step 52
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername,instanceid,namespace);

-- Step 53
drop index argo_workflows_i1;

-- Step 54
drop index argo_workflows_i2;

-- Step 55
create index argo_workflows_i1 on argo_workflows (clustername,namespace,updatedat);

-- Step 56
create index argo_archived_workflows_i2 on argo_archived_workflows (clustername,instanceid,finishedat);

-- Step 57
create index argo_archived_workflows_i3 on argo_archived_workflows (clustername,instanceid,name);

-- Step 58
create index argo_archived_workflows_i4 on argo_archived_workflows (startedat);

-- Step 59
create index argo_archived_workflows_labels_i1 on argo_archived_workflows_labels (name,value);

-- Step 60
alter table argo_archived_workflows alter column workflow set data type jsonb using workflow::jsonb;

-- Step 61
drop index argo_archived_workflows_i4;

-- Step 62
create index argo_archived_workflows_i4 on argo_archived_workflows (clustername, startedat);

-- Step 63
alter table argo_archived_workflows add column creationtimestamp timestamp null;

-- Step 64
update argo_archived_workflows set creationtimestamp = startedat where creationtimestamp is null;

-- Step 65
alter table argo_archived_workflows alter column creationtimestamp set default CURRENT_TIMESTAMP;

-- Step 66
create index argo_archived_workflows_i5 on argo_archived_workflows (creationtimestamp);

-- Step 67
drop index argo_archived_workflows_i1;

-- Step 68
create index argo_archived_workflows_i1 on argo_archived_workflows (clustername, instanceid, namespace, startedat DESC);

```

## Sync Database

### MySQL

```sql
-- Step 0
create table if not exists sync_limit (
    name varchar(256) not null,
    sizelimit int,
    primary key (name)
);

-- Step 1
create unique index ilimit_name on sync_limit (name);

-- Step 2
create table if not exists sync_controller (
    controller varchar(64) not null,
    time timestamp,
    primary key (controller)
);

-- Step 3
create unique index icontroller_name on sync_controller (controller);

-- Step 4
create table if not exists sync_state (
    name varchar(256),
    workflowkey varchar(256),
    controller varchar(64) not null,
    held boolean,
    priority int,
    time timestamp,
    primary key(name, workflowkey, controller)
);

-- Step 5
create index istate_name on sync_state (name);

-- Step 6
create index istate_workflowkey on sync_state (workflowkey);

-- Step 7
create index istate_controller on sync_state (controller);

-- Step 8
create index istate_held on sync_state (held);

-- Step 9
create table if not exists sync_lock (
    name varchar(256),
    controller varchar(64) not null,
    time timestamp,
    primary key(name)
);

-- Step 10
create unique index ilock_name on sync_lock (name);

```

### PostgreSQL

```sql
-- Step 0
create table if not exists sync_limit (
    name varchar(256) not null,
    sizelimit int,
    primary key (name)
);

-- Step 1
create unique index ilimit_name on sync_limit (name);

-- Step 2
create table if not exists sync_controller (
    controller varchar(64) not null,
    time timestamp,
    primary key (controller)
);

-- Step 3
create unique index icontroller_name on sync_controller (controller);

-- Step 4
create table if not exists sync_state (
    name varchar(256),
    workflowkey varchar(256),
    controller varchar(64) not null,
    held boolean,
    priority int,
    time timestamp,
    primary key(name, workflowkey, controller)
);

-- Step 5
create index istate_name on sync_state (name);

-- Step 6
create index istate_workflowkey on sync_state (workflowkey);

-- Step 7
create index istate_controller on sync_state (controller);

-- Step 8
create index istate_held on sync_state (held);

-- Step 9
create table if not exists sync_lock (
    name varchar(256),
    controller varchar(64) not null,
    time timestamp,
    primary key(name)
);

-- Step 10
create unique index ilock_name on sync_lock (name);

```

