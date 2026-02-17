# Local Development

## Prerequisites

- Docker
- Go 1.22+
- Node 20+
- Access to a Kubernetes cluster (kind, k3s, kubeadm, etc.)

## Run backend services locally

Each service is standalone:

```bash
cd app/deployments-service && go run .
cd app/events-service && go run .
cd app/notifications-service && go run .
cd app/gateway && go run .
```

Set env vars as needed:

- `DATABASE_URL`
- `REDIS_ADDR`
- `WEBHOOK_URLS`
- `DEPLOYMENTS_SERVICE_URL`

## Run frontend

```bash
cd app/frontend
npm install
npm run dev
```

Environment variables:

- `VITE_API_BASE` (default `http://localhost:8080`)
- `VITE_WS_URL` (default `ws://localhost:8082/ws`)
