package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	shutdown, err := initTracer()
	if err != nil {
		logger.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer shutdown(context.Background())

	port := env("PORT", "8080")
	deploymentsURL := env("DEPLOYMENTS_SERVICE_URL", "http://deployments-service:8081")

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("gateway"))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "gateway"})
	})
	r.GET("/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	client := &http.Client{Timeout: 5 * time.Second}
	r.Any("/api/deployments", func(c *gin.Context) {
		proxy(c, client, deploymentsURL+"/deployments")
	})
	r.Any("/api/deployments/:id", func(c *gin.Context) {
		proxy(c, client, deploymentsURL+"/deployments/"+c.Param("id"))
	})

	logger.Info("gateway starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Error("server crashed", "error", err)
	}
}

func proxy(c *gin.Context, client *http.Client, target string) {
	req, err := http.NewRequestWithContext(c, c.Request.Method, target, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request creation failed"})
		return
	}
	req.Header = c.Request.Header.Clone()
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
