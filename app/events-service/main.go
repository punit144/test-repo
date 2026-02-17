package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := env("PORT", "8082")
	redisAddr := env("REDIS_ADDR", "redis:6379")

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()
	pubsub := redisClient.Subscribe(context.Background(), "deployments.events")
	defer pubsub.Close()

	clients := map[*websocket.Conn]struct{}{}
	mu := sync.Mutex{}

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			mu.Lock()
			for c := range clients {
				_ = c.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			}
			mu.Unlock()
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery(), otelgin.Middleware("events-service"))
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) { c.Status(200) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.Status(500)
			return
		}
		mu.Lock()
		clients[conn] = struct{}{}
		mu.Unlock()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				mu.Lock()
				delete(clients, conn)
				mu.Unlock()
				_ = conn.Close()
				break
			}
		}
	})

	logger.Info("events service starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Error("service crashed", "error", err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
