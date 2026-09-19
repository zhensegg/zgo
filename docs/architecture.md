# Architecture

A one-page map. zgo is a thin composition of four ideas: a config loader,
a constructor-based DI container, a fiber server, and your models as the schema.

## The shape

```text
main: zgo.New(ctors...) .Run()
  │
  ├─ config          config.yaml + ZGO_* env
  ├─ di              constructors → resolved graph, singletons
  ├─ engine          fiber v3 + recovery + error handler + graceful stop
  └─ app             mounts controllers + /healthz /metrics /debug/pprof

your code:  controller → service → repository → zgo.Repo → PostgreSQL
```

Each layer talks to the next through plain interfaces — services depend on a
`UserRepository` interface, not on the database.

## How things happen

- **At boot** `zgo.Postgres(&Model{})` opens the pool and creates tables and
  indexes from `db` tags (`CREATE TABLE IF NOT EXISTS` — safe to re-run).
  There is no migration pipeline; column changes are an explicit decision.
- **The DI container** registers constructors by their return type, resolves
  dependencies recursively, and instantiates each component once. On startup it
  resolves the whole graph and mounts everything that implements `Controller`.
- **On a request** a typed adapter (JSON/Param/...) binds the input, calls your
  method, and turns the result into JSON — or the error into `{"code","message"}`
  with the right status. Errors are `errs.HTTPError` values; everything else
  becomes a generic `500`.

## Design at a glance

| Choice | Why |
|---|---|
| Models as schema | one source of truth, no migration files |
| Generic `Repo[T, ID]` | CRUD and "editable fields" live in the tags, not in each repository |
| Constructor DI | adding a feature = adding a constructor; main stays declarative |
| Typed adapters | binding and error mapping are one-liners per route |
| Soft delete by convention | deleted rows stay auditable, unique keys stay reusable |