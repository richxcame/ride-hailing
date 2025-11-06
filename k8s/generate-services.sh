#!/bin/bash

# Script to generate Kubernetes deployment files for all services
# This creates consistent deployments for all 12 microservices

set -e

SERVICES=(
  "rides-service:3:10:500m:2Gi:300m:1Gi"
  "geo-service:3:15:400m:1Gi:200m:512Mi"
  "payments-service:2:8:300m:1Gi:200m:512Mi"
  "notifications-service:2:6:300m:1Gi:150m:512Mi"
  "realtime-service:2:10:500m:2Gi:250m:1Gi"
  "mobile-service:3:10:300m:1Gi:150m:512Mi"
  "admin-service:2:5:256m:512Mi:128m:256Mi"
  "promos-service:2:6:256m:512Mi:128m:256Mi"
  "scheduler-service:1:3:256m:512Mi:100m:256Mi"
  "analytics-service:2:6:512m:2Gi:256m:1Gi"
  "fraud-service:2:8:512m:1Gi:256m:512Mi"
)

for service_config in "${SERVICES[@]}"; do
  IFS=':' read -r service min_replicas max_replicas cpu_limit mem_limit cpu_request mem_request <<< "$service_config"

  cat > "${service}.yaml" <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${service}
  namespace: ridehailing
  labels:
    app: ${service}
    tier: backend
spec:
  replicas: ${min_replicas}
  selector:
    matchLabels:
      app: ${service}
  template:
    metadata:
      labels:
        app: ${service}
        tier: backend
    spec:
      containers:
      - name: ${service}
        image: ridehailing/${service}:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        envFrom:
        - configMapRef:
            name: ridehailing-config
        - secretRef:
            name: ridehailing-secrets
        resources:
          requests:
            memory: "${mem_request}"
            cpu: "${cpu_request}"
          limits:
            memory: "${mem_limit}"
            cpu: "${cpu_limit}"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3

---
apiVersion: v1
kind: Service
metadata:
  name: ${service}
  namespace: ridehailing
  labels:
    app: ${service}
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: ${service}

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ${service}-hpa
  namespace: ridehailing
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ${service}
  minReplicas: ${min_replicas}
  maxReplicas: ${max_replicas}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
      - type: Pods
        value: 2
        periodSeconds: 30
      selectPolicy: Max
EOF

  echo "✅ Generated ${service}.yaml"
done

echo ""
echo "🎉 All service deployments generated!"
