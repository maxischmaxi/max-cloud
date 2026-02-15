#!/bin/bash
set -e

echo "🔧 max-cloud Setup mit Kubernetes 1.35 Fix"
echo "=========================================="

# Prüfe Knative-Version für Kubernetes 1.35
# Kubernetes 1.35 benötigt Knative v1.15+
K8S_VERSION=$(kubectl version --short | grep "Server" | cut -d' ' -f3 | cut -d'.' -f2)
if [ "$K8S_VERSION" -ge "35" ]; then
    echo "ℹ️  Kubernetes Version: 1.$K8S_VERSION - benutze Knative v1.15+"
    K_SERVING_URL="https://github.com/knative/serving/releases/download/knative-v1.15.1/serving-core.yaml"
    K_KOURIER_URL="https://github.com/knative/net-kourier/releases/download/knative-v1.15.0/kourier.yaml"
else
    echo "ℹ️  Kubernetes älter als 1.35 - benutze Knative v1.14"
    K_SERVING_URL="https://github.com/knative/serving/releases/download/knative-v1.14.0/serving-core.yaml"
    K_KOURIER_URL="https://github.com/knative/net-kourier/releases/download/knative-v1.14.0/kourier.yaml"
fi

echo "📦 Verwende Knative Serving: $K_SERVING_URL"

# Knative Serving installieren
echo "🔧 Installiere Knative Serving..."
kubectl apply -f "$K_SERVING_URL"

# CRDs separat prüfen
echo "⚙️  Prüfe CRDs..."
kubectl get crd | grep knative.dev || echo "⚠️  CRDs müssen installiert werden..."

# Kourier Ingress installieren
echo "🌐 Installiere Kourier Ingress..."
kubectl apply -f "$K_KOURIER_URL"

echo "⏳ Warte 30 Sekunden für Pods..."
sleep 30

# Prüfe ob Pods laufen
echo "🔍 Prüfe Pod Status:"
kubectl get pods -n knative-serving
kubectl get pods -n kourier-system

# Kourier als Standard-Ingress konfigurieren
echo "🔧 Konfiguriere Kourier als Standard-Ingress..."
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'

echo "✅ Setup abgeschlossen"

# Test-Service deployen
echo "🧪 Deploye Test Service..."
cat > test-service.yaml <<EOF
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello-knative
spec:
  template:
    spec:
      containers:
      - image: gcr.io/knative-samples/helloworld-go
        env:
        - name: TARGET
          value: "max-cloud"
EOF

kubectl apply -f test-service.yaml

echo "⏳ Warte auf Test-Service..."
sleep 60

echo "📊 Service Status:"
kubectl get ksvc hello-knative

echo ""
echo "🌐 Service URL:"
kubectl get ksvc hello-knative -o jsonpath='{.status.url}'
echo ""
echo "✅ Test abgeschlossen"