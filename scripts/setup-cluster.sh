#!/usr/bin/env bash
# Erstellt einen lokalen kind-Cluster mit Knative Serving + Kourier.
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-maxcloud-dev}"
KNATIVE_VERSION="${KNATIVE_VERSION:-v1.16.0}"

echo "==> kind-Cluster '$CLUSTER_NAME' erstellen..."
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "    Cluster existiert bereits, überspringe."
else
  kind create cluster --name "$CLUSTER_NAME" --wait 60s
fi

echo "==> Knative Serving CRDs installieren..."
kubectl apply -f "https://github.com/knative/serving/releases/download/knative-${KNATIVE_VERSION}/serving-crds.yaml"

echo "==> Knative Serving Core installieren..."
kubectl apply -f "https://github.com/knative/serving/releases/download/knative-${KNATIVE_VERSION}/serving-core.yaml"

echo "==> Kourier (Networking) installieren..."
kubectl apply -f "https://github.com/knative/net-kourier/releases/download/knative-${KNATIVE_VERSION}/kourier.yaml"

echo "==> Knative auf Kourier konfigurieren..."
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'

echo "==> Warten auf Ready-Pods in knative-serving..."
kubectl wait --for=condition=Ready pods --all -n knative-serving --timeout=120s

echo "==> Warten auf Ready-Pods in kourier-system..."
kubectl wait --for=condition=Ready pods --all -n kourier-system --timeout=120s

echo ""
echo "Cluster '$CLUSTER_NAME' ist bereit."
echo "Kubeconfig exportieren:"
echo "  kind export kubeconfig --name=$CLUSTER_NAME"
