package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alimov/whiteboard/internal/hub"
	"github.com/alimov/whiteboard/internal/handler"
	"github.com/alimov/whiteboard/internal/storage"
	"github.com/gin-gonic/gin"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := env("PORT", "80")
	dbPath := env("DB_PATH", "/tmp/whiteboard.db")
	gin.SetMode(gin.DebugMode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store, err := storage.NewSQLite(ctx, dbPath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	h := hub.New()
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "ok %s", dbPath)
	})
	if err := handler.RegisterRoutes(router, store, h); err != nil {
		log.Fatalf("routes: %v", err)
	}

	log.Printf("listening on :%s", port)
	log.Fatal(router.Run(":" + port))
}
