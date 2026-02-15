#!/bin/bash
set -e

echo "🚀 max-cloud Hetzner Production Setup"
echo "======================================"

# Farben für Output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Variablen
HETZNER_API_TOKEN="${HETZNER_API_TOKEN:?HETZNER_API_TOKEN must be set}"
DNS_ZONE="${DNS_ZONE:-maxcloud.dev}"
CLUSTER_NAME="${CLUSTER_NAME:-maxcloud}"
K8S_VERSION="${K8S_VERSION:-v1.31.0+k3s1}"
K3S_VERSION="${K3S_VERSION:-v1.31.4+k3s2}"

# Prüfe Tools
command -v hcloud >/dev/null 2>&1 || { echo -e "${RED}hcloud CLI nicht installiert${NC}"; exit 1; }

echo -e "${GREEN}✓ Hetzner CLI installiert${NC}"

# Hetzner CLI konfigurieren
hcloud context create default --token "$HETZNER_API_TOKEN" 2>/dev/null || true

echo "📦 Erstelle Netzwerk..."
hcloud network create --name "$CLUSTER_NAME-network" --ip-range "10.0.0.0/16" 2>/dev/null || NETWORK_ID=$(hcloud network list -o json | jq -r '.[] | select(.name=="maxcloud-network") | .id')

hcloud network add-subnet "$CLUSTER_NAME-network" --network-zone eu-central --type cloud 2>/dev/null || true

echo "📦 Erstelle Firewall..."
hcloud firewall create --name "$CLUSTER_NAME-firewall" 2>/dev/null || FIREWALL_ID=$(hcloud firewall list -o json | jq -r '.[] | select(.name=="maxcloud-firewall") | .id')

hcloud firewall add-rule "$CLUSTER_NAME-firewall" \
  --direction in \
  --protocol tcp \
  --port 22,80,443,6443,30000-32767 \
  --source-ips "0.0.0.0/0" \
  2>/dev/null || true

echo "📦 Erstelle Server..."

# Control Plane
hcloud server create \
  --name "$CLUSTER_NAME-control" \
  --type cpx21 \
  --image ubuntu-22.04 \
  --location nbg1 \
  --network "$CLUSTER_NAME-network" \
  --firewall "$CLUSTER_NAME-firewall" \
  --ssh-key default \
  --label "node-role=control-plane" \
  --label "node-type=master"

# Worker Nodes
for i in 1 2; do
  hcloud server create \
    --name "$CLUSTER_NAME-worker-$i" \
    --type cpx31 \
    --image ubuntu-22.04 \
    --location nbg1 \
    --network "$CLUSTER_NAME-network" \
    --firewall "$CLUSTER_NAME-firewall" \
    --ssh-key default \
    --label "node-role=worker" \
    --label "node-type=worker"
done

echo "⏳ Warte auf Server..."
sleep 30

# DNS Zone erstellen
echo "🌐 Erstelle DNS Zone..."
hcloud dns zone create --name "$DNS_ZONE" --ttl 300 2>/dev/null || true

echo "📜 Installiere k3s auf Control Plane..."
CONTROL_IP=$(hcloud server ip "$CLUSTER_NAME-control")

# k3s Installation auf Control Plane
ssh -o StrictHostKeyChecking=no "root@$CONTROL_IP" << 'EOF'
  curl -sfL https://get.k3s.io | K3S_KUBECONFIG_MODE="644" INSTALL_K3S_VERSION=v1.31.4+k3s2 sh -
  sleep 10
  cat /etc/rancher/k3s/k3s.yaml
EOF

# Token für Worker holen
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

K3S_TOKEN=$(ssh $SSH_OPTS "root@$CONTROL_IP" 'cat /var/lib/rancher/k3s/server/node-token')

# Worker Nodes installieren
for i in 1 2; do
  WORKER_IP=$(hcloud server ip "$CLUSTER_NAME-worker-$i")
  echo "📜 Installiere k3s auf Worker $i..."
  ssh $SSH_OPTS "root@$WORKER_IP" << EOF
    curl -sfL https://get.k3s.io | K3S_URL=https://$CONTROL_IP:6443 K3S_TOKEN=$K3S_TOKEN INSTALL_K3S_EXEC="--disable=traefik" sh -
EOF
done

# kubeconfig holen
echo "🔑 Hole kubeconfig..."
ssh $SSH_OPTS "root@$CONTROL_IP" 'cat /etc/rancher/k3s/k3s.yaml' > kubeconfig

# kubeconfig anpassen (IP ersetzen)
sed -i "s|127.0.0.1|$CONTROL_IP|g" kubeconfig

echo "✅ Kubernetes Cluster erstellt!"
echo ""
echo "📋 Nächste Schritte:"
echo "1. kubeconfig: ./kubeconfig"
echo "2. Export: export KUBECONFIG=./kubeconfig"
echo "3. Knative installieren:"
echo "   kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.14.0/serving-core.yaml"
echo "   kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.14.0/kourier.yaml"
echo "4. ExternalDNS installieren:"
echo "   kubectl apply -f deploy/external-dns.yaml"
echo ""
echo "🌐 DNS Zone: $DNS_ZONE"
echo "📍 Control Plane: $CONTROL_IP"