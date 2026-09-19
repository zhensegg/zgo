package configs

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/zhensegg/zgo/errs"
)

type JWTConfig struct {
	Secret string
	TTL    time.Duration
	Issuer string
}

type JWT struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

type Claims struct {
	jwt.RegisteredClaims
	Data map[string]any `json:"data,omitempty"`
}

func NewJWT(cfg JWTConfig) (*JWT, error) {
	if cfg.Secret == "" {
		return nil, errors.New("configs: JWT secret is required")
	}
	j := &JWT{
		secret: []byte(cfg.Secret),
		ttl:    cfg.TTL,
		issuer: cfg.Issuer,
	}
	if j.ttl <= 0 {
		j.ttl = 24 * time.Hour
	}
	return j, nil
}

func (j *JWT) Sign(subject string, data map[string]any) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
		Data: data,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

func (j *JWT) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	var opts []jwt.ParserOption
	if j.issuer != "" {
		opts = append(opts, jwt.WithIssuer(j.issuer))
	}
	tok, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("configs: unexpected signing method")
		}
		return j.secret, nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("configs: invalid token")
	}
	return claims, nil
}

func (j *JWT) Protect() fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := strings.TrimPrefix(c.Req().Get(fiber.HeaderAuthorization), "Bearer ")
		claims, err := j.Parse(raw)
		if err != nil {
			return errs.Unauthorized("invalid or missing token")
		}
		c.Locals("jwt.claims", claims)
		return c.Next()
	}
}

func ClaimsOf(c fiber.Ctx) *Claims {
	claims, _ := c.Locals("jwt.claims").(*Claims)
	return claims
}
