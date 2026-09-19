# The zgo-example starter

[github.com/zhensegg/zgo-example](https://github.com/zhensegg/zgo-example) is a
complete users REST API built on zgo — the reference for idiomatic usage and a
starter for new projects. It is a thin `replace` away from your local framework
checkout.

## Run it

```bash
git clone https://github.com/zhensegg/zgo-example && cd zgo-example
docker compose up -d          # PostgreSQL 16
JWT_SECRET=dev go run ./src   # :8080
```

Without `JWT_SECRET` the app still boots in the `development` env with a
fallback secret; outside development a missing secret exits the process.

## Endpoints

| Method | Path | Auth | Result |
|---|---|---|---|
| POST | `/api/v1/auth/register` | — | 201 |
| POST | `/api/v1/auth/login` | — | 200 `{token}` |
| GET | `/api/v1/users` | — | 200 · newest first |
| GET | `/api/v1/users/:id` | — | 200 · 404 |
| PUT | `/api/v1/users/:id` | Bearer | 200 · 404 |
| DELETE | `/api/v1/users/:id` | Bearer | 204 · 404 |

## What it shows

| Piece | Pattern |
|---|---|
| Bootstrap | `zgo.New(zgo.Postgres(&model.User{}), ...).Run()` — declarative root |
| Schema | `User` model with `db` tags; table + partial unique index auto-created |
| Persistence | `zgo.Repo[model.User, uuid.UUID]` behind the `UserRepository` interface |
| Auth | `jwt.Sign` in login, `jwt.Protect()` on mutation routes |
| Errors | `ErrNotFound` sentinel → `404`, `zgo.Err4xx` → structured JSON |
| Soft delete | `deleted_at` column; deleted users vanish from reads, emails reusable |

## A curl session

```bash
curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"frank","email":"frank@example.com","password":"secret123"}'
# 201 {"id":"...","name":"frank",...}

TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"frank@example.com","password":"secret123"}' | jq -r .token)

curl -s localhost:8080/api/v1/users          # -> [ {...} ]
curl -s -X PUT localhost:8080/api/v1/users/<id> \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"admin"}'
curl -s -X DELETE localhost:8080/api/v1/users/<id> -H "Authorization: Bearer $TOKEN"
curl -s localhost:8080/api/v1/users          # -> []
```