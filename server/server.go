package server

import (
	"context"
	"fmt"
	"time"

	"anon-bestdori-database/data"
	"anon-bestdori-database/database"
	"anon-bestdori-database/pkg/log"
	"anon-bestdori-database/version"

	"github.com/gofiber/fiber/v2"
)

type Server struct {
	app      *fiber.App
	database *database.Database
	updater  *data.DataUpdater
}

func loggerMiddleware(c *fiber.Ctx) error {
	start := time.Now()
	log.Infof("REQ START %s %s", c.Method(), c.Path())

	err := c.Next()

	status := c.Response().StatusCode()
	latency := time.Since(start)
	log.Infof("REQ END %s %s %d %.3fms", c.Method(), c.Path(), status, latency.Seconds()*1000)

	return err
}

func corsMiddleware(c *fiber.Ctx) error {
	c.Response().Header.Set("Access-Control-Allow-Origin", "*")
	c.Response().Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Response().Header.Set("Access-Control-Allow-Headers", "*")
	c.Response().Header.Set("Server-Version", "Anon-Database/"+version.Version)
	if c.Method() == "OPTIONS" {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return c.Next()
}

func New(db *database.Database, updater *data.DataUpdater) *Server {
	app := fiber.New(fiber.Config{
		ServerHeader: "anon-bestdori-database",
		AppName:      "Anon Bestdori Database",
	})

	// 中间件
	app.Use(loggerMiddleware)
	app.Use(corsMiddleware)

	// 路由注册
	registerPostsRouter(app.Group("/posts"), db)
	registerSongsRoutes(app.Group("/songs"), db)
	registerChartsRoutes(app.Group("/charts"), db)
	registerAssetsRoutes(app.Group("/assets"), db)

	s := &Server{
		app:      app,
		database: db,
		updater:  updater,
	}

	return s
}

func (s *Server) UpdateSongByID(id int) (bool, error) {
	if s.updater == nil {
		return false, fmt.Errorf("data updater not configured")
	}
	return s.updater.UpdateSongByID(id)
}

func (s *Server) UpdatePostByID(id int) (bool, error) {
	if s.updater == nil {
		return false, fmt.Errorf("data updater not configured")
	}
	return s.updater.UpdatePostByID(id)
}

func (s *Server) Start(ctx context.Context, addr string) error {
	go func() {
		<-ctx.Done()
		_ = s.app.Shutdown()
	}()
	return s.app.Listen(addr)
}
