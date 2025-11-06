#!/bin/bash

# Istio Service Mesh Installation Script
# This script installs and configures Istio for the Ride Hailing platform

set -e

echo "🚀 Installing Istio Service Mesh..."
echo ""

# Check if istioctl is installed
if ! command -v istioctl &> /dev/null; then
    echo "📥 Downloading Istio..."
    curl -L https://istio.io/downloadIstio | sh -

    # Get latest Istio version
    ISTIO_VERSION=$(ls -d istio-* | tail -1)
    cd $ISTIO_VERSION
    export PATH=$PWD/bin:$PATH
    cd ..

    echo "✅ Istio downloaded: $ISTIO_VERSION"
else
    echo "✅ Istio already installed: $(istioctl version --short)"
fi

echo ""
echo "📦 Installing Istio with production profile..."

# Install Istio with demo profile (use production for real deployments)
istioctl install --set profile=production -y

echo ""
echo "🏷️  Labeling namespace for automatic sidecar injection..."

# Label namespace for automatic sidecar injection
kubectl label namespace ridehailing istio-injection=enabled --overwrite

echo ""
echo "📊 Installing Istio addons (Kiali, Prometheus, Grafana, Jaeger)..."

# Install observability addons
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/prometheus.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/grafana.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/jaeger.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/kiali.yaml

echo ""
echo "⏳ Waiting for Istio components to be ready..."

kubectl wait --for=condition=available --timeout=300s \
  deployment/istiod -n istio-system

kubectl wait --for=condition=available --timeout=300s \
  deployment/kiali -n istio-system || echo "Kiali not yet ready"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "✅ Istio installation complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📌 Access Points:"
echo "   • Kiali Dashboard:    http://localhost:20001"
echo "   • Grafana:            http://localhost:3000"
echo "   • Jaeger:             http://localhost:16686"
echo "   • Prometheus:         http://localhost:9090"
echo ""
echo "🔍 To access dashboards, run:"
echo "   istioctl dashboard kiali"
echo "   istioctl dashboard grafana"
echo "   istioctl dashboard jaeger"
echo ""
echo "📝 Next Steps:"
echo "   1. Apply gateway and virtual service configurations"
echo "   2. Restart all pods to inject Istio sidecars:"
echo "      kubectl rollout restart deployment -n ridehailing"
echo "   3. Verify sidecars are running:"
echo "      kubectl get pods -n ridehailing"
echo "      (Should show 2/2 READY for each pod)"
echo ""
