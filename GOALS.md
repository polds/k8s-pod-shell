# GOALS.md

web ui to exec into k8s pods. one helm release targets exactly one workload in one namespace. assumes a trusted reverse proxy in front for tls/authn.

## tagline

Because running `kubectl exec -it` is boring. You can do better.

## scope (v1, must ship)

1. single Go binary serves http + websocket on configurable port (default `:8080`)
2. binary embeds the compiled SPA via `go:embed`
3. SPA lists `Running` pods belonging to one configured target workload
4. user picks a pod from a dropdown; opens an interactive shell session in-page
5. shell streams over a single websocket per session with full pty semantics (resize, signals)
6. responsive UI (desktop + mobile), TailwindCSS
7. mobile control bar with one-tap modifier and shortcut keys
8. Helm chart deploys binary + ServiceAccount + namespace-scoped Role + RoleBinding
9. GitHub Actions CI: lint, test, container build, helm lint, chart push on tag
10. Well documented, beautiful, and intuitive README

## out of scope (v1, do NOT build)

- authn/authz inside the app (delegated to upstream proxy entirely)
- credential forwarding from proxy headers
- cross-namespace targeting
- multi-pod input broadcast
- split-pane / multi-attach
- ephemeral debug pod launcher
- file upload/download in the terminal
- session recording / audit log

## locked tech stack

- backend: Go 1.25+, stdlib `net/http`, `coder/websocket` (formerly nhooyr), `k8s.io/client-go`, in-cluster config (with kubeconfig fallback for local dev)
- frontend: Vite + TypeScript + React 18 + TailwindCSS + `xterm.js` with `xterm-addon-fit` and `xterm-addon-web-links`
- container: distroless or chainguard static base, multi-arch (linux/amd64, linux/arm64)
- chart: Helm 3, `apiVersion: v2`
- CI: GitHub Actions, container to `ghcr.io`, chart pushed as OCI artifact to `ghcr.io`
- `asdf` with `.tool-versions` file for managed tool versions

## target workload model

`values.yaml` declares exactly one of `target.kind = Deployment | StatefulSet | DaemonSet` plus `target.name` and optional `target.namespace` (defaults to release namespace). binary reads these from env vars `TARGET_KIND`, `TARGET_NAME`, `TARGET_NAMESPACE` injected by the deployment template.

pod list resolution:

- `Deployment` → list ReplicaSets owned by deployment, then pods owned by those RSes
- `StatefulSet` / `DaemonSet` → pods directly owned by the workload
- always filter `status.phase == Running` and `status.containerStatuses[*].ready == true`

## rbac (chart-bundled)

namespace-scoped `Role` ONLY. no `ClusterRole`, no `ClusterRoleBinding`. verbs:

| resource | verbs |
| --- | --- |
| `pods` | `get`, `list`, `watch` |
| `pods/exec` | `create`, `get` |
| `replicasets` (apps) | `get`, `list` (only when target.kind == Deployment) |
| target kind (`deployments`/`statefulsets`/`daemonsets`) | `get` (only for the configured kind) |

`RoleBinding` binds the above to a chart-created `ServiceAccount` consumed by the pod.

## shell selection

on session start, attempt shells in this order, picking the first whose `<shell> -c 'true'` exits 0 inside the target container: `bash`, `zsh`, `ash`, `sh`. if all fail, send a server-side error frame and close the websocket with code 1011.

## websocket protocol

endpoint: `GET /api/v1/exec?pod=<name>&container=<name>` (container optional; default = first container)

binary frames both directions. first byte is a type tag:

- client → server
  - `0x00`: raw stdin bytes (rest of frame)
  - `0x01`: JSON control, e.g. `{"type":"resize","cols":80,"rows":24}`
- server → client
  - `0x00`: raw stdout/stderr bytes
  - `0x02`: JSON status, e.g. `{"type":"exit","code":0}` or `{"type":"error","msg":"..."}`

ping/pong every 20s; idle session timeout = 30 min, configurable via `IDLE_TIMEOUT`.

## http endpoints

- `GET /` → SPA (with SPA-style fallback for unknown paths under `/`)
- `GET /api/v1/pods` → `[{name, container[], ready, node, age}]`
- `GET /api/v1/info` → `{target:{kind,name,namespace}, version, gitSha}`
- `GET /api/v1/healthz` → 200 always
- `GET /api/v1/readyz` → 200 once k8s client has done one successful list
- `GET /api/v1/exec` → ws upgrade

## mobile ui requirements

below the terminal, a horizontally scrollable button strip in this exact order:

`Esc`, `Tab`, `Ctrl`, `Alt`, `↑`, `↓`, `←`, `→`, `Ctrl+C`, `Ctrl+D`, `Ctrl+Z`, `Ctrl+L`, `Ctrl+A`, `Ctrl+E`, `Ctrl+U`, `Ctrl+K`, `Ctrl+R`

- `Ctrl` and `Alt` are sticky one-shot modifiers (tap to arm; next non-modifier key combines and disarms)
- all other buttons fire-and-forget
- terminal auto-fits container on viewport resize and orientation change
- minimum supported viewport: 360px wide; no horizontal scroll on the terminal area at that width

## helm chart structure

```tree
chart/
  latest/                # Versioned release directory
    Chart.yaml             # name: kubeshell-web, type: application, apiVersion: v2
    values.yaml
    values.schema.json     # validates target.kind enum and required fields
    templates/
      deployment.yaml
      service.yaml
      serviceaccount.yaml
      role.yaml
      rolebinding.yaml
      ingress.yaml         # rendered only if .Values.ingress.enabled
      _helpers.tpl
      NOTES.txt
```

required `values.yaml` keys:

- `image.repository`, `image.tag` (defaults to `.Chart.AppVersion`), `image.pullPolicy`
- `target.kind`, `target.name`, `target.namespace`
- `service.type`, `service.port`
- `resources`, `nodeSelector`, `tolerations`, `affinity`
- `podSecurityContext`, `securityContext` (default: nonroot, readonly rootfs, drop ALL caps)
- `ingress.enabled`, `ingress.className`, `ingress.annotations`, `ingress.hosts`, `ingress.tls`

`values.schema.json` MUST enforce that `target.kind` ∈ {`Deployment`,`StatefulSet`,`DaemonSet`} and that `target.name` is non-empty.

## ci/cd (github actions)

three workflows under `.github/workflows/`:

1. `pr.yml` (on `pull_request`): `go vet`, `golangci-lint run`, `go test -race -coverprofile=cover.out ./...`, frontend `tsc --noEmit` + `vitest run`, `helm lint chart/`, `helm template chart/ | kubeconform -strict -ignore-missing-schemas`
2. `main.yml` (on push to `main`): all of `pr.yml` plus container build + push to `ghcr.io/<owner>/kubeshell-web` tagged `main` and `sha-<short>`
3. `release.yml` (on tag `v*.*.*`): all of `pr.yml`, then container build + push tagged `<version>` and `latest`, then `helm package chart/` and `helm push` to `oci://ghcr.io/<owner>/charts`, then create a GitHub Release with autogenerated changelog

container builds use `docker/build-push-action` with `buildx` for `linux/amd64,linux/arm64`. images are signed with cosign keyless (sigstore) in `release.yml`.

## testing requirements

- Go: unit tests for pod-list resolution per workload kind, shell-selection fallback, ws frame codec. integration test for the exec path using `kind` (preferred) or `envtest`. table-driven where it fits.
- frontend: vitest component tests for the modifier-key state machine and the fit-on-resize behavior. no e2e required v1.
- helm: `helm unittest` snapshots covering at least: each `target.kind`, ingress on/off, custom resources block.
- coverage gate: ≥ 70% on Go packages excluding `cmd/` and any generated code.

## security posture (document in README)

- the service has NO built-in authn or authz; deploying it without a proxy enforcing authn is a config error
- container runs nonroot, readonly rootfs, all caps dropped
- RBAC is scoped to one namespace and the configured workload's pods
- no command auditing, no session recording in v1
- websocket origin check enforced by default; allowlist configurable via `ALLOWED_ORIGINS` env
- container itself should not have a shell

## acceptance criteria

shippable when ALL of the following hold on a fresh kind cluster:

1. `helm install kubeshell-web ./chart --set target.kind=Deployment --set target.name=foo --namespace demo` succeeds against a `demo` namespace containing a `foo` Deployment with ≥ 1 ready pod
2. browsing the service lists every ready pod of `foo`
3. selecting a pod establishes a working shell in ≤ 2s p50 on cluster-local network
4. typing `exit` closes the ws cleanly; the UI shows a "session ended" state with a Reconnect button
5. UI is fully usable on a 390×844 viewport with no horizontal scroll on the terminal area
6. all three CI workflows pass on a fresh fork without manual intervention beyond setting `GHCR_TOKEN` (or relying on `GITHUB_TOKEN` where sufficient)
7. `helm uninstall` leaves no orphaned RBAC, SA, or workload resources

## stretch goals (v2+, explicitly DO NOT implement now)

- broadcast input to all ready pods of the target workload
- split-pane attach to multiple pods simultaneously
- ephemeral debug pod launcher (`kubectl run --rm -it`-style) with TTL
- asciinema-format session recording export
- session recording / webhook-based audit log of commands executed
- file upload/download in the terminal
