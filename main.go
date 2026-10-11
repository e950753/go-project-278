package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/e950753/go-project-278/internal/db"
)

func setupRouter(app *App) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	api := router.Group("/api")
	api.GET("/links", app.listLinks)
	api.POST("/links", app.createLink)
	api.GET("/links/:id", app.getLink)
	api.PUT("/links/:id", app.updateLink)
	api.DELETE("/links/:id", app.deleteLink)

	return router
}

func main() {
	_ = godotenv.Load() // локально читает .env, на Render файла нет, это нормально

	if err := sentry.Init(sentry.ClientOptions{Dsn: os.Getenv("SENTRY_DSN")}); err != nil {
		log.Printf("sentry init: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	app := &App{
		q:       db.New(pool),
		baseURL: os.Getenv("BASE_URL"),
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := setupRouter(app).Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
