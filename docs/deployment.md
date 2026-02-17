# Kubernetes Deployment Steps

## 1) Build and publish images

Build each service image from its directory and push to your registry. Update `helm/values.yaml` image references.

## 2) Install prerequisites

Install CRDs/controllers:

- Argo Rollouts
- Istio
- cert-manager
- Prometheus Operator
- Gatekeeper
- Calico (or equivalent policy engine)

## 3) Deploy chart

```bash
helm upgrade --install deployment-event-platform ./helm -n deployment-event-platform --create-namespace
```

## 4) Deploy via Argo CD (optional)

```bash
kubectl apply -f argocd/project.yaml
kubectl apply -f argocd/application.yaml
```

Argo CD app enables automated sync with prune and self-heal.
