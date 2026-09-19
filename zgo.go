package zgo

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/zhensegg/zgo/app"
	"github.com/zhensegg/zgo/config"
	"github.com/zhensegg/zgo/configs"
	"github.com/zhensegg/zgo/db"
	"github.com/zhensegg/zgo/errs"
	"github.com/zhensegg/zgo/httpx"
	"github.com/zhensegg/zgo/repo"
	"github.com/zhensegg/zgo/schema"
)

func New(ctors ...any) *app.App {
	return app.New(ctors...)
}

func OpenPostgres(dsn string) (*db.DB, error) { return db.OpenPostgres(dsn) }

func CreateTables(ctx context.Context, d *db.DB, models ...any) error {
	return schema.Create(ctx, d, models...)
}

func Postgres(models ...any) func(a *app.App) (*db.DB, error) {
	return func(a *app.App) (*db.DB, error) {
		pool, err := db.OpenPostgres(a.Cfg.Postgres.URL)
		if err != nil {
			return nil, err
		}
		a.OnShutdown(func() { _ = pool.Close() })
		if err := schema.Create(context.Background(), pool, models...); err != nil {
			_ = pool.Close()
			return nil, err
		}
		return pool, nil
	}
}

type Repo[T any, ID any] = *repo.Repo[T, ID]

var ErrNotFound = repo.ErrNotFound

func NewRepo[T any, ID any](d *db.DB) (Repo[T, ID], error) {
	return repo.New[T, ID](d)
}

func NewJWT(cfg configs.JWTConfig) (*configs.JWT, error) { return configs.NewJWT(cfg) }

func ClaimsOf(c fiber.Ctx) *configs.Claims { return configs.ClaimsOf(c) }

func JSON[In, Out any](svc func(context.Context, In) (Out, error), status int) fiber.Handler {
	return httpx.JSON(svc, status)
}

func Query[Out any](svc func(context.Context) (Out, error), status int) fiber.Handler {
	return httpx.Query(svc, status)
}

func Param[Out any](svc func(context.Context, string) (Out, error), status int) fiber.Handler {
	return httpx.Param(svc, status)
}

func ParamJSON[In, Out any](svc func(context.Context, string, In) (Out, error), status int) fiber.Handler {
	return httpx.ParamJSON(svc, status)
}

func ParamVoid(svc func(context.Context, string) error, status int) fiber.Handler {
	return httpx.ParamVoid(svc, status)
}

type App = app.App

type Router = app.Router

type Handler = app.Handler

type Config = config.Config

type DB = db.DB

type HTTPError = errs.HTTPError

type FieldError = errs.FieldError

type JWT = configs.JWT

type JWTConfig = configs.JWTConfig

type Claims = configs.Claims

func Err400(code, msg string) error { return errs.BadRequest(code, msg) }
func Err401(code, msg string) error { return errs.New(http.StatusUnauthorized, code, msg) }
func Err403(code, msg string) error { return errs.New(http.StatusForbidden, code, msg) }
func Err404(code, msg string) error { return errs.NotFound(code, msg) }
func Err409(code, msg string) error { return errs.Conflict(code, msg) }
