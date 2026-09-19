package engine

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/zhensegg/zgo/errs"
)

type Engine struct {
	Fiber *fiber.App
	Addr  string
}

func New(addr string) *Engine {
	f := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})
	f.Use(recovermw.New())
	return &Engine{Fiber: f, Addr: addr}
}

func errorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	code := "internal"
	message := "internal error"
	var fields []errs.FieldError

	var he *errs.HTTPError
	if errors.As(err, &he) && he != nil {
		status = he.Status
		code = he.Code
		message = he.Message
		fields = he.Fields
	}

	body := fiber.Map{"code": code, "message": message}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	return c.Status(status).Res().JSON(body)
}

func (e *Engine) Run(onShutdown ...func()) error {
	go func() {
		if err := e.Fiber.Listen(e.Addr); err != nil {
			log.Printf("zgo: http server stopped: %v", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Printf("zgo: shutdown signal received, graceful shutdown...")

	for _, fn := range onShutdown {
		fn()
	}
	done := make(chan error, 1)
	go func() { done <- e.Fiber.Shutdown() }()
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-time.After(10 * time.Second):
		log.Printf("zgo: shutdown did not complete within 10s")
	}
	log.Printf("zgo: server stopped")
	return nil
}
