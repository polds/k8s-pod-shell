# k8s-pod-shell

Because running `kubectl exec -it` is boring. You can do better.

## What it is

`k8s-pod-shell` is a single-binary web terminal for pods in one configured workload and one namespace.

- Backend: Go (`net/http`, `coder/websocket`, `client-go`)
- Frontend: React + Tailwind + `xterm.js`
- Packaging: Helm chart (`chart/latest`)

## Security model

This app has no built-in authentication/authorization. Deploy it only behind a trusted reverse proxy that enforces auth and TLS.

- Container runs non-root with read-only root filesystem and dropped capabilities
- RBAC is namespace-scoped and workload-limited
- Websocket origin checks can be configured via `ALLOWED_ORIGINS`

## API

- `GET /api/v1/pods`
- `GET /api/v1/info`
- `GET /api/v1/healthz`
- `GET /api/v1/readyz`
- `GET /api/v1/exec?pod=<name>&container=<name>`

## Local development

```bash
make deps
make build
./bin/kubeshell-web
```

Frontend only:

```bash
cd web
npm run dev
```

## Helm install

```bash
helm install kubeshell-web ./chart/latest \
  --namespace demo --create-namespace \
  --set target.kind=Deployment \
  --set target.name=foo
```

## Release strategy

Versioned releases are driven by semver git tags and publish both container and Helm chart artifacts for the same version.

- Tag format: `vX.Y.Z`
- Container tags: `vX.Y.Z`, `X.Y.Z`, and `latest`
- Chart version + appVersion: `X.Y.Z` (packaged at release time)
- Chart push target: `oci://ghcr.io/<owner>/charts`

Detailed runbook: `docs/RELEASING.md`

## Out of scope (v1)

- In-app authn/authz
- Cross-namespace targeting
- Multi-pod broadcast
- Split panes/multi-attach
- File upload/download
- Session recording/audit logs
