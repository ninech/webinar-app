# nine Webinar App

Stateless Go application for demonstrating NKE (Nine Kubernetes Engine) and Deploio capabilities.

Displays pod metadata (name, namespace, IP, worker node, hostname) and exposes a Prometheus `/metrics` endpoint for Grafana dashboards.

## Local Development

```bash
go run main.go
# Open http://localhost:8080
```

## Docker

```bash
docker build -t webinar-app .
docker run -p 8080:8080 webinar-app
```

## CI/CD

On every git tag push (`v*`), GitHub Actions will:
1. Build and push the Docker image to the Nine registry
2. Update the image tag in [ninech/webinar-helm](https://github.com/ninech/webinar-helm) `values.yaml`
3. ArgoCD detects the change and syncs the deployment

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `REGISTRY_URL` | Nine NKE registry URL (e.g. `registry-xxxxx.xxxxx.registry.nineapis.ch`) |
| `REGISTRY_USERNAME` | Registry username |
| `REGISTRY_PASSWORD` | Registry password |
| `PERSONAL_ACCESS_TOKEN` | GitHub PAT with repo write access to `ninech/webinar-helm` |

### Creating a Release

```bash
git tag v1.0.0
git push origin v1.0.0
```

## NKE Cluster Setup

Full setup guide for running this webinar from scratch. All resources are created via the [nine CLI](https://docs.nine.ch/docs/cli) or the [nine Cockpit](https://cockpit.nine.ch).

### 1. Create the Registry

```bash
nctl create registry webinar-registry
```

Note the registry URL, username, and password from the output. Store them as GitHub Actions secrets (`REGISTRY_URL`, `REGISTRY_USERNAME`, `REGISTRY_PASSWORD`) in the [webinar-app repo settings](https://github.com/ninech/webinar-app/settings/secrets/actions).

### 2. Create the NKE Cluster

```bash
nctl create kcluster webinar-cluster \
  --min-nodes 3 \
  --max-nodes 5 \
  --machine-type nine-standard-2
```

Wait for the cluster to be ready:

```bash
nctl get kcluster webinar-cluster
```

Get kubeconfig:

```bash
nctl get credentials webinar-cluster
```

### 3. Attach the Registry to the Cluster

This allows the cluster to pull images from your private registry without extra imagePullSecrets:

```bash
nctl apply kcluster webinar-cluster --registry webinar-registry
```

### 4. Install NGINX Ingress Controller

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer
```

Get the external IP for DNS:

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

### 5. Install Loki (Log Storage)

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install loki grafana/loki \
  --namespace monitoring \
  --create-namespace \
  --set loki.auth_enabled=false \
  --set singleBinary.replicas=1 \
  --set loki.commonConfig.replication_factor=1 \
  --set loki.storage.type=filesystem \
  --set singleBinary.persistence.size=10Gi
```

### 6. Install Promtail (Log Shipping)

```bash
helm install promtail grafana/promtail \
  --namespace monitoring \
  --set config.clients[0].url=http://loki-gateway.monitoring.svc.cluster.local/loki/api/v1/push
```

### 7. Install Prometheus (Metrics Collection)

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/prometheus \
  --namespace monitoring \
  --set server.persistentVolume.size=10Gi \
  --set alertmanager.enabled=false
```

The app pods have annotations for automatic scraping:

```yaml
prometheus.io/scrape: "true"
prometheus.io/port: "8080"
prometheus.io/path: "/metrics"
```

### 8. Install Grafana

```bash
helm install grafana grafana/grafana \
  --namespace monitoring \
  --set persistence.enabled=true \
  --set persistence.size=5Gi \
  --set adminPassword=admin
```

Get the Grafana password:

```bash
kubectl get secret -n monitoring grafana -o jsonpath="{.data.admin-password}" | base64 -d
```

Port-forward to access Grafana:

```bash
kubectl port-forward -n monitoring svc/grafana 3000:80
```

#### Add Data Sources in Grafana

1. **Prometheus**: URL = `http://prometheus-server.monitoring.svc.cluster.local`
2. **Loki**: URL = `http://loki-gateway.monitoring.svc.cluster.local`

### 9. Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Get the ArgoCD admin password:

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

Port-forward to access ArgoCD UI:

```bash
kubectl port-forward -n argocd svc/argocd-server 8443:443
```

### 10. Create the ArgoCD Application

```bash
kubectl apply -f - <<EOF
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: webinar-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/ninech/webinar-helm.git
    targetRevision: main
    path: .
    helm:
      valuesObject:
        image:
          repository: REPLACE_WITH_REGISTRY_URL/webinar-app
        ingress:
          host: webinar.example.com
  destination:
    server: https://kubernetes.default.svc
    namespace: webinar
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
EOF
```

Replace `REPLACE_WITH_REGISTRY_URL` with your actual registry URL and `webinar.example.com` with your domain.

### 11. Create the First Release

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will build the image, push it to the registry, and update the helm chart. ArgoCD will automatically deploy it.

## Endpoints

| Path | Description |
|------|-------------|
| `/` | Main page with pod info |
| `/metrics` | Prometheus metrics |
| `/healthz` | Liveness probe |
| `/readyz` | Readiness probe |
