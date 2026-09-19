package httpx

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/zhensegg/zgo/errs"
)

func JSON[In, Out any](svc func(context.Context, In) (Out, error), status int) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req In
		if err := bindBody(c, &req); err != nil {
			return err
		}
		out, err := svc(c.Context(), req)
		if err != nil {
			return WriteError(c, err)
		}
		return c.Res().Status(status).JSON(out)
	}
}

func Query[Out any](svc func(context.Context) (Out, error), status int) fiber.Handler {
	return func(c fiber.Ctx) error {
		out, err := svc(c.Context())
		if err != nil {
			return WriteError(c, err)
		}
		return c.Res().Status(status).JSON(out)
	}
}

func Param[Out any](svc func(context.Context, string) (Out, error), status int) fiber.Handler {
	return func(c fiber.Ctx) error {
		out, err := svc(c.Context(), c.Params("id"))
		if err != nil {
			return WriteError(c, err)
		}
		return c.Res().Status(status).JSON(out)
	}
}

func ParamJSON[In, Out any](svc func(context.Context, string, In) (Out, error), status int) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req In
		if err := bindBody(c, &req); err != nil {
			return err
		}
		out, err := svc(c.Context(), c.Params("id"), req)
		if err != nil {
			return WriteError(c, err)
		}
		return c.Res().Status(status).JSON(out)
	}
}

func ParamVoid(svc func(context.Context, string) error, status int) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := svc(c.Context(), c.Params("id")); err != nil {
			return WriteError(c, err)
		}
		return c.Res().Status(status).Send([]byte{})
	}
}

func bindBody(c fiber.Ctx, dst any) error {
	body := c.Req().Body()
	if len(body) == 0 {
		return errs.BadRequest("empty_body", "request body is required")
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return errs.BadRequest("invalid_json", "request body is not valid JSON")
	}
	return nil
}

func WriteError(c fiber.Ctx, err error) error {
	var he *errs.HTTPError
	if errors.As(err, &he) {
		body := fiber.Map{"code": he.Code, "message": he.Message}
		if len(he.Fields) > 0 {
			body["fields"] = he.Fields
		}
		return c.Res().Status(he.Status).JSON(body)
	}
	return c.Res().Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "internal", "message": "internal server error"})
}