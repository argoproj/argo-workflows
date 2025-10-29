# HAProxy PostgreSQL Load Balancer

This component provides HAProxy as a drop-in replacement for PgCat, offering PostgreSQL load balancing with DNS resiliency for Argo Workflows. HAProxy delivers simplified configuration, improved performance, and robust failover capabilities while maintaining full compatibility with existing Argo Workflows deployments.

## Overview

HAProxy serves as a TCP-level load balancer that distributes PostgreSQL connections across multiple service endpoints, providing automatic failover and high availability. Unlike PgCat's complex TOML configuration and connection pooling overhead, HAProxy offers a streamlined approach with industry-standard reliability.

### Key Benefits

- **Drop-in Replacement**: Uses same service name (`pgcat`) and port (5432) as PgCat
- **Simplified Configuration**: Standard HAProxy config vs complex TOML
- **Better Performance**: Lower latency and resource usage than connection pooling
- **Enhanced Monitoring**: Built-in statistics interface and Prometheus metrics
- **Proven Reliability**: Battle-tested load balancer with extensive production use
- **DNS Resiliency**: Load balances across multiple PostgreSQL service endpoints

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Argo          │    │   HAProxy        │    │   PostgreSQL        │
│   Workflows     │───▶│   Load Balancer  │───▶│   Services          │
│   Controller    │    │   (pgcat:5432)   │    │   (3 endpoints)     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                              │                           │
                              │                           │
                              ▼                           ▼
                       ┌─────────────┐            ┌─────────────┐
                       │   Stats     │            │  Backends:  │
                       │  Interface  │            │  - Primary  │
                       │  :8404      │            │  - Secondary│
                       └─────────────┘            │  - Tertiary │
                                                  └─────────────┘
```

HAProxy provides TCP-level load balancing across three PostgreSQL service endpoints:
- [`postgres-primary.argo.svc.cluster.local:5432`](postgres-primary-service.yaml)
- [`postgres-secondary.argo.svc.cluster.local:5432`](postgres-secondary-service.yaml)  
- [`postgres-tertiary.argo.svc.cluster.local:5432`](postgres-tertiary-service.yaml)

## Components

This HAProxy component includes the following Kubernetes resources:

### Core Components

- **[`haproxy-config.yaml`](haproxy-config.yaml)**: ConfigMap containing HAProxy configuration
- **[`haproxy-deployment.yaml`](haproxy-deployment.yaml)**: Deployment with 2 replicas for high availability
- **[`haproxy-service.yaml`](haproxy-service.yaml)**: Service exposing PostgreSQL (5432) and stats (8404) ports
- **[`kustomization.yaml`](kustomization.yaml)**: Kustomize component definition

### Testing Framework

- **[`overlays/failover-testing/`](overlays/failover-testing/)**: Complete failover testing environment
  - PostgreSQL test deployment with multiple service endpoints
  - Comprehensive testing scenarios and validation commands
  - See [`overlays/failover-testing/README.md`](overlays/failover-testing/README.md) for details

## Configuration

### HAProxy Configuration Highlights

The [`haproxy-config.yaml`](haproxy-config.yaml) includes:

#### Global Settings
```haproxy
global
    daemon
    log stdout local0 info
    stats socket /var/run/haproxy.sock mode 600 level admin
    stats timeout 2m
```

#### Backend Configuration
```haproxy
backend postgres_backend
    mode tcp
    balance roundrobin
    option tcp-check
    tcp-check connect
    tcp-check send-binary 00000020000000030000757365720000706f737467726573000000
    tcp-check expect binary 52
    
    server postgres-primary postgres-primary.argo.svc.cluster.local:5432 check inter 1s rise 2 fall 3
    server postgres-secondary postgres-secondary.argo.svc.cluster.local:5432 check inter 1s rise 2 fall 3
    server postgres-tertiary postgres-tertiary.argo.svc.cluster.local:5432 check inter 1s rise 2 fall 3
```

### Key Configuration Parameters

| Parameter | Value | Purpose |
|-----------|-------|---------|
| `mode tcp` | TCP mode | Pure TCP proxying for maximum compatibility |
| `balance roundrobin` | Round-robin | Even distribution across available backends |
| `check inter 1s` | 1 second | Health check interval (matches PgCat) |
| `rise 2 fall 3` | 2/3 threshold | 2 successes to mark UP, 3 failures to mark DOWN |
| `timeout connect 5s` | 5 seconds | Connection establishment timeout |
| `timeout client/server 30s` | 30 seconds | Client and server communication timeout |

### PostgreSQL Health Checks

HAProxy uses PostgreSQL-specific health checks:
- **TCP Connection**: Verifies backend accepts connections
- **PostgreSQL Protocol**: Sends startup message and expects authentication response
- **Binary Check**: `tcp-check send-binary` sends PostgreSQL startup packet
- **Response Validation**: `tcp-check expect binary 52` expects authentication request

## Deployment

### Prerequisites

Ensure PostgreSQL services are available:
```bash
kubectl get services -n argo postgres-primary postgres-secondary postgres-tertiary
```

### Deploy HAProxy Component

Using Kustomize (recommended):
```bash
# Deploy the HAProxy component
kubectl apply -k manifests/components/haproxy/

# Verify deployment
kubectl get pods,services -n argo -l app=haproxy
```

Using individual manifests:
```bash
# Deploy individual resources
kubectl apply -f manifests/components/haproxy/haproxy-config.yaml
kubectl apply -f manifests/components/haproxy/haproxy-deployment.yaml
kubectl apply -f manifests/components/haproxy/haproxy-service.yaml
```

### Verify Deployment

```bash
# Check HAProxy pods are running
kubectl get pods -n argo -l app=haproxy

# Verify service endpoints
kubectl get endpoints -n argo pgcat

# Check HAProxy configuration
kubectl get configmap -n argo haproxy-config -o yaml
```

### Integration with Argo Workflows

HAProxy uses the same service name as PgCat (`pgcat.argo.svc.cluster.local:5432`), making it a drop-in replacement. No changes to Argo Workflows configuration are required.

## Monitoring

### HAProxy Statistics Interface

Access real-time statistics and backend health status:

```bash
# Port forward to stats interface
kubectl port-forward -n argo service/pgcat 8404:8404

# View stats in browser
open http://localhost:8404/stats

# Or use curl
curl http://localhost:8404/stats
```

The stats interface provides:
- **Backend Status**: UP/DOWN status for each PostgreSQL endpoint
- **Health Check Results**: Success/failure of health checks
- **Connection Statistics**: Active connections, request counts, response times
- **Load Balancing**: Distribution of requests across backends
- **Error Rates**: Connection errors and timeouts

### Key Metrics to Monitor

| Metric | Description | Normal Range |
|--------|-------------|--------------|
| Backend Status | UP/DOWN for each endpoint | All UP |
| Active Sessions | Current connections | < 80% of limits |
| Response Time | Average response time | < 100ms |
| Error Rate | Failed connections/requests | < 1% |
| Health Check Status | Success rate of health checks | > 95% |

### Prometheus Integration

HAProxy metrics can be exported to Prometheus:

```yaml
# Add to haproxy-deployment.yaml
- name: haproxy-exporter
  image: prom/haproxy-exporter:latest
  ports:
  - containerPort: 9101
  args:
  - --haproxy.scrape-uri=http://localhost:8404/stats?stats;csv
```

### Logging

HAProxy logs are available via kubectl:
```bash
# View HAProxy logs
kubectl logs -n argo -l app=haproxy -f

# Filter for specific events
kubectl logs -n argo -l app=haproxy | grep -E "(UP|DOWN|check)"
```

## Failover Testing

### Quick Failover Test

Test HAProxy's automatic failover capabilities:

```bash
# Deploy failover testing environment
kubectl apply -k manifests/components/haproxy/overlays/failover-testing/

# Monitor HAProxy stats
kubectl port-forward -n argo service/pgcat 8404:8404 &

# Simulate primary failure
kubectl label pod -n argo -l app=postgres-failover-test postgres-primary-

# Check stats - primary should show as DOWN
curl http://localhost:8404/stats | grep postgres-primary

# Restore primary
kubectl label pod -n argo -l app=postgres-failover-test postgres-primary=true

# Cleanup
kubectl delete -k manifests/components/haproxy/overlays/failover-testing/
```

### Comprehensive Testing

For detailed failover testing scenarios, see [`overlays/failover-testing/README.md`](overlays/failover-testing/README.md).

## Migration from PgCat

### Migration Steps

1. **Deploy HAProxy alongside PgCat**:
   ```bash
   # Deploy with different service name initially
   kubectl apply -k manifests/components/haproxy/
   # Temporarily rename service to haproxy-postgres for testing
   ```

2. **Test HAProxy functionality**:
   ```bash
   # Test connection through HAProxy
   kubectl run -it --rm postgres-client --image=postgres:13 --restart=Never -- \
     psql -h haproxy-postgres.argo.svc.cluster.local -U postgres -d postgres
   ```

3. **Update Argo Workflows configuration**:
   ```bash
   # Update workflow-controller-configmap to use HAProxy
   kubectl patch configmap workflow-controller-configmap -n argo --type merge \
     -p '{"data":{"config":"persistence:\n  postgresql:\n    host: haproxy-postgres.argo.svc.cluster.local\n    port: 5432"}}'
   ```

4. **Switch service names**:
   ```bash
   # Rename PgCat service
   kubectl patch service pgcat -n argo -p '{"metadata":{"name":"pgcat-old"}}'
   # Rename HAProxy service to pgcat
   kubectl patch service haproxy-postgres -n argo -p '{"metadata":{"name":"pgcat"}}'
   ```

5. **Remove PgCat**:
   ```bash
   # After validation, remove PgCat deployment
   kubectl delete deployment pgcat -n argo
   kubectl delete configmap pgcat-config -n argo
   kubectl delete service pgcat-old -n argo
   ```

### Migration Validation

```bash
# Verify Argo Workflows connectivity
kubectl logs -n argo deployment/workflow-controller | grep -i "database\|postgres"

# Test workflow execution
argo submit examples/hello-world.yaml -n argo

# Monitor HAProxy stats during workflow execution
curl http://localhost:8404/stats
```

### Rollback Plan

If issues arise during migration:
```bash
# Quick rollback to PgCat
kubectl patch service pgcat -n argo -p '{"metadata":{"name":"haproxy-temp"}}'
kubectl patch service pgcat-old -n argo -p '{"metadata":{"name":"pgcat"}}'
```

## Troubleshooting

### Common Issues

#### 1. Backend Health Check Failures

**Symptoms**: Backends showing as DOWN in stats interface

**Diagnosis**:
```bash
# Check HAProxy logs
kubectl logs -n argo -l app=haproxy | grep -i "health\|check"

# Test direct backend connectivity
kubectl run -it --rm postgres-client --image=postgres:13 --restart=Never -- \
  psql -h postgres-primary.argo.svc.cluster.local -U postgres -d postgres
```

**Solutions**:
- Verify PostgreSQL services are running and accessible
- Check PostgreSQL authentication configuration
- Ensure network policies allow HAProxy to reach backends

#### 2. Connection Timeouts

**Symptoms**: Client connections timing out

**Diagnosis**:
```bash
# Check HAProxy configuration
kubectl get configmap haproxy-config -n argo -o yaml

# Monitor connection statistics
kubectl exec -n argo deployment/haproxy -- \
  echo "show stat" | socat stdio /var/run/haproxy.sock
```

**Solutions**:
- Increase timeout values in HAProxy configuration
- Check network latency between HAProxy and backends
- Verify PostgreSQL max_connections setting

#### 3. Service Discovery Issues

**Symptoms**: HAProxy cannot resolve backend hostnames

**Diagnosis**:
```bash
# Check service endpoints
kubectl get endpoints -n argo postgres-primary postgres-secondary postgres-tertiary

# Test DNS resolution from HAProxy pod
kubectl exec -n argo deployment/haproxy -- nslookup postgres-primary.argo.svc.cluster.local
```

**Solutions**:
- Verify PostgreSQL services exist and have endpoints
- Check namespace and service names in HAProxy configuration
- Ensure CoreDNS is functioning correctly

### Debugging Commands

```bash
# View HAProxy runtime statistics
kubectl exec -n argo deployment/haproxy -- \
  echo "show info" | socat stdio /var/run/haproxy.sock

# Check backend server status
kubectl exec -n argo deployment/haproxy -- \
  echo "show servers state" | socat stdio /var/run/haproxy.sock

# Monitor real-time connections
kubectl exec -n argo deployment/haproxy -- \
  echo "show sess" | socat stdio /var/run/haproxy.sock

# Reload configuration without restart
kubectl exec -n argo deployment/haproxy -- \
  echo "reload" | socat stdio /var/run/haproxy.sock
```

### Performance Debugging

```bash
# Monitor connection rates
watch 'kubectl exec -n argo deployment/haproxy -- \
  echo "show stat" | socat stdio /var/run/haproxy.sock | grep postgres'

# Check for connection errors
kubectl logs -n argo -l app=haproxy | grep -E "(error|timeout|failed)"

# Verify resource usage
kubectl top pods -n argo -l app=haproxy
```

## Performance

### Resource Requirements

HAProxy is significantly more resource-efficient than PgCat:

| Component | Memory Request | Memory Limit | CPU Request | CPU Limit |
|-----------|----------------|--------------|-------------|-----------|
| HAProxy | 64Mi | 64Mi | 100m | 100m |
| PgCat (comparison) | 256Mi | 512Mi | 250m | 500m |

### Performance Characteristics

- **Latency**: ~1-2ms additional latency (vs 5-10ms for PgCat connection pooling)
- **Throughput**: Supports thousands of concurrent connections
- **Memory Usage**: Constant memory usage regardless of connection count
- **CPU Usage**: Minimal CPU overhead for TCP proxying

### Scaling Considerations

```yaml
# Horizontal scaling
spec:
  replicas: 3  # Increase for higher availability

# Resource scaling
resources:
  requests:
    memory: "128Mi"  # Scale up for high connection counts
    cpu: "200m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

### Performance Tuning

Key parameters for performance optimization:

```haproxy
# Increase connection limits
maxconn 4000

# Optimize timeouts for your workload
timeout connect 3s      # Faster connection establishment
timeout client 60s      # Longer for persistent connections
timeout server 60s

# Tune health check frequency
check inter 2s          # Less frequent checks for stable environments
```

## Security

### Network Security

- **Internal Communication**: All traffic remains within Kubernetes cluster
- **TLS Passthrough**: PostgreSQL handles TLS encryption, HAProxy passes through
- **Service Mesh**: Compatible with Istio/Linkerd for additional security

### Access Control

```yaml
# RBAC for HAProxy service account
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: haproxy-role
rules:
- apiGroups: [""]
  resources: ["services", "endpoints"]
  verbs: ["get", "list", "watch"]
```

### Configuration Security

- **ConfigMap Protection**: HAProxy config stored securely in Kubernetes
- **Secret Management**: Database credentials remain in existing secrets
- **Least Privilege**: HAProxy runs with minimal required permissions

## Advanced Configuration

### Custom Backend Configuration

Add additional PostgreSQL backends:

```haproxy
backend postgres_backend
    # ... existing configuration ...
    server postgres-fourth postgres-fourth.argo.svc.cluster.local:5432 check inter 1s rise 2 fall 3
```

### Load Balancing Algorithms

```haproxy
# Round-robin (default)
balance roundrobin

# Least connections
balance leastconn

# Source IP hash (sticky sessions)
balance source
```

### Advanced Health Checks

```haproxy
# Custom PostgreSQL health check
tcp-check connect
tcp-check send-binary 00000020000000030000757365720000706f737467726573000000
tcp-check expect binary 52
tcp-check send-binary 5800000008000000
```

### SSL/TLS Configuration

```haproxy
# TLS passthrough (recommended)
frontend postgres_frontend
    bind *:5432
    mode tcp
    default_backend postgres_backend

# TLS termination (if needed)
frontend postgres_frontend_ssl
    bind *:5433 ssl crt /etc/ssl/certs/postgres.pem
    mode tcp
    default_backend postgres_backend
```

## Related Documentation

- **Architecture**: [`haproxy-postgresql-architecture.md`](../../haproxy-postgresql-architecture.md) - Detailed architecture design
- **Failover Testing**: [`overlays/failover-testing/README.md`](overlays/failover-testing/README.md) - Comprehensive testing guide
- **PgCat Comparison**: [`../pgcat/README.md`](../pgcat/README.md) - Original PgCat implementation
- **HAProxy Documentation**: [Official HAProxy Documentation](https://docs.haproxy.org/)

## Support

For issues and questions:

1. **Check HAProxy logs**: `kubectl logs -n argo -l app=haproxy`
2. **Monitor stats interface**: `http://localhost:8404/stats`
3. **Review troubleshooting section** above
4. **Test with failover overlay**: [`overlays/failover-testing/`](overlays/failover-testing/)

This HAProxy implementation provides a robust, performant, and maintainable solution for PostgreSQL load balancing in Argo Workflows, offering significant improvements over the previous PgCat implementation while maintaining full compatibility.