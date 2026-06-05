package main

import (
	"context"
	"log"
	"os"

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
	port := env("PORT", "8080")
	dbPath := env("DB_PATH", "whiteboard.db")

	ctx := context.Background()
	store, err := storage.NewSQLite(ctx, dbPath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	h := hub.New()
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})
	if err := handler.RegisterRoutes(r, store, h); err != nil {
		log.Fatalf("routes: %v", err)
	}

	log.Printf("listening on :%s", port)
	log.Fatal(r.Run(":" + port))
}
