#!/bin/bash
set -e

echo "🧪 Knative Test Script"
echo "======================="

# Minimal Knative Test direkt mit kubectl
cat > test-knative.yaml <<EOF
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: simple-test
spec:
  template:
    spec:
      containers:
      - image: nginx:alpine
EOF

echo "📦 Deploye Test Service..."
kubectl apply -f test-knative.yaml

echo "⏳ Warte auf Service..."
sleep 30

echo ""
echo "📊 Service Status:"
kubectl get ksvc simple-test

echo ""
echo "🔍 Service Details:"
kubectl describe ksvc simple-test

echo ""
echo "🌐 Service URL:"
kubectl get ksvc simple-test -o jsonpath='{.status.url}'
echo ""

# Aufräumen
echo "🧹 Räume auf..."
kubectl delete -f test-knative.yaml
rm test-knative.yaml

echo "✅ Test abgeschlossen"