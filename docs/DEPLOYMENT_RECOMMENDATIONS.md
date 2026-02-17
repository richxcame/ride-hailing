# Deployment Recommendations

## Where to Deploy

### Recommended: Managed Kubernetes on a Major Cloud

| Provider | Service | Why it fits |
|----------|---------|-------------|
| **GCP (Google Cloud)** | GKE | Best-in-class Kubernetes, native Istio support (project already has Istio configs in `k8s/istio/`), Cloud SQL for PostGIS, Memorystore for Redis. Codebase already imports `cloud.google.com/go/*` (Secret Manager, Cloud Storage, Pub/Sub). |
| **AWS** | EKS | Largest ecosystem. RDS for PostGIS, ElastiCache for Redis. Codebase already imports `aws-sdk-go-v2`. More operational overhead than GKE. |
| **Azure** | AKS | Competitive with GKE. Good option if your team is already on Azure. |

**Top pick: GCP/GKE** — the codebase already integrates GCP services, GKE has native Istio/Anthos mesh support, and Cloud SQL supports PostGIS natively. GKE Autopilot reduces cluster management overhead.

### For Smaller Scale / MVP

- **GCP Cloud Run** or **AWS ECS Fargate** — the multi-stage `Dockerfile` is already compatible. Deploy each service separately. WebSocket support exists but with connection timeouts.
- **Single VM + Docker Compose** — the existing `docker-compose.yml` is production-shaped. Viable for early traction (<1000 concurrent users). Pair with managed PostgreSQL instead of containerized Postgres.

### Avoid

- **Heroku / Railway / Fly.io** — not suited for 14-service architectures with StatefulSets, WebSockets, and PostGIS.
- **Bare metal** — operational burden without a dedicated infra team.
- **Self-managed Kubernetes** — use a managed offering; k8s operations is a full-time job.

---

## How to Deploy

### Phase 1: Foundation

#### 1. Use Managed Databases

Replace containerized databases (`k8s/postgres.yaml`, `k8s/redis.yaml`) with managed services for production:

| Component | Self-hosted (current) | Production replacement |
|-----------|----------------------|----------------------|
| PostgreSQL+PostGIS | StatefulSet | Cloud SQL (GCP) / RDS (AWS) |
| Redis | StatefulSet | Memorystore (GCP) / ElastiCache (AWS) |
| NATS | StatefulSet | Keep self-managed (no managed NATS), or migrate to Cloud Pub/Sub (already supported in code) |

Benefits: automated backups, failover, patching, connection pooling.

#### 2. Extend Existing CI/CD

The `.github/workflows/docker.yml` pipeline already builds all 14 services and pushes to `ghcr.io`. Add a deployment step:

```
push to main → build → push to registry → deploy to staging
tag v*.*.* → build → push to registry → deploy to production
```

Use `kubectl set image` or a GitOps tool (ArgoCD / Flux) for the deploy step.

#### 3. Apply Kubernetes Manifests

The `k8s/` directory is already well-structured. Deployment order:

1. `namespace.yaml` + `configmap.yaml` + secrets (via cloud secret manager, not base64 `secrets.yaml`)
2. Infrastructure: managed Postgres + Redis + NATS StatefulSet
3. All 14 service deployments
4. `ingress.yaml` with cert-manager for TLS
5. Optional: `k8s/istio/` for service mesh (mTLS, traffic splitting)

#### 4. DNS and TLS

Ingress expects three hosts: `api.ridehailing.com`, `admin.ridehailing.com`, `ws.ridehailing.com`.

- Set DNS A records pointing to the ingress controller's external IP
- cert-manager with Let's Encrypt is already configured in ingress annotations

### Phase 2: Production Hardening

#### 5. Secrets Management

The codebase supports multiple providers (`pkg/secrets/`):

- GCP: `SECRETS_PROVIDER=gcp` with Google Secret Manager
- AWS: `SECRETS_PROVIDER=aws` with AWS Secrets Manager
- Kubernetes: `SECRETS_PROVIDER=kubernetes` with External Secrets Operator

Never store secrets in ConfigMaps or plain-text environment variables.

#### 6. Observability

The monitoring stack (Prometheus, Grafana, Tempo, OTel Collector) is defined in `monitoring/` and `deploy/`.

For production:
- Use managed Prometheus (GCP Managed Prometheus / Amazon Managed Prometheus)
- Keep Grafana and import existing dashboards from `monitoring/grafana/`
- Use Cloud Trace (GCP) / X-Ray (AWS) or keep Tempo with cloud object storage

#### 7. Database Migrations

Run migrations as a Kubernetes Job before deploying new service versions:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: migrate/migrate
        command: ["migrate", "-path", "/migrations", "-database", "$(DATABASE_URL)", "up"]
      restartPolicy: Never
```

---

## Best Practices

### Architecture

1. **Use GitOps (ArgoCD or Flux)** — the `k8s/` directory is already structured for declarative state management.
2. **Separate stateful from stateless** — managed databases outside k8s, stateless services inside k8s.
3. **Tune HPA configs** — service manifests define Horizontal Pod Autoscalers. Adjust thresholds after load testing.
4. **WebSocket sticky sessions** — the realtime service needs session affinity:
   ```yaml
   nginx.ingress.kubernetes.io/affinity: "cookie"
   nginx.ingress.kubernetes.io/session-cookie-name: "ws-affinity"
   ```

### Security

5. **Image security** — Dockerfile already uses non-root user + Alpine base. Trivy scanning runs in CI.
6. **Network policies** — restrict pod-to-pod communication. Only the API gateway should be externally reachable.
7. **Follow the production checklist** in `docs/DEPLOYMENT.md` (lines 106-122): SSL, CORS, rate limiting, circuit breakers, secrets.
8. **Fix CORS wildcard** in `k8s/ingress.yaml` — change `cors-allow-origin: "*"` to actual domains.
9. **Restrict Konga admin access** in `k8s/ingress.yaml` — change whitelist from `0.0.0.0/0` to VPN/office IPs.

### Reliability

10. **Minimum 2 replicas** for critical services (auth, rides, geo, payments, realtime). Increase HPA `minReplicas` from 1 to 2.
11. **Pod Disruption Budgets** — ensure at least 1 pod per service during rolling updates.
12. **Health-check probes** — services expose `/health`. Verify readiness and liveness probes are configured in k8s manifests.
13. **Database connection pooling** — use PgBouncer or cloud provider's connection proxy (Cloud SQL Proxy) when running multiple replicas.

### Cost Optimization

14. **Start small** — a 3-node cluster (e2-standard-4 on GCP / t3.xlarge on AWS) can run all 14 services at low traffic.
15. **Spot/preemptible nodes** for non-critical services (analytics, scheduler, promos). Keep critical services on on-demand nodes.
16. **Right-size databases** — start with the smallest PostGIS-capable instance, scale vertically as needed.

### Deployment Workflow

17. **Release process:**
    ```
    feature branch → PR → CI (lint + test + security scan + build) → merge to main
    main → auto-deploy to staging → manual approval → deploy to production
    tag vX.Y.Z → deploy to production
    ```

18. **Canary / blue-green deployments** — Istio (already configured in `k8s/istio/`) supports traffic splitting natively.

---

## Quick-Start Recommendation

**GKE Autopilot + Cloud SQL (PostGIS) + Memorystore (Redis) + NATS on k8s + ArgoCD**

This gives the least operational overhead while using infrastructure the codebase is already designed for. The existing `k8s/` manifests, `Dockerfile`, and CI pipeline need minimal changes — swap database connection strings to managed endpoints, configure secrets, and deploy.
