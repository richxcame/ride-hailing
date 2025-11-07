# Istio Service Mesh - Configuration Guide

Complete guide for deploying Istio service mesh for the Ride Hailing Platform.

## Overview

Istio provides:

-   **Traffic Management** - Smart routing, load balancing, circuit breaking
-   **Security** - mTLS, authentication, authorization
-   **Observability** - Distributed tracing, metrics, logs
-   **Resilience** - Retries, timeouts, circuit breaking, fault injection

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Internet / Clients                     │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  Istio Ingress       │
              │  Gateway             │
              │  (with mTLS)         │
              └──────────┬───────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌────▼────┐    ┌────▼────┐
    │  Auth   │    │  Rides  │   │   Geo   │
    │  Pod    │    │  Pod    │   │  Pod    │
    │ [Envoy] │    │ [Envoy] │   │ [Envoy] │
    │ [App]   │    │ [App]   │   │ [App]   │
    └────┬────┘    └────┬────┘    └────┬────┘
         │              │              │
         │    mTLS Encrypted Traffic   │
         │              │              │
         └──────────────┴──────────────┘
                        │
              ┌─────────▼─────────┐
              │   Istio Control   │
              │   Plane (Pilot)   │
              └───────────────────┘
```

## Installation

### 1. Install Istio

```bash
# Run the installation script
./install-istio.sh
```

Or manually:

```bash
# Download Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install Istio
istioctl install --set profile=production -y

# Label namespace for sidecar injection
kubectl label namespace ridehailing istio-injection=enabled
```

### 2. Verify Installation

```bash
# Check Istio components
kubectl get pods -n istio-system

# Verify namespace is labeled
kubectl get namespace ridehailing --show-labels

# Check sidecar injector
kubectl get mutatingwebhookconfiguration
```

### 3. Deploy Istio Configurations

```bash
# Apply Gateway and VirtualServices
kubectl apply -f gateway.yaml

# Apply DestinationRules
kubectl apply -f destination-rules.yaml

# Apply Security Policies
kubectl apply -f security-policies.yaml
```

### 4. Restart Pods for Sidecar Injection

```bash
# Restart all deployments to inject Envoy sidecars
kubectl rollout restart deployment -n ridehailing

# Verify sidecars are injected (should show 2/2 READY)
kubectl get pods -n ridehailing
```

## Traffic Management

### Gateway Configuration

The Istio Gateway handles external traffic:

**Hosts:**

-   `api.ridehailing.com` - Main API
-   `admin.ridehailing.com` - Admin API
-   `ws.ridehailing.com` - WebSocket endpoint

**Ports:**

-   Port 80: HTTP (redirects to HTTPS)
-   Port 443: HTTPS with TLS

### Virtual Services

Virtual Services define routing rules:

**Features:**

-   Path-based routing
-   Automatic retries
-   Configurable timeouts
-   WebSocket support

**Example routes:**

```yaml
# Auth Service - 10s timeout, 3 retries
/api/v1/auth → auth-service:8080

# Rides Service - 30s timeout
/api/v1/rides → rides-service:8080

# WebSocket - 1h timeout
/ws → realtime-service:8080
```

### Destination Rules

Destination Rules configure traffic policies:

**Circuit Breaking:**

-   Auth: Max 100 connections, 50 pending requests
-   Rides: Max 200 connections, 100 pending requests
-   Geo: Max 300 connections (high throughput)
-   Payments: Max 100 connections (strict limits)

**Load Balancing:**

-   ROUND_ROBIN: Most services
-   LEAST_REQUEST: Auth, Geo, Analytics (better for variable workloads)
-   CONSISTENT_HASH: Real-time (sticky sessions for WebSocket)

**Outlier Detection:**

-   Consecutive errors threshold: 2-5 (depending on service)
-   Ejection time: 15-60 seconds
-   Health checks every 5-30 seconds

## Security

### Mutual TLS (mTLS)

All service-to-service communication is encrypted with mTLS:

```yaml
# Enforce STRICT mTLS
spec:
    mtls:
        mode: STRICT
```

**Verify mTLS:**

```bash
# Check mTLS status
istioctl authn tls-check <pod-name> <service-name>

# Example
istioctl authn tls-check auth-service-xxx auth-service.ridehailing.svc.cluster.local
```

### JWT Authentication

API requests require valid JWT tokens:

```yaml
# JWT issuer
issuer: 'ridehailing-auth-service'
audiences: ['ridehailing-api']
```

**Exempt endpoints:**

-   `/healthz` - Health checks
-   `/metrics` - Monitoring
-   `/api/v1/auth/*` - Login/registration

### Authorization Policies

Fine-grained access control:

**Admin Service:**

-   Requires `role=admin` in JWT claims

**Payments Service:**

-   Requires `role=rider|driver|admin`

**Service-to-Service:**

-   Rides → can call Payments, Geo, Promos, Fraud
-   Analytics → can read from all services
-   Notifications → can be called by any service

## Observability

### Kiali Dashboard

Visual service mesh management:

```bash
# Open Kiali
istioctl dashboard kiali
```

**Features:**

-   Service topology graph
-   Traffic flow visualization
-   Configuration validation
-   Health metrics

### Grafana Dashboards

Pre-built Istio dashboards:

```bash
# Open Grafana
istioctl dashboard grafana
```

**Dashboards:**

-   Istio Service Dashboard
-   Istio Workload Dashboard
-   Istio Performance Dashboard
-   Istio Control Plane Dashboard

### Jaeger Tracing

Distributed tracing for request flows:

```bash
# Open Jaeger
istioctl dashboard jaeger
```

**Use Cases:**

-   Track request through multiple services
-   Identify bottlenecks
-   Debug latency issues
-   Analyze error chains

### Prometheus Metrics

Service mesh metrics:

```bash
# Open Prometheus
istioctl dashboard prometheus
```

**Key Metrics:**

-   `istio_requests_total` - Total requests
-   `istio_request_duration_milliseconds` - Latency
-   `istio_tcp_connections_opened_total` - TCP connections
-   `istio_tcp_received_bytes_total` - Bandwidth

## Advanced Features

### Traffic Splitting (Canary Deployments)

Deploy new versions gradually:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
    name: rides-canary
spec:
    hosts:
        - rides-service
    http:
        - match:
              - headers:
                    x-version:
                        exact: 'v2'
          route:
              - destination:
                    host: rides-service
                    subset: v2
        - route:
              - destination:
                    host: rides-service
                    subset: v1
                weight: 90
              - destination:
                    host: rides-service
                    subset: v2
                weight: 10 # 10% to new version
```

### Fault Injection

Test resilience:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
    name: fault-injection-test
spec:
    hosts:
        - payments-service
    http:
        - fault:
              delay:
                  percentage:
                      value: 10 # 10% of requests
                  fixedDelay: 5s
              abort:
                  percentage:
                      value: 5 # 5% of requests
                  httpStatus: 500
          route:
              - destination:
                    host: payments-service
```

### Request Mirroring

Shadow traffic to new versions:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
    name: mirror-test
spec:
    hosts:
        - rides-service
    http:
        - route:
              - destination:
                    host: rides-service
                    subset: v1
          mirror:
              host: rides-service
              subset: v2
          mirrorPercentage:
              value: 100
```

### Circuit Breaking

Prevent cascading failures:

```yaml
# Already configured in destination-rules.yaml
connectionPool:
    tcp:
        maxConnections: 100
    http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 2
outlierDetection:
    consecutiveErrors: 5
    interval: 30s
    baseEjectionTime: 30s
```

**Test Circuit Breaking:**

```bash
# Generate load
kubectl run -it fortio --image=fortio/fortio -- load \
  -c 3 -qps 0 -n 20 -loglevel Warning \
  http://auth-service:8080/healthz
```

## Monitoring & Alerting

### Service Level Objectives (SLOs)

Track service health:

```bash
# Install SLO exporter
kubectl apply -f https://raw.githubusercontent.com/slok/sloth/main/deploy/kubernetes/helm/sloth/crds/sloth.slok.dev_prometheusservicelevels.yaml
```

**Example SLO:**

```yaml
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
    name: rides-service-slo
spec:
    service: 'rides-service'
    slos:
        - name: 'requests-availability'
          objective: 99.9
          sli:
              events:
                  errorQuery: |
                      sum(rate(istio_requests_total{destination_service="rides-service",response_code=~"5.."}[5m]))
                  totalQuery: |
                      sum(rate(istio_requests_total{destination_service="rides-service"}[5m]))
```

### Alerts

Configure Prometheus alerts:

```yaml
groups:
    - name: istio
      rules:
          - alert: HighErrorRate
            expr: rate(istio_requests_total{response_code=~"5.."}[5m]) > 0.05
            for: 5m
            annotations:
                summary: 'High error rate detected'

          - alert: HighLatency
            expr: histogram_quantile(0.99, istio_request_duration_milliseconds_bucket) > 1000
            for: 5m
            annotations:
                summary: 'High latency detected (p99 > 1s)'
```

## Troubleshooting

### Sidecar Not Injected

```bash
# Check namespace label
kubectl get namespace ridehailing --show-labels

# Check injection status
kubectl get deployment auth-service -o yaml | grep sidecar.istio.io/inject

# Force injection
kubectl patch deployment auth-service \
  -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}'
```

### mTLS Issues

```bash
# Check mTLS configuration
istioctl authn tls-check <pod-name> <service-name>

# View effective policy
kubectl get peerauthentication -n ridehailing -o yaml

# Check certificates
istioctl proxy-config secret <pod-name>
```

### Gateway Not Working

```bash
# Check gateway status
kubectl get gateway -n ridehailing

# Check virtual services
kubectl get virtualservice -n ridehailing

# View gateway configuration
istioctl proxy-config routes <istio-ingress-pod> -o json

# Check Istio ingress logs
kubectl logs -l app=istio-ingressgateway -n istio-system
```

### High Latency

```bash
# Check Envoy stats
istioctl dashboard envoy <pod-name>

# View request traces
istioctl dashboard jaeger

# Check circuit breaker stats
kubectl exec <pod-name> -c istio-proxy -- \
  curl localhost:15000/stats | grep -i "circuit_breakers"
```

## Performance Tuning

### Resource Limits for Envoy Sidecar

```yaml
# Adjust sidecar resources
apiVersion: v1
kind: ConfigMap
metadata:
    name: istio-sidecar-injector
    namespace: istio-system
data:
    values: |
        global:
          proxy:
            resources:
              requests:
                cpu: 100m
                memory: 128Mi
              limits:
                cpu: 2000m
                memory: 1Gi
```

### Connection Pool Tuning

```yaml
# Increase for high-throughput services
connectionPool:
    tcp:
        maxConnections: 500 # Increase from default 100
    http:
        http2MaxRequests: 500
        maxRequestsPerConnection: 5
```

### Disable Tracing for High-Volume Endpoints

```yaml
# Reduce overhead
spec:
    meshConfig:
        defaultConfig:
            tracing:
                sampling: 1.0 # Sample 1% instead of 100%
```

## Migration Strategy

### Phase 1: Install Istio (No Sidecar Injection)

```bash
# Install without auto-injection
istioctl install --set profile=production -y

# Don't label namespace yet
```

### Phase 2: Test with One Service

```bash
# Inject sidecar for one deployment
kubectl patch deployment auth-service \
  -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}'

# Verify it works
kubectl get pods -l app=auth-service
```

### Phase 3: Gradual Rollout

```bash
# Enable for namespace
kubectl label namespace ridehailing istio-injection=enabled

# Restart deployments one by one
kubectl rollout restart deployment auth-service
# Wait and verify
kubectl rollout restart deployment rides-service
# Continue...
```

### Phase 4: Enable Security Policies

```bash
# Start with PERMISSIVE mode
kubectl apply -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ridehailing
spec:
  mtls:
    mode: PERMISSIVE  # Allow both mTLS and plain text
EOF

# After all services are ready, switch to STRICT
kubectl patch peerauthentication default -n ridehailing \
  --type merge -p '{"spec":{"mtls":{"mode":"STRICT"}}}'
```

## Cost Optimization

**Sidecar Resource Usage:**

-   Each Envoy sidecar: ~50-100MB RAM, 50-100m CPU
-   For 30 pods: ~1.5-3GB RAM, 1.5-3 CPU cores

**Recommendations:**

-   Use Istio only for critical services
-   Disable sidecar injection for internal tools
-   Use ambient mesh (new Istio feature) for lower overhead

## Cleanup

```bash
# Remove all Istio configurations
kubectl delete -f security-policies.yaml
kubectl delete -f destination-rules.yaml
kubectl delete -f gateway.yaml

# Unlabel namespace
kubectl label namespace ridehailing istio-injection-

# Restart pods to remove sidecars
kubectl rollout restart deployment -n ridehailing

# Uninstall Istio
istioctl uninstall --purge -y

# Delete Istio namespace
kubectl delete namespace istio-system
```

## References

-   [Istio Documentation](https://istio.io/latest/docs/)
-   [Istio Best Practices](https://istio.io/latest/docs/ops/best-practices/)
-   [Envoy Proxy](https://www.envoyproxy.io/docs/envoy/latest/)
-   [Service Mesh Patterns](https://www.manning.com/books/istio-in-action)

---

**Last Updated:** 2025-11-06
**Istio Version:** 1.20+
**Phase:** 3 - Enterprise Ready
