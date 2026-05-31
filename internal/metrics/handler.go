package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler creates an isolated Prometheus registry containing only
// OpenSourceBackup metrics (no Go runtime / process metrics) and returns
// an http.Handler for the /metrics endpoint.
//
// Using an isolated registry — not prometheus.DefaultRegisterer — prevents
// leaking Go runtime internals and keeps the output focused on business metrics.
// If Go runtime metrics are needed, add prometheus.NewGoCollector() explicitly.
func NewHandler(stores Stores, log *slog.Logger) http.Handler {
	reg := prometheus.NewRegistry()

	collector := New(stores, log)
	reg.MustRegister(collector)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:            promhttpLogger{log},
		EnableOpenMetrics:   false, // standard Prometheus text format
		DisableCompression:  false,
	})
}

// promhttpLogger adapts slog to the promhttp.Logger interface.
type promhttpLogger struct{ log *slog.Logger }

func (l promhttpLogger) Println(v ...any) {
	l.log.Warn("metrics handler error", "msg", v)
}
