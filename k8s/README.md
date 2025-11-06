># Kubernetes Deployment Guide

Complete guide for deploying the Ride Hailing Platform on Kubernetes.

## Overview

This Kubernetes deployment includes:
- **12 Microservices** with auto-scaling
- **PostgreSQL** StatefulSet with persistent storage
- **Redis** StatefulSet with persistent storage
- **Kong API Gateway** with rate limiting and JWT
- **Horizontal Pod Autoscaling** (HPA) for all services
- **Ingress** with TLS/SSL support
- **Service Mesh ready** (Istio compatible)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Internet / Clients                     │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  Nginx Ingress       │
              │  (TLS Termination)   │
              └──────────┬───────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌────▼────┐    ┌────▼────┐
    │  Auth   │    │  Rides  │    │   Geo   │
    │  Pod    │    │  Pod    │    │  Pod    │
    │  (3x)   │    │  (3x)   │    │  (3x)   │
    └────┬────┘    └────┬────┘    └────┬────┘
         │              │              │
         │         HPA Scaling         │
         │    (CPU/Memory based)       │
         │              │              │
    ┌────▼──────────────▼──────────────▼────┐
    │        PostgreSQL StatefulSet         │
    │         (Persistent Volume)           │
    └───────────────────────────────────────┘
    ┌───────────────────────────────────────┐
    │          Redis StatefulSet            │
    │         (Persistent Volume)           │
    └───────────────────────────────────────┘
```

## Prerequisites

### 1. Kubernetes Cluster

**Local Development:**
- [Minikube](https://minikube.sigs.k8s.io/) (recommended for testing)
- [Kind](https://kind.sigs.k8s.io/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop) with Kubernetes

**Production:**
- [GKE](https://cloud.google.com/kubernetes-engine) (Google Kubernetes Engine)
- [EKS](https://aws.amazon.com/eks/) (Amazon Elastic Kubernetes Service)
- [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/) (Azure Kubernetes Service)
- [DigitalOcean Kubernetes](https://www.digitalocean.com/products/kubernetes/)

### 2. Required Tools

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install k9s (optional but recommended)
brew install k9s
```

### 3. Container Registry

Build and push Docker images to a registry:

```bash
# Using Docker Hub
docker login

# Build all services
for service in auth rides geo payments notifications realtime mobile admin promos scheduler analytics fraud; do
  docker build -t yourusername/ridehailing-${service}:latest \
    --build-arg SERVICE_NAME=${service} .
  docker push yourusername/ridehailing-${service}:latest
done
```

## Quick Start

### 1. Start Minikube (Local Testing)

```bash
# Start Minikube with enough resources
minikube start --cpus=4 --memory=8192 --disk-size=50g

# Enable addons
minikube addons enable ingress
minikube addons enable metrics-server
minikube addons enable dashboard
```

### 2. Create Namespace

```bash
kubectl apply -f namespace.yaml
```

### 3. Create Secrets

**Update secrets with your values:**

```bash
# Create database credentials
kubectl create secret generic ridehailing-secrets \
  --from-literal=DB_USER=postgres \
  --from-literal=DB_PASSWORD=your-secure-password \
  --from-literal=JWT_SECRET=your-super-secret-jwt-key \
  --from-literal=STRIPE_API_KEY=sk_live_xxx \
  --from-literal=TWILIO_ACCOUNT_SID=ACxxx \
  --from-literal=TWILIO_AUTH_TOKEN=xxx \
  --from-literal=TWILIO_FROM_NUMBER=+1234567890 \
  --from-literal=SMTP_USERNAME=your-email \
  --from-literal=SMTP_PASSWORD=your-password \
  --namespace=ridehailing

# Create Firebase credentials
kubectl create secret generic firebase-credentials \
  --from-file=firebase.json=path/to/firebase.json \
  --namespace=ridehailing
```

Or use the YAML file (not recommended for production):

```bash
kubectl apply -f secrets.yaml
```

### 4. Create ConfigMap

```bash
kubectl apply -f configmap.yaml
```

### 5. Deploy Infrastructure

```bash
# Deploy PostgreSQL
kubectl apply -f postgres.yaml

# Deploy Redis
kubectl apply -f redis.yaml

# Wait for databases to be ready
kubectl wait --for=condition=ready pod -l app=postgres --timeout=300s -n ridehailing
kubectl wait --for=condition=ready pod -l app=redis --timeout=300s -n ridehailing
```

### 6. Run Database Migrations

```bash
# Create a migration job
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  namespace: ridehailing
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: migrate/migrate
        args:
        - "-path=/migrations"
        - "-database=postgres://postgres:postgres@postgres-service:5432/ridehailing?sslmode=disable"
        - "up"
        volumeMounts:
        - name: migrations
          mountPath: /migrations
      volumes:
      - name: migrations
        configMap:
          name: db-migrations
      restartPolicy: Never
  backoffLimit: 4
EOF

# Check migration status
kubectl logs -l job-name=db-migration -n ridehailing
```

### 7. Deploy All Services

```bash
# Deploy Auth service
kubectl apply -f auth-service.yaml

# Deploy all other services
kubectl apply -f rides-service.yaml
kubectl apply -f geo-service.yaml
kubectl apply -f payments-service.yaml
kubectl apply -f notifications-service.yaml
kubectl apply -f realtime-service.yaml
kubectl apply -f mobile-service.yaml
kubectl apply -f admin-service.yaml
kubectl apply -f promos-service.yaml
kubectl apply -f scheduler-service.yaml
kubectl apply -f analytics-service.yaml
kubectl apply -f fraud-service.yaml

# Or deploy all at once
kubectl apply -f .
```

### 8. Deploy Ingress

```bash
kubectl apply -f ingress.yaml
```

### 9. Verify Deployment

```bash
# Check all pods
kubectl get pods -n ridehailing

# Check all services
kubectl get svc -n ridehailing

# Check HPA status
kubectl get hpa -n ridehailing

# Check ingress
kubectl get ingress -n ridehailing
```

## Accessing Services

### Local Development (Minikube)

```bash
# Get Minikube IP
minikube ip

# Add to /etc/hosts
echo "$(minikube ip) api.ridehailing.local admin.ridehailing.local ws.ridehailing.local" | sudo tee -a /etc/hosts

# Access services
curl http://api.ridehailing.local/api/v1/auth/healthz
```

### Production

Services are accessible at:
- **Main API**: https://api.ridehailing.com
- **Admin API**: https://admin.ridehailing.com
- **WebSocket**: wss://ws.ridehailing.com/ws
- **Kong Gateway**: https://gateway.ridehailing.com

## Auto-Scaling Configuration

All services are configured with Horizontal Pod Autoscaler (HPA):

| Service | Min Pods | Max Pods | CPU Target | Memory Target |
|---------|----------|----------|------------|---------------|
| Auth | 3 | 10 | 70% | 80% |
| Rides | 3 | 10 | 70% | 80% |
| Geo | 3 | 15 | 70% | 80% |
| Payments | 2 | 8 | 70% | 80% |
| Notifications | 2 | 6 | 70% | 80% |
| Real-time | 2 | 10 | 70% | 80% |
| Mobile | 3 | 10 | 70% | 80% |
| Admin | 2 | 5 | 70% | 80% |
| Promos | 2 | 6 | 70% | 80% |
| Scheduler | 1 | 3 | 70% | 80% |
| Analytics | 2 | 6 | 70% | 80% |
| Fraud | 2 | 8 | 70% | 80% |

### Monitoring HPA

```bash
# Watch HPA in real-time
kubectl get hpa -n ridehailing -w

# Describe HPA for specific service
kubectl describe hpa auth-service-hpa -n ridehailing
```

### Testing Auto-Scaling

```bash
# Generate load on auth service
kubectl run -it --rm load-generator --image=busybox --restart=Never -- /bin/sh

# Inside the pod
while true; do wget -q -O- http://auth-service:8080/healthz; done

# Watch scaling
kubectl get hpa -n ridehailing -w
```

## Resource Limits

### Service Resource Allocation

| Service | Request CPU | Request Memory | Limit CPU | Limit Memory |
|---------|-------------|----------------|-----------|--------------|
| Auth | 100m | 128Mi | 500m | 512Mi |
| Rides | 300m | 1Gi | 500m | 2Gi |
| Geo | 200m | 512Mi | 400m | 1Gi |
| Payments | 200m | 512Mi | 300m | 1Gi |
| Notifications | 150m | 512Mi | 300m | 1Gi |
| Real-time | 250m | 1Gi | 500m | 2Gi |
| Mobile | 150m | 512Mi | 300m | 1Gi |
| Admin | 128m | 256Mi | 256m | 512Mi |
| Promos | 128m | 256Mi | 256m | 512Mi |
| Scheduler | 100m | 256Mi | 256m | 512Mi |
| Analytics | 256m | 1Gi | 512m | 2Gi |
| Fraud | 256m | 512Mi | 512m | 1Gi |

### Infrastructure Resource Allocation

| Component | Request CPU | Request Memory | Limit CPU | Limit Memory |
|-----------|-------------|----------------|-----------|--------------|
| PostgreSQL | 500m | 512Mi | 2000m | 2Gi |
| Redis | 250m | 256Mi | 1000m | 1Gi |

## Monitoring

### Kubernetes Dashboard

```bash
# Start dashboard (Minikube)
minikube dashboard

# Production - Deploy dashboard
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

# Create admin user
kubectl create serviceaccount dashboard-admin -n kubernetes-dashboard
kubectl create clusterrolebinding dashboard-admin --clusterrole=cluster-admin --serviceaccount=kubernetes-dashboard:dashboard-admin

# Get token
kubectl -n kubernetes-dashboard create token dashboard-admin
```

### Prometheus & Grafana

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace ridehailing \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n ridehailing

# Default credentials: admin / prom-operator
```

### Logs

```bash
# View logs for a specific service
kubectl logs -l app=auth-service -n ridehailing --tail=100 -f

# View logs for all pods
kubectl logs -l tier=backend -n ridehailing --tail=50

# Install Stern for better log viewing
brew install stern

stern -n ridehailing ".*" --tail=20
```

## Production Deployment

### 1. SSL Certificates

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create Let's Encrypt ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@ridehailing.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### 2. Database Backups

```bash
# Create backup CronJob
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: ridehailing
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15-alpine
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -h postgres-service -U postgres ridehailing | \
              gzip > /backups/backup-\$(date +%Y%m%d-%H%M%S).sql.gz
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: ridehailing-secrets
                  key: DB_PASSWORD
            volumeMounts:
            - name: backups
              mountPath: /backups
          restartPolicy: OnFailure
          volumes:
          - name: backups
            persistentVolumeClaim:
              claimName: postgres-backups
EOF
```

### 3. Network Policies

```bash
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ridehailing-network-policy
  namespace: ridehailing
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ridehailing
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: ridehailing
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
EOF
```

### 4. Pod Disruption Budgets

```bash
# Ensure high availability during updates
kubectl apply -f - <<EOF
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: auth-service-pdb
  namespace: ridehailing
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: auth-service
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: rides-service-pdb
  namespace: ridehailing
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: rides-service
EOF
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n ridehailing

# Check events
kubectl get events -n ridehailing --sort-by='.lastTimestamp'

# Check logs
kubectl logs <pod-name> -n ridehailing
```

### Service Not Reachable

```bash
# Check service endpoints
kubectl get endpoints -n ridehailing

# Test from within cluster
kubectl run -it --rm debug --image=alpine --restart=Never -- sh
apk add curl
curl http://auth-service:8080/healthz
```

### Database Connection Issues

```bash
# Check PostgreSQL pod
kubectl logs -l app=postgres -n ridehailing

# Connect to PostgreSQL
kubectl exec -it postgres-0 -n ridehailing -- psql -U postgres -d ridehailing
```

### HPA Not Scaling

```bash
# Check metrics-server
kubectl top nodes
kubectl top pods -n ridehailing

# If metrics-server not working
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

## Cleanup

```bash
# Delete all resources
kubectl delete namespace ridehailing

# Delete persistent volumes
kubectl delete pv --all

# Stop Minikube
minikube stop
minikube delete
```

## Cost Optimization

### 1. Use Spot/Preemptible Instances

```yaml
# Node affinity for spot instances
nodeSelector:
  cloud.google.com/gke-preemptible: "true"
```

### 2. Cluster Autoscaler

```bash
# GKE
gcloud container clusters update ridehailing \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=10

# EKS
eksctl utils enable-cluster-autoscaler \
  --cluster=ridehailing \
  --region=us-east-1
```

### 3. Vertical Pod Autoscaler

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml
```

## Next Steps

- ✅ Kubernetes deployment configured
- ⏭️ Implement Istio service mesh
- ⏭️ Add distributed tracing (Jaeger)
- ⏭️ Implement ML-based features
- ⏭️ Multi-region deployment

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Helm Charts](https://helm.sh/docs/)
- [Kubernetes Patterns](https://www.redhat.com/en/resources/kubernetes-patterns-e-book)

---

**Last Updated:** 2025-11-06
**Kubernetes Version:** 1.28+
**Phase:** 3 - Enterprise Ready
