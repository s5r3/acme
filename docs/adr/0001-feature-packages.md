# 1. Organize code by feature, one package per feature

Date: 2026-10-20 · Status: accepted

## Context

The API will grow to dozens of resource families with dozens of routes each,
worked on by a medium-sized team in parallel. Layer-first layouts
(`handlers/`, `services/`, `repositories/`) spread one change across many
directories and make every pull request touch shared folders.

## Decision

Each resource family is one Go package under `internal/feature/<name>` that
owns its handlers, service, repository, models, DTOs, forms and views, with a
fixed set of file names. Layers are separated by file and type, not by
sub-package. Dependencies point one way (handler → service → repository) and
are enforced with depguard. Features talk to each other only through small
interfaces declared by the consumer (e.g. `order.UserLookup`).

## Consequences

- A change to one feature touches one directory; merge conflicts are rare.
- No import cycles between layers; no `orderhandler.OrderHandler` stutter.
- Features that need each other in both directions signal a wrong boundary.
