#!/usr/bin/env bash
# Löscht den lokalen kind-Cluster.
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-maxcloud-dev}"

echo "==> kind-Cluster '$CLUSTER_NAME' löschen..."
kind delete cluster --name "$CLUSTER_NAME"
echo "    Fertig."
