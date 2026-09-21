# Cerebrum

> OpenWolf's learning memory. Updated automatically as the AI learns from interactions.
> Do not edit manually unless correcting an error.
> Last updated: 2026-09-21

## User Preferences

- Cleanup PRs must stay subtractive and single-scoped: re-verify dead code before deleting; do not implement SelectShell, coverage gates, GOALS.md rewrites, or mass-delete `.agents/` scaffolding unless asked.
- Keep `jsdom` (vitest) and `cmd/kubeshell-web/web/dist/index.html` embed placeholder when pruning frontend deps.

## Key Learnings

- **Project:** k8s-pod-shell
- **Description:** Because running `kubectl exec -it` is boring.
- `SelectorFromMap` in `internal/kube/pods.go` was an unused exported helper; removing it also drops the sole `k8s.io/apimachinery/pkg/labels` import in that file.
- `@testing-library/react` was listed in `web/package.json` but never imported under `web/src/**`; vitest + jsdom remain sufficient for web tests.
- Root `package-lock.json` was an orphan lockfile (no root `package.json`); only `web/package-lock.json` is authoritative for the UI.

## Do-Not-Repeat

- [2026-09-21] Do not expand a dead-code cleanup into SelectShell work, coverage gates, GOALS.md edits, or `.agents/` deletion — keep kill-list-only PRs.

## Decision Log

- [2026-09-21] Dead-code cleanup PR limited to: unused `SelectorFromMap`, unused `@testing-library/react` (+ lockfile refresh), orphan root `package-lock.json`.
