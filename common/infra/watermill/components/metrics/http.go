package metrics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// CreateRegistryAndServeHTTP establishes an HTTP server that exposes the /metrics endpoint for Prometheus at the given address.
// It returns a new prometheus registry (to register the metrics on) and a canceling function that ends the server.
func CreateRegistryAndServeHTTP(addr string) (registry *prometheus.Registry, cancel func()) {
	registry = prometheus.NewRegistry()
	return registry, ServeHTTP(addr, registry)
}

// ServeHTTP establishes an HTTP server that exposes the /metrics endpoint for Prometheus at the given address.
// It takes an existing Prometheus registry and returns a canceling function that ends the server.
func ServeHTTP(addr string, registry *prometheus.Registry) (cancel func()) {
	router := gin.New()

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	router.GET("/metrics", gin.WrapF(handler.ServeHTTP))
	server := http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()

	return func() { _ = server.Close() }
}
