package obs

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type Metrics struct {
	Registry *prometheus.Registry
	Requests *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

func New() *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),
		Requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zgo_http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"route", "method", "status"},
		),
		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "zgo_http_request_duration_seconds",
				Help:    "Request handling time in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "method"},
		),
	}
	m.Registry.MustRegister(m.Requests, m.Duration)
	return m
}

func Handler(reg *prometheus.Registry) fiber.Handler {
	h := fasthttpadaptor.NewFastHTTPHandler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return func(c fiber.Ctx) error {
		h(c.RequestCtx())
		return nil
	}
}

func (m *Metrics) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		route := c.Route().Path
		if route == "" {
			route = "unmatched"
		}
		method := c.Req().Method()
		m.Requests.WithLabelValues(route, method, itoa(c.Response().StatusCode())).Inc()
		m.Duration.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
		return err
	}
}

func itoa(n int) string {
	var b [4]byte
	i := len(b)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
