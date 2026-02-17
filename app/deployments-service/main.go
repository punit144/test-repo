package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Deployment struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name" binding:"required"`
	Version   string    `json:"version" binding:"required"`
	Status    string    `json:"status" binding:"required"`
	CreatedAt time.Time `json:"createdAt"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	shutdown, err := initTracer()
	if err != nil {
		logger.Error("tracer init failed", "error", err)
		os.Exit(1)
	}
	defer shutdown(context.Background())

	ctx := context.Background()
	db, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/deployments?sslmode=disable"))
	if err != nil {
		logger.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})
	defer redisClient.Close()

	_, err = db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS deployments (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("deployments-service"))

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) { c.Status(200) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/deployments", func(c *gin.Context) {
		rows, err := db.Query(c, "SELECT id, name, version, status, created_at FROM deployments ORDER BY id DESC")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		items := []Deployment{}
		for rows.Next() {
			var d Deployment
			_ = rows.Scan(&d.ID, &d.Name, &d.Version, &d.Status, &d.CreatedAt)
			items = append(items, d)
		}
		c.JSON(200, items)
	})

	r.POST("/deployments", func(c *gin.Context) {
		var req Deployment
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var d Deployment
		err := db.QueryRow(c, "INSERT INTO deployments(name, version, status) VALUES($1,$2,$3) RETURNING id,name,version,status,created_at", req.Name, req.Version, req.Status).Scan(&d.ID, &d.Name, &d.Version, &d.Status, &d.CreatedAt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		publish(c, redisClient, d)
		c.JSON(http.StatusCreated, d)
	})

	r.PUT("/deployments/:id", func(c *gin.Context) {
		var req Deployment
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		var d Deployment
		err := db.QueryRow(c, "UPDATE deployments SET name=$1, version=$2, status=$3 WHERE id=$4 RETURNING id,name,version,status,created_at", req.Name, req.Version, req.Status, c.Param("id")).Scan(&d.ID, &d.Name, &d.Version, &d.Status, &d.CreatedAt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		publish(c, redisClient, d)
		c.JSON(200, d)
	})

	r.DELETE("/deployments/:id", func(c *gin.Context) {
		_, err := db.Exec(c, "DELETE FROM deployments WHERE id=$1", c.Param("id"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})

	port := env("PORT", "8081")
	logger.Info("deployments service starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		logger.Error("server stopped", "error", err)
	}
}

func publish(ctx context.Context, client *redis.Client, d Deployment) {
	payload, _ := json.Marshal(d)
	_ = client.Publish(ctx, "deployments.events", payload).Err()
}

func initTracer() (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
