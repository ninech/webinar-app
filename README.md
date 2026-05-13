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

Full setup guide for running this webinar from scratch. All resources are created via the [nine Cockpit](https://cockpit.nine.ch).

### 1. Create the Registry

Create a registry via the Cockpit:

1. Go to **Kubernetes > Registries > Create Registry**
2. Note the registry URL, username, and password
3. Store them as GitHub Actions secrets (`REGISTRY_URL`, `REGISTRY_USERNAME`, `REGISTRY_PASSWORD`) in the [webinar-app repo settings](https://github.com/ninech/webinar-app/settings/secrets/actions)

### 2. Create the NKE Cluster

Create an NKE cluster manually via the [nine Cockpit](https://cockpit.nine.ch):

1. Go to **Kubernetes > Clusters > Create Cluster**
2. Choose 3 nodes, machine type `nine-standard-2`
3. Attach the registry created in step 1 to the cluster
4. Download the kubeconfig once the cluster is ready

### 3. Add Managed Services to the Cluster

Add the following nine managed services to the NKE cluster via the Cockpit or API:

- **NGINX Ingress** — ingress controller for routing traffic
- **Loki** — log storage
- **Promtail** — ships logs to Loki
- **Metrics Agent** — collects and exposes Prometheus metrics
- **Grafana** — dashboards for logs and metrics
- **ArgoCD** — GitOps continuous deployment

### 4. Configure ArgoCD Application

Create an ArgoCD Application pointing to the helm chart repo:

- **Repo URL**: `https://github.com/ninech/webinar-helm.git`
- **Path**: `.`
- **Target Revision**: `main`
- **Destination Namespace**: `webinar`
- **Sync Policy**: Automated with prune and self-heal

### 5. Create the First Release

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
