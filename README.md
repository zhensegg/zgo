<div align="center">

# zgo

**A Spring Boot-style web framework for Go.**

Configuration · DI · Routing · Generic CRUD · JWT · Metrics

*One config. Your models are the schema.*

</div>

---

## Why zgo

Most Go frameworks leave you to wire everything. zgo doesn't.

- **Your models are the schema.** `db` tags define tables, indexes, defaults
  and soft delete — created at boot. No migration files anywhere.
- **Generic CRUD, no SQL.** `zgo.NewRepo[T, ID]` gives every model a typed
  repository: `Insert`, `ByID`, `FindBy`, `All`, `Update`, `Delete`, `Count`.
- **Spring-like DI.** Components are plain constructor functions; zgo resolves
  the graph and mounts your controllers. Adding an endpoint means adding one
  constructor.
- **Handlers are one-liners.** Write `func(ctx, In) (Out, error)`; the adapter
  binds the body, writes the response and maps errors. No middleware soup.
- **Boring operations, on purpose.** `/healthz`, Prometheus `/metrics`, pprof,
  graceful shutdown and env-first configuration out of the box.

## Killer features

| | |
|---|---|
| **Auto-schema** | tables + partial unique indexes from model tags (`unique` on soft-deleted tables → `WHERE deleted_at IS NULL`) |
| **Generic CRUD** | `Repo[T, ID]` with soft delete, `updatable`-field guards and a single `ErrNotFound` sentinel |
| **Constructor DI** | functions in, wired graph out; controllers auto-discovered and mounted |
| **Typed adapters** | `zgo.JSON / Query / Param / ParamJSON / ParamVoid` turn plain methods into routes |
| **JWT auth** | HS256 sign/parse with typed claims; `Protect()` middleware per route |
| **Env-first config** | `config.yaml` overlaid by `ZGO_*` variables |
| **Metrics + pprof** | per-route Prometheus histograms, `/healthz`, `/debug/pprof` |

## Quick start

The fastest way in is the working starter:

```bash
git clone https://github.com/zhensegg/zgo-example && cd zgo-example
docker compose up -d          # PostgreSQL
JWT_SECRET=dev go run ./src   # REST API on :8080
```

```bash
curl localhost:8080/healthz          # -> {"status":"ok",...}
curl localhost:8080/api/v1/users     # -> []
```

All of it behind a single declarative line — see the [starter](https://github.com/zhensegg/zgo-example)
for the pattern:

```go
zgo.New(zgo.Postgres(&model.User{}), repository.NewUserRepository, configs.JWT,
        service.NewUserService, controller.NewUserController).Run()
```

## Docs

| doc | contents |
|---|---|
| [Getting started](docs/getting-started.md) | config, models, repositories, JWT — with the starter |
| [Architecture](docs/architecture.md) | a one-page map of how it fits |
| [Reference](docs/reference.md) | model tags, repository API, adapters, errors |
| [Example](docs/example.md) | the starter API, endpoints, a curl session |

## Status

MVP. The happy path — a PostgreSQL REST API with JWT auth and soft deletes — is
battle-tested by the example; pagination helpers and richer query builders are
next.

## License

[GPL-3.0](LICENSE).