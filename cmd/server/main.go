package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/blaiseva001-cloud/backend/internal/config"
	"github.com/blaiseva001-cloud/backend/internal/db"
	"github.com/blaiseva001-cloud/backend/internal/handlers"
	"github.com/blaiseva001-cloud/backend/internal/kv"
	"github.com/blaiseva001-cloud/backend/internal/middleware"
)

func main() {
	cfg := config.Load()
	pg := db.ConnectPostgres(cfg)

	var store kv.KV
	if cfg.RedisURL != "" {
		store = kv.NewRedis(cfg.RedisURL)
		log.Println("[kv] redis")
	} else {
		store = kv.NewMem()
		log.Println("[kv] memory (set REDIS_URL to enable redis)")
	}

	db.Migrate(pg)
	h := handlers.New(cfg, pg, store)
	mw := middleware.New(cfg, store)

	app := fiber.New(fiber.Config{AppName: "bla.link api", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second})
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{Format: "${time} | ${status} ${method} ${path}\n"}))
	app.Use(cors.New(cors.Config{AllowOrigins: cfg.ClientURLs, AllowCredentials: true, AllowHeaders: []string{"Origin", "Content-Type", "Authorization"}}))

	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })

	auth := api.Group("/auth")
	auth.Use(limiter.New(limiter.Config{Max: 60, Expiration: 60 * time.Second}))
	auth.Post("/check-email", h.CheckEmail)
	auth.Get("/username/:username", h.CheckUsername)
	auth.Post("/register", h.Register)
	auth.Post("/verify", h.Verify)
	auth.Post("/resend", h.Resend)
	auth.Post("/login", h.Login)
	auth.Post("/logout", mw.Auth, h.Logout)
	auth.Get("/me", mw.Auth, h.Me)

	me := api.Group("/me", mw.Auth)
	me.Get("/site", h.GetSite)
	me.Put("/site", h.UpdateSite)
	me.Post("/links", h.AddLink)
	me.Patch("/links/reorder", h.ReorderLinks)
	me.Patch("/links/:id", h.UpdateLink)
	me.Delete("/links/:id", h.DeleteLink)
	me.Get("/analytics", h.MyAnalytics)

	pub := api.Group("/p")
	pub.Get("/:username", h.PublicSite)
	pub.Post("/:username/click/:link", h.PublicClick)

	admin := api.Group("/admin", mw.Auth, mw.Admin)
	admin.Get("/stats", h.AdminStats)
	admin.Get("/users", h.AdminUsers)
	admin.Patch("/users/:id/role", h.AdminSetRole)
	admin.Patch("/users/:id/ban", h.AdminSetBan)
	admin.Delete("/users/:id", h.AdminDeleteUser)
	admin.Get("/sites", h.AdminSites)
	admin.Delete("/sites/:id", h.AdminDeleteSite)

	log.Printf("[boot] bla.link api on :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
