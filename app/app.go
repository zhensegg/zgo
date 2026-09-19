package app

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"reflect"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/zhensegg/zgo/config"
	"github.com/zhensegg/zgo/di"
	"github.com/zhensegg/zgo/engine"
	"github.com/zhensegg/zgo/obs"
)

type Handler = fiber.Handler

type Router struct {
	fiber *fiber.App
}

type Controller interface {
	Register(r *Router)
}

type App struct {
	Cfg    *config.Config
	Engine *engine.Engine
	Met    *obs.Metrics
	DI     *di.Container

	shutdown []func()
	prepared bool
}

func New(ctors ...any) *App {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("zgo: failed to load config: %v", err)
	}
	a := &App{
		Cfg:    cfg,
		Engine: engine.New(cfg.Addr()),
		Met:    obs.New(),
		DI:     di.New(),
	}
	a.Engine.Fiber.Use(a.Met.Middleware())

	if err := a.DI.Provide(func() *App { return a }); err != nil {
		log.Fatalf("zgo: Provide(app): %v", err)
	}
	for _, ctor := range ctors {
		a.Provide(ctor)
	}
	return a
}

func (a *App) router() *Router { return &Router{fiber: a.Engine.Fiber} }

func (a *App) AddController(c Controller) *App {
	c.Register(a.router())
	return a
}

func (r *Router) Get(path string, handler Handler, handlers ...Handler) *Router {
	r.fiber.Get(path, handler, toAny(handlers)...)
	return r
}

func (r *Router) Post(path string, handler Handler, handlers ...Handler) *Router {
	r.fiber.Post(path, handler, toAny(handlers)...)
	return r
}

func (r *Router) Put(path string, handler Handler, handlers ...Handler) *Router {
	r.fiber.Put(path, handler, toAny(handlers)...)
	return r
}

func (r *Router) Delete(path string, handler Handler, handlers ...Handler) *Router {
	r.fiber.Delete(path, handler, toAny(handlers)...)
	return r
}

func (a *App) Use(handlers ...Handler) *App {
	args := make([]any, len(handlers))
	for i, h := range handlers {
		args[i] = h
	}
	a.Engine.Fiber.Use(args...)
	return a
}

func (a *App) Get(path string, handler Handler, handlers ...Handler) *App {
	a.Engine.Fiber.Get(path, handler, toAny(handlers)...)
	return a
}

func (a *App) Post(path string, handler Handler, handlers ...Handler) *App {
	a.Engine.Fiber.Post(path, handler, toAny(handlers)...)
	return a
}

func (a *App) Put(path string, handler Handler, handlers ...Handler) *App {
	a.Engine.Fiber.Put(path, handler, toAny(handlers)...)
	return a
}

func (a *App) Delete(path string, handler Handler, handlers ...Handler) *App {
	a.Engine.Fiber.Delete(path, handler, toAny(handlers)...)
	return a
}

func toAny(handlers []Handler) []any {
	args := make([]any, len(handlers))
	for i, h := range handlers {
		args[i] = h
	}
	return args
}

func (a *App) Provide(ctor any) *App {
	if err := a.DI.Provide(ctor); err != nil {
		log.Fatalf("zgo: Provide: %v", err)
	}
	return a
}

func (a *App) OnShutdown(fn func()) *App {
	a.shutdown = append(a.shutdown, fn)
	return a
}

func (a *App) Run() error {
	if err := a.Prepare(); err != nil {
		return err
	}
	log.Printf("zgo: %s running on %s (env=%s)", a.Cfg.App.Name, a.Cfg.Addr(), a.Cfg.App.Env)
	return a.Engine.Run(a.shutdown...)
}

func (a *App) Prepare() error {
	if a.prepared {
		return nil
	}
	if err := a.mountControllers(); err != nil {
		return err
	}
	a.mountSystemRoutes()
	a.prepared = true
	return nil
}

var controllerType = reflect.TypeOf((*Controller)(nil)).Elem()

func (a *App) mountControllers() error {
	values, err := a.DI.All()
	if err != nil {
		return err
	}
	for _, v := range values {
		if v.Type().Implements(controllerType) {
			v.Interface().(Controller).Register(a.router())
		}
	}
	return nil
}

func (a *App) Shutdown() error {
	return a.Engine.Fiber.Shutdown()
}

func (a *App) Test(req *http.Request, cfg ...fiber.TestConfig) (*http.Response, error) {
	if err := a.Prepare(); err != nil {
		return nil, err
	}
	return a.Engine.Fiber.Test(req, cfg...)
}

func (a *App) mountSystemRoutes() {
	f := a.Engine.Fiber

	f.Get("/healthz", func(c fiber.Ctx) error {
		return c.Res().Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "app": a.Cfg.App.Name, "env": a.Cfg.App.Env})
	})

	f.Get("/metrics", obs.Handler(a.Met.Registry))

	pprof := fasthttpadaptor.NewFastHTTPHandler(http.DefaultServeMux)
	f.Get("/debug/pprof/", func(c fiber.Ctx) error {
		pprof(c.RequestCtx())
		return nil
	})
	f.Get("/debug/pprof/*", func(c fiber.Ctx) error {
		pprof(c.RequestCtx())
		return nil
	})
}
