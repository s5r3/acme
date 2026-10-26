# syntax=docker/dockerfile:1
# One Dockerfile, one target per binary:
#   docker build -f deploy/Dockerfile --target api -t acme-api .
#   docker build -f deploy/Dockerfile --target web -t acme-web .
#   docker build -f deploy/Dockerfile --target migrate -t acme-migrate .

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X example.com/acme/internal/app.Version=${VERSION}" \
      -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot AS base
WORKDIR /app
COPY configs/config.yaml /app/configs/config.yaml
USER nonroot:nonroot

FROM base AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api", "-config", "/app/configs/config.yaml"]

FROM base AS web
COPY --from=build /out/web /app/web
EXPOSE 8081
ENTRYPOINT ["/app/web", "-config", "/app/configs/config.yaml"]

FROM base AS migrate
COPY --from=build /out/migrate /app/migrate
ENTRYPOINT ["/app/migrate", "-config", "/app/configs/config.yaml"]
CMD ["up"]
