package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/blaiseva001-cloud/backend/internal/config"
	"github.com/blaiseva001-cloud/backend/internal/kv"
	"github.com/blaiseva001-cloud/backend/internal/services"
)

var validate = validator.New()

type H struct {
	Cfg *config.Config
	DB  *gorm.DB
	KV  kv.KV
	OTP *services.OTP
}

func New(cfg *config.Config, db *gorm.DB, store kv.KV) *H {
	return &H{Cfg: cfg, DB: db, KV: store, OTP: services.NewOTP(store, cfg.OTPExpiry, cfg.OTPCooldown)}
}

func fail(c fiber.Ctx, code int, msg string) error {
	return c.Status(code).JSON(fiber.Map{"error": msg})
}

func bind(c fiber.Ctx, v any) error { return c.Bind().JSON(v) }

func userID(c fiber.Ctx) uuid.UUID { return uuid.MustParse(c.Locals("user_id").(string)) }
