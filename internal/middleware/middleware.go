package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/blaiseva001-cloud/backend/internal/config"
	"github.com/blaiseva001-cloud/backend/internal/kv"
	"github.com/blaiseva001-cloud/backend/internal/utils"
)

type Middleware struct {
	cfg *config.Config
	kv  kv.KV
}

func New(cfg *config.Config, store kv.KV) *Middleware { return &Middleware{cfg: cfg, kv: store} }

func (m *Middleware) Auth(c fiber.Ctx) error {
	h := c.Get("Authorization")
	raw := strings.TrimPrefix(h, "Bearer ")
	if raw == h || raw == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	cl, err := utils.ParseToken(m.cfg.JWTSecret, raw)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}
	if _, err := m.kv.Get(c.Context(), "blacklist:"+cl.ID); err == nil {
		return c.Status(401).JSON(fiber.Map{"error": "token revoked"})
	}
	c.Locals("user_id", cl.UserID)
	c.Locals("user_role", cl.Role)
	c.Locals("token_jti", cl.ID)
	c.Locals("token_exp", cl.ExpiresAt.Unix())
	return c.Next()
}

func (m *Middleware) Admin(c fiber.Ctx) error {
	if c.Locals("user_role") != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "admin only"})
	}
	return c.Next()
}
