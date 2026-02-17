package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	redisClient := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})
	defer redisClient.Close()
	webhooks := strings.Split(env("WEBHOOK_URLS", ""), ",")

	go func() {
		pubsub := redisClient.Subscribe(context.Background(), "deployments.events")
		ch := pubsub.Channel()
		for msg := range ch {
			for _, url := range webhooks {
				url = strings.TrimSpace(url)
				if url == "" {
					continue
				}
				req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(msg.Payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Argo-Notification", "rollout-event")
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					logger.Error("webhook failed", "url", url, "error", err)
					continue
				}
				_ = resp.Body.Close()
			}
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery(), otelgin.Middleware("notifications-service"))
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) { c.Status(200) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	port := env("PORT", "8083")
	logger.Info("notifications service starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Error("server crashed", "error", err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
