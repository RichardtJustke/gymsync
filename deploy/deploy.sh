#!/bin/bash
# Roda no servidor, dentro de /opt/apps/unika-app: builda as imagens, importa no containerd e aplica no k8s.
set -euo pipefail
cd "$(dirname "$0")/.."

if ! kubectl -n unika get secret unika-app-secrets >/dev/null 2>&1; then
  echo "🔐 Criando segredos"
  kubectl -n unika create secret generic unika-app-secrets \
    --from-literal=DB_PASSWORD="$(openssl rand -hex 24)" \
    --from-literal=JWT_SECRET="$(openssl rand -hex 32)" \
    --from-literal=PAYMENT_WEBHOOK_SECRET="$(openssl rand -hex 24)"
fi

for img in api web; do
  echo "🐳 Build unika-app/$img"
  docker build -q -f deploy/Dockerfile.$img -t unika-app/$img:latest .
  docker save unika-app/$img:latest | sudo ctr -n k8s.io images import - >/dev/null
done

kubectl apply -f deploy/k8s/unika-app.yaml
kubectl -n unika rollout restart deploy/unika-api deploy/unika-web
kubectl -n unika rollout status deploy/unika-postgres --timeout=180s
kubectl -n unika rollout status deploy/unika-api --timeout=180s
kubectl -n unika rollout status deploy/unika-web --timeout=180s
echo "✅ Deploy concluído"
