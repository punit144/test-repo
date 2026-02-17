# Service Interaction Flow

1. A user creates/updates a deployment from the frontend.
2. Frontend calls `gateway /api/deployments`.
3. Gateway proxies request to deployments-service.
4. Deployments-service persists data in PostgreSQL.
5. Deployments-service publishes event payload to Redis channel `deployments.events`.
6. events-service consumes Redis pub/sub and broadcasts to active WebSocket clients.
7. notifications-service consumes the same events and pushes outbound webhooks.
8. Rollout and service metrics are scraped by Prometheus and used by Argo Rollouts analysis.
