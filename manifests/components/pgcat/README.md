# PgCat DNS Resiliency Configuration

This configuration sets up PgCat as a PostgreSQL connection pooler with DNS resiliency and failover capabilities for external PostgreSQL instances.

## Overview

The modified PgCat deployment provides:
- **DNS Resiliency**: Multiple PostgreSQL endpoints pointing to the same database cluster
- **Automatic Failover**: Health checks and automatic switching between endpoints
- **High Availability**: Connection pooling with resilient configuration
- **External Database Support**: No embedded PostgreSQL container

## Configuration Features

### DNS Resiliency
- Multiple server endpoints in the same pool for redundancy
- Load balancing across available endpoints
- Automatic failover when endpoints become unavailable

### Failover Configuration
- `automatic_failover = true`: Enables automatic switching between servers
- `failover_timeout = 5000`: 5-second timeout for failover detection
- `healthcheck_delay = 30000`: Health checks every 30 seconds
- `load_balancing_mode = "random"`: Distributes connections randomly

### Connection Pooling
- `pool_mode = "transaction"`: Transaction-level pooling for efficiency
- `pool_size = 25`: Maximum 25 connections per pool
- `min_pool_size = 5`: Minimum 5 connections maintained
- Connection timeouts and limits for resilience

## Setup Instructions

### 1. Configure PostgreSQL Endpoints

Edit the `argo-pgcat-config` secret to update the PostgreSQL server endpoints:

**Important**: PgCat requires only one primary server per shard. For DNS resiliency with multiple endpoints pointing to the same database, configure multiple shards with one primary each:

```toml
# Primary shard with first endpoint
[pools.primary.shards.0]
servers = [
  [ "your-postgres-primary.example.com", 5432, "primary" ],
]
database = "your-database-name"
connect_timeout = 5000
query_timeout = 30000

# Secondary shard for DNS resiliency
[pools.primary.shards.1]
servers = [
  [ "your-postgres-secondary.example.com", 5432, "primary" ],
]
database = "your-database-name"
connect_timeout = 5000
query_timeout = 30000

# Tertiary shard for maximum DNS resiliency
[pools.primary.shards.2]
servers = [
  [ "your-postgres-tertiary.example.com", 5432, "primary" ],
]
database = "your-database-name"
connect_timeout = 5000
query_timeout = 30000
```

Replace the example hostnames with your actual PostgreSQL service endpoints:
- These should all point to the same underlying database cluster
- Use different DNS names or IP addresses for maximum resiliency
- All endpoints should be accessible from the Kubernetes cluster
- Each shard can only have one primary server (pgcat requirement)

### 2. Update Database Credentials

Update the database credentials in the secret:

```yaml
stringData:
  username: your-postgres-username
  password: your-postgres-password
```

### 3. Configure Database Name

Update the database name in the configuration:

```toml
database = "your-database-name"
```

### 4. Deploy the Configuration

Apply the manifests:

```bash
kubectl apply -f manifests/components/pgcat/
```

## Usage

### Connecting to PgCat

Applications should connect to PgCat instead of directly to PostgreSQL:

- **Host**: `pgcat.default.svc.cluster.local` (or the service name in your namespace)
- **Port**: `6432`
- **Database**: The database name configured in pgcat.toml
- **Username/Password**: The credentials configured in the secret

### Connection String Example

```
postgresql://username:password@pgcat.default.svc.cluster.local:6432/database_name
```

## Monitoring

PgCat exposes Prometheus metrics on port 9930:
- Connection pool statistics
- Server health status
- Failover events
- Query performance metrics

## Troubleshooting

### Check PgCat Logs
```bash
kubectl logs -l app=pgcat -f
```

### Verify Configuration
```bash
kubectl get secret argo-pgcat-config -o yaml
```

### Test Connectivity
```bash
kubectl exec -it deployment/pgcat -- psql -h localhost -p 6432 -U username -d database_name
```

### Health Check Endpoints
- PgCat health: `http://pgcat:6432/health`
- Prometheus metrics: `http://pgcat:9930/metrics`

## DNS Resiliency Benefits

1. **Multiple Entry Points**: If one DNS name fails, others remain available
2. **Geographic Distribution**: Endpoints can be in different regions/zones
3. **Load Distribution**: Connections spread across available endpoints
4. **Automatic Recovery**: Failed endpoints automatically rejoin when healthy
5. **Zero Downtime**: Failover happens transparently to applications

## Configuration Customization

Key parameters that can be adjusted:

- `connect_timeout`: Time to wait for new connections
- `idle_timeout`: How long to keep idle connections
- `healthcheck_delay`: Frequency of health checks
- `pool_size`: Maximum connections per pool
- `failover_timeout`: How quickly to detect failures

Adjust these values based on your specific requirements and network conditions.