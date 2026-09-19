# Reference

A cheat sheet for the things you actually use. Deeper layout details live in
[architecture](architecture.md); the runnable patterns are in the
[starter](https://github.com/zhensegg/zgo-example).

## Model tags

```go
type X struct {
	ID   uuid.UUID  `db:"id,pk"`
	Name string     `db:"name,updatable,default='x'"`
	N    *int       `db:"n,index"`  // nullable + indexed
}
```

| Option | Effect |
|---|---|
| `pk` | primary key — exactly one per model |
| `unique` | unique index; partial `WHERE deleted_at IS NULL` on soft-deleted tables |
| `index` | plain index |
| `updatable` | writable through `Repo.Update` |
| `default=...` | SQL default, e.g. `now()`, `'user'`, `true` |
| `-` | skip the field |

- Pointers → nullable columns; everything else → `NOT NULL`.
- A `deleted_at` column enables soft delete on the table.
- Table name: `TableName()` method if present, else `lowercase(type) + "s"`.
- Every exported field needs a `db` tag.

## Repository

```go
crud, err := zgo.NewRepo[User, uuid.UUID](db)
```

| Method | Returns |
|---|---|
| `Insert(ctx, v *T)` | `error` |
| `ByID(ctx, id ID)` | `*T, error` — `ErrNotFound` if missing |
| `FindBy(ctx, column, value)` | `*T, error` — `ErrNotFound` if missing |
| `All(ctx, orderBy ...string)` | `[]T, error` |
| `Update(ctx, v *T, fields ...string)` | `error` — `updatable` fields only |
| `Delete(ctx, id ID)` | `error` — soft if `deleted_at`, else hard |
| `Count(ctx)` | `int, error` |

Soft-deleted rows are hidden from all reads. `zgo.ErrNotFound` is the one
sentinel for "row absent".

## HTTP adapters

| Adapter | Service signature |
|---|---|
| `zgo.JSON` | `func(ctx, In) (Out, error)` |
| `zgo.Query` | `func(ctx) (Out, error)` |
| `zgo.Param` | `func(ctx, id string) (Out, error)` |
| `zgo.ParamJSON` | `func(ctx, id string, In) (Out, error)` |
| `zgo.ParamVoid` | `func(ctx, id string) error` |

All take a status code for success. Errors become:

```json
{ "code": "email_taken", "message": "email is already in use" }
```

with the matching status; unexpected errors → `500 {"code":"internal"}`.

## Errors

```go
zgo.Err400(code, msg)   // 400
zgo.Err401(code, msg)   // 401
zgo.Err403(code, msg)   // 403
zgo.Err404(code, msg)   // 404
zgo.Err409(code, msg)   // 409
```

Return them from services; wrap or compare with `errors.As` when you need to
act on a specific error.

## JWT

```go
j, err := zgo.NewJWT(zgo.JWTConfig{Secret: string, TTL: time.Duration, Issuer: string})
token, err := j.Sign(subject string, data map[string]any)
j.Protect()                    // middleware on a route
claims := zgo.ClaimsOf(c)      // inside a handler
```

HS256; `Secret` is required, default TTL 24h. `Protect` answers `401` on a bad
or missing `Authorization: Bearer` token.

## Config

```yaml
app:
  name: myapp
  env: development
  host: 0.0.0.0
  port: 8080
postgres:
  url: postgres://...?sslmode=disable
```

Overrides via `ZGO_APP_NAME`, `ZGO_APP_ENV`, `ZGO_APP_HOST`, `ZGO_APP_PORT`,
`ZGO_POSTGRES_URL`, `ZGO_POSTGRES_MAX_CONNS`, `ZGO_POSTGRES_MIN_CONNS`.
Config file: `config.yaml` / `config.yml` / `zgo.yaml` / `zgo.yml` in the
working directory or `$ZGO_CONFIG_DIR`.

## Built-in endpoints

| Path | Purpose |
|---|---|
| `/healthz` | liveness: `{"status":"ok",...}` |
| `/metrics` | Prometheus counters/histograms per route |
| `/debug/pprof/` | Go profiling |