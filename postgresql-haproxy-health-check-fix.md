# PostgreSQL HAProxy Health Check Fix

## Problem Summary

PostgreSQL was logging frequent errors indicating incomplete startup packets and connection resets:
```
2025-10-28 08:31:46.242 UTC [882] LOG:  incomplete startup packet
2025-10-28 08:31:46.279 UTC [883] LOG:  could not receive data from client: Connection reset by peer
2025-10-28 08:31:46.279 UTC [883] LOG:  incomplete startup packet
```

## Root Cause Analysis

### Original HAProxy Configuration Issues

The original HAProxy configuration in [`manifests/components/haproxy/haproxy-config.yaml`](manifests/components/haproxy/haproxy-config.yaml) had the following problematic health check setup:

```haproxy
option tcp-check
tcp-check connect
tcp-check send-binary 00000020000000030000757365720000706f737467726573000000
tcp-check expect binary 52
```

### Identified Problems

1. **Incomplete PostgreSQL Protocol Handshake**: HAProxy was sending a PostgreSQL startup packet but not completing the full authentication handshake, causing PostgreSQL to log "incomplete startup packet" when the connection was abruptly closed.

2. **Missing Connection Termination**: HAProxy wasn't sending a proper PostgreSQL termination message before closing the connection, causing PostgreSQL to log "connection reset by peer".

3. **Aggressive Health Check Frequency**: 1-second health check intervals were overwhelming PostgreSQL with incomplete connection attempts.

4. **Malformed Startup Packet**: The binary data being sent may not have been properly formatted for PostgreSQL protocol version 3.0.

## Solution Implemented

### Simple TCP Health Check Approach

The fix implements a simple TCP connection health check instead of attempting PostgreSQL protocol communication:

```haproxy
# PostgreSQL backend pool
backend postgres_backend
    mode tcp
    balance roundrobin
    
    # Use simple TCP health checks to avoid PostgreSQL protocol interference
    # This prevents "incomplete startup packet" and "connection reset by peer" errors
    option tcp-check
    tcp-check connect
    
    # Increased health check interval to reduce PostgreSQL log noise
    # Changed from 1s to 10s to be less aggressive
    server postgres-primary postgres-primary.argo.svc.cluster.local:5432 check inter 10s rise 2 fall 3
    server postgres-secondary postgres-secondary.argo.svc.cluster.local:5432 check inter 10s rise 2 fall 3
    server postgres-tertiary postgres-tertiary.argo.svc.cluster.local:5432 check inter 10s rise 2 fall 3
```

### Key Changes

1. **Removed PostgreSQL Protocol Commands**: Eliminated `tcp-check send-binary` and `tcp-check expect binary` directives that were causing protocol interference.

2. **Simple TCP Connect Check**: Uses only `tcp-check connect` to verify that PostgreSQL is accepting connections on port 5432.

3. **Increased Health Check Interval**: Changed from 1 second to 10 seconds to reduce the frequency of health checks and minimize log noise.

4. **Maintained Failover Parameters**: Kept `rise 2 fall 3` settings for proper failover behavior.

## Alternative Solutions Considered

### Option 1: Enhanced PostgreSQL Protocol Health Check
Created [`optimal-postgresql-health-check.yaml`](optimal-postgresql-health-check.yaml) with proper PostgreSQL protocol sequence including:
- Correct startup message format
- Authentication handling
- Proper connection termination

### Option 2: Debug Configuration
Created [`debug-haproxy-config.yaml`](debug-haproxy-config.yaml) with enhanced logging and partial protocol termination.

### Option 3: Simple TCP (Recommended)
Created [`simple-tcp-health-check.yaml`](simple-tcp-health-check.yaml) - the approach implemented in the fix.

## Benefits of the Solution

1. **Eliminates PostgreSQL Errors**: No more "incomplete startup packet" or "connection reset by peer" errors.

2. **Maintains Health Check Functionality**: Still detects when PostgreSQL services are down or unreachable.

3. **Reduces Log Noise**: Significantly fewer health check-related log entries.

4. **Improved Performance**: Lower overhead from health checks.

5. **Simplified Configuration**: Easier to understand and maintain.

## Validation Steps

To validate the fix:

1. **Deploy the updated configuration**:
   ```bash
   kubectl apply -f manifests/components/haproxy/haproxy-config.yaml
   ```

2. **Monitor PostgreSQL logs** for elimination of errors:
   ```bash
   kubectl logs -f deployment/postgres-primary -n argo | grep -E "(incomplete|reset)"
   ```

3. **Check HAProxy health status**:
   ```bash
   kubectl port-forward -n argo service/pgcat 8404:8404
   curl http://localhost:8404/stats
   ```

4. **Verify backend connectivity**:
   ```bash
   kubectl exec -n argo deployment/haproxy -- \
     echo "show servers state" | socat stdio /var/run/haproxy.sock
   ```

## Monitoring Recommendations

- Monitor HAProxy stats interface at `:8404/stats` for backend health status
- Watch for any increase in connection errors or timeouts
- Verify that all three PostgreSQL backends show as "UP" in HAProxy stats
- Monitor PostgreSQL logs to confirm elimination of health check-related errors

## Rollback Plan

If issues arise, the original configuration can be restored by reverting the changes to use PostgreSQL protocol health checks, though this will reintroduce the logging errors.

## Conclusion

The simple TCP health check approach provides effective backend monitoring while eliminating the PostgreSQL protocol interference that was causing the error logs. This solution maintains high availability and failover capabilities while significantly reducing operational noise.