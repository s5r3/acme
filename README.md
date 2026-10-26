# Acme — sample Go service

A runnable reference project for the team's Go project convention: one
codebase, two binaries (JSON API + server-rendered web app), organized by
feature. It has two features, `user` and `order`; orders can be created through
the API or the web pages and both go through the same service.

**Stack:** Go 1.26 · Echo v5 · GORM + PostgreSQL · goose (SQL migrations) ·
Viper · slog + lumberjack · templ · go-playground/validator · JWT

## Quick start

Requirements: Go 1.26+, Docker (for PostgreSQL), `make`.

```sh
make db-up          # PostgreSQL 16 in Docker (databases: acme, acme_test)
make migrate-up     # apply migrations
make seed           # demo accounts (see below)
make run-api        # http://localhost:8080   (terminal 1)
make run-web        # http://localhost:8081   (terminal 2)
```

Or run everything in containers: `make up` (then `make down`).

Demo accounts created by `make seed`:

| Email | Password | Role |
| --- | --- | --- |
| admin@example.com | admin12345 | admin — sees and manages every order and user |
| customer@example.com | customer123 | customer — sees only their own orders |

### Try the API

```sh
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/token \
  -H 'content-type: application/json' \
  -d '{"email":"customer@example.com","password":"customer123"}' | jq -r .accessToken)

curl -s -X POST localhost:8080/api/v1/orders -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{"note":"first","items":[{"sku":"BK-1","name":"Go book","quantity":2,"unitPriceCents":2550}]}'

curl -s localhost:8080/api/v1/orders -H "authorization: Bearer $TOKEN"
```

The spec is served at `http://localhost:8080/api/openapi/openapi.yaml`.

### Try the web app

Open http://localhost:8081, log in as the customer, place an order, then list
it through the API above: it is the same order.

## Layout

```
cmd/
  api/ web/ migrate/       one main.go per binary; nothing but flag parsing + app.Run*
internal/
  app/                     composition root: Core (shared), NewAPIServer, NewWebServer, modules.go
  feature/
    user/                  accounts, API tokens, web login/sign-up
    order/                 orders + items; API and web front-ends over one Service
    _template/             copied by `make feature name=x` (ignored by the Go tool)
  platform/
    config/                Viper -> typed, validated Config (only place Viper is used)
    database/              GORM open, TxManager, slog query logger, goose migrator, dbtest/
    logger/                slog JSON + lumberjack; request-scoped logger in context
    httpx/                 app Error type, problem+json error handler, validation, list envelope
    web/                   sessions, CSRF, flash, render, HTML error pages, layout/, assets/
    middleware/            request ID, access log, recover, timeout, auth guards
    auth/                  Principal, role->permission registry, JWT, bcrypt
  shared/                  dependency-free helpers: page, id (UUID v7), clock
migrations/                goose SQL files (embedded into cmd/migrate)
api/openapi/               hand-written OpenAPI 3.1, split per feature (embedded, served)
configs/config.yaml        defaults for every binary; config.local.yaml is git-ignored
test/integration/          both binaries in-process against real PostgreSQL
deploy/                    Dockerfile (one target per binary), docker-compose.yml
docs/adr/                  architecture decision records
```

### Inside a feature (`internal/feature/order`)

| File | Role |
| --- | --- |
| `module.go` | `Deps`, `New`, wiring of repository → service → handlers |
| `service.go` | business rules + input types (`CreateOrderInput`); the only place orders are created |
| `repository.go` | GORM; translates GORM errors into feature errors |
| `model.go` | GORM entities, no json tags |
| `errors.go`, `permissions.go` | feature errors; permission constants + role grants |
| `api_routes.go`, `api_handler*.go`, `dto.go` | JSON front-end |
| `web_routes.go`, `web_handler.go`, `form.go`, `views/` | HTML front-end (templ) |
| `mapper.go` | DTO/form → service input; entity → response/view model |

Dependencies point one way: handler → service → repository → database.

**What a feature exports.** Only what another package must name: `Module`,
`New`, `Deps`, `Service` (other features hold a `*user.Service`), entity types,
errors and permissions. Everything used only inside the feature stays
unexported: the `repository` interface the service declares, its GORM
implementation `gormRepository`, the `newService`/`newGormRepository`
constructors, and the handlers' own service interfaces (`apiService`,
`service`). The service still receives its repository as a parameter (never a
`*gorm.DB`), so unit tests pass a fake and `service.go` never imports GORM. The
import rules are enforced by `depguard` in `.golangci.yml`.

## Configuration

Precedence (lowest → highest): defaults in `config.Load` → `configs/config.yaml`
→ `configs/config.local.yaml` → environment variables `ACME_<PATH>` (dots become
underscores). Examples: `ACME_DATABASE_DSN`, `ACME_AUTH_JWT_SECRET`,
`ACME_API_ADDRESS`, `ACME_LOG_LEVEL`, `ACME_LOG_FILE_PATH=/var/log/acme/api.log`
(enables lumberjack rotation).

Set `ACME_AUTH_JWT_SECRET` and `ACME_WEB_COOKIE_SECURE=true` outside development.

## Development tasks

```sh
make help                       # all targets
make test                       # unit tests (race detector)
make test-integration           # + PostgreSQL tests (uses acme_test, truncates it)
make lint                       # golangci-lint incl. import rules
make generate                   # templ -> *_templ.go (commit the output)
make migrate-new name=add_x     # new timestamped goose migration
make feature name=invoice       # scaffold a feature from the template
make openapi                    # lint + bundle the spec (needs Node)
```

Integration tests include two guard rails worth knowing about:

- `TestEveryProtectedRouteRequiresAuth` calls every API route without a token
  and fails if any non-public route answers anything but 401.
- `TestOpenAPICoversRoutes` fails if a route is missing from the spec or the
  spec documents a route that doesn't exist.

## Adding a feature

1. `make feature name=invoice` and follow the printed steps.
2. Add one line to `internal/app/modules.go`.
3. Write the migration, the OpenAPI files, and tests.

## Notes and simplifications

- The order "products" are free-form lines (SKU, name, price) to keep the
  sample small; a real catalog would be a third feature.
- Multi-tenancy is not modelled; add a tenant scope in repositories if needed.
- `make test-integration` runs packages one at a time (`-p 1`) because they
  share one test database.
