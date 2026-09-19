# Getting started

zgo builds a REST API against PostgreSQL without wiring boilerplate: you write
a model, a service and a controller, the framework does the rest. The fastest
path is the [starter](https://github.com/zhensegg/zgo-example) — clone it and
swap in your own domain.

## Install

```bash
go get github.com/zhensegg/zgo
```

Needs Go 1.26+ and PostgreSQL.

## Config

zgo reads `config.yaml` from the working directory, then `ZGO_*` env vars.

```yaml
app:
  name: myapp
  env: development        # outside development, JWT_SECRET is required
  port: 8080
postgres:
  url: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
```

What you'll actually touch:

| Env | Sets |
|---|---|
| `ZGO_POSTGRES_URL` | database URL — the usual override |
| `ZGO_CONFIG_DIR` | directory of the config file (default: working dir) |
| `ZGO_APP_PORT` · `ZGO_APP_ENV` | port and environment |

## The model

A struct with `db` tags is the struct *and* the table. zgo creates the table at
startup — no migration files.

```go
type User struct {
	ID        uuid.UUID  `db:"id,pk"`
	Name      string     `db:"name,updatable"`
	Email     string     `db:"email,unique"`
	Role      string     `db:"role,default='user'"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func (User) TableName() string { return "users" }
```

What each tag does:

| Tag | Effect |
|---|---|
| `pk` | primary key — exactly one per model |
| `unique` | unique index |
| `default=...` | SQL default, e.g. `now()`, `'user'`, `true` |
| `updatable` | writable by `Repo.Update` |
| `deleted_at *time.Time` | enables soft delete on the table |

Pointers become nullable columns; everything else is `NOT NULL`. Register the
model at boot with `zgo.Postgres(&User{})` — it opens the pool and creates the
schema.

## Repository

Let the generic repository do the CRUD; wrap it in an interface so services
don't care about storage.

```go
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	// ...
}
```

```go
type PostgresUserRepository struct {
	crud zgo.Repo[model.User, uuid.UUID]
}

func NewUserRepository(db *zgo.DB) (UserRepository, error) {
	crud, err := zgo.NewRepo[model.User, uuid.UUID](db)
	return &PostgresUserRepository{crud: crud}, err
}
```

Operations you get out of the box:

| Call | Meaning |
|---|---|
| `crud.Insert(ctx, &u)` | create |
| `crud.ByID(ctx, id)` | get by id |
| `crud.FindBy(ctx, "email", e)` | get by any column |
| `crud.All(ctx, "created_at DESC")` | list, optionally ordered |
| `crud.Update(ctx, &u, "name", "role")` | update only `updatable` fields |
| `crud.Delete(ctx, id)` | soft or hard delete |
| `crud.Count(ctx)` | count |

Missing rows return `zgo.ErrNotFound`. Soft-deleted rows are invisible to every
read automatically. Anything the generic repo can't express (joins, aggregates)
is handwritten here — the only layer that talks SQL.

## Handlers

Write a service method with `func(ctx, Input) (Output, error)`, register a route.
Middlewares like `jwt.Protect()` slot in between.

```go
r.Get("/users", zgo.Query(svc.List, 200))
r.Get("/users/:id", zgo.Param(svc.Get, 200))
r.Post("/users", zgo.JSON(svc.Create, 201))
r.Put("/users/:id", c.jwt.Protect(), zgo.ParamJSON(svc.Update, 200))
r.Delete("/users/:id", c.jwt.Protect(), zgo.ParamVoid(svc.Delete, 204))
```

| Adapter | Service signature | Binds |
|---|---|---|
| `zgo.JSON` | `(ctx, In) (Out, error)` | JSON body |
| `zgo.Query` | `(ctx) (Out, error)` | nothing |
| `zgo.Param` | `(ctx, id) (Out, error)` | `:id` |
| `zgo.ParamJSON` | `(ctx, id, In) (Out, error)` | `:id` + JSON body |
| `zgo.ParamVoid` | `(ctx, id) error` | `:id`, empty response (204) |

Services return DTOs and **return** errors — they never touch HTTP:

```go
return dto.UserResponse{}, zgo.Err409("email_taken", "email is already in use")
```

The framework turns that into `409 {"code":"email_taken","message":"..."}`.
Malformed JSON comes back as `400 invalid_json`, missing body as `400 empty_body`.

## JWT

Use `Sign` to issue, `Protect` on protected routes, `ClaimsOf` in handlers.

```go
j, err := zgo.NewJWT(zgo.JWTConfig{Secret: os.Getenv("JWT_SECRET"), TTL: 24 * time.Hour})
// ...
token, err := j.Sign(userID, map[string]any{"role": "admin"})
```

`NewJWT` is a DI constructor (see the starter). Outside the `development` env,
a missing `JWT_SECRET` aborts startup.

## Run

```go
func main() {
	if err := zgo.New(
		zgo.Postgres(&model.User{}),   // creates the schema
		repository.NewUserRepository,  // (db) → (UserRepository, error)
		service.NewUserService,
		controller.NewUserController,
	).Run(); err != nil {
		log.Fatal(err)
	}
}
```

`Run()` starts the server and stays up until Ctrl-C. Free built-ins:
`GET /healthz`, `GET /metrics`, `GET /debug/pprof/`.

## Next

- [Reference](reference.md) — the cheat sheet.
- [Example](example.md) — the working starter.
- [Architecture](architecture.md) — a one-page map of how it fits.