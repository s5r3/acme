# 2. Separate API and web binaries over one codebase

Date: 2026-10-20 · Status: accepted

## Context

Orders must be creatable both through the JSON API and through server-rendered
pages, with identical behaviour.

## Decision

Build two binaries from the same commit: `cmd/api` (JSON) and `cmd/web`
(HTML). Both use the same feature services through `internal/app.Core`. Each
feature exposes `RegisterAPI` and/or `RegisterWeb`; each binary mounts only
its own. Services accept their own input types, never a DTO or form.

Downtime was not the reason: with several replicas, rolling updates and
graceful shutdown, either option deploys without downtime. The reasons are
independent release cadence, different middleware stacks (tokens/CORS/JSON
versus sessions/CSRF/HTML), independent scaling and a smaller attack surface.

## Consequences

- One more deployable and pipeline job.
- The two binaries can briefly run different versions, so every migration must
  be backward compatible with the previous release (expand → migrate → contract).
- A single-process mode is still possible (a `cmd/all` running both servers).
